package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ishaan29/vectorDB/internal/config"
	"github.com/ishaan29/vectorDB/internal/engine"
	"github.com/ishaan29/vectorDB/internal/index"
	"github.com/ishaan29/vectorDB/internal/logger"
	"github.com/ishaan29/vectorDB/pkg/types"
)

// HNSW build parameters currently hardcoded in index.NewHNSWIndex.
// Recorded here so they end up in the result file.
const (
	hnswM              = 16
	hnswEfConstruction = 200
	defaultEfSearch    = 50
)

type LatencyStats struct {
	MeanMs float64 `json:"mean_ms"`
	P50Ms  float64 `json:"p50_ms"`
	P90Ms  float64 `json:"p90_ms"`
	P95Ms  float64 `json:"p95_ms"`
	P99Ms  float64 `json:"p99_ms"`
	MaxMs  float64 `json:"max_ms"`
}

type BuildStats struct {
	TotalSec    float64       `json:"total_sec"`
	VecPerSec   float64       `json:"vec_per_sec"`
	InsertLat   *LatencyStats `json:"insert_latency,omitempty"`
	HeapBytes   uint64        `json:"heap_bytes,omitempty"`
	BytesPerVec float64       `json:"bytes_per_vec,omitempty"`
}

type SearchRun struct {
	K       int          `json:"k"`
	Ef      int          `json:"ef"`
	Workers int          `json:"workers"`
	Recall  float64      `json:"recall"`
	QPS     float64      `json:"qps"`
	Latency LatencyStats `json:"latency"`
}

type LevelResults struct {
	Build      *BuildStats `json:"build,omitempty"`
	Searches   []SearchRun `json:"searches"`
	Concurrent []SearchRun `json:"concurrent,omitempty"`
	RestartSec float64     `json:"restart_sec,omitempty"`
}

// searchFn abstracts index-level vs engine-level search: takes a query and
// k, returns the IDs of the neighbors found.
type searchFn func(q []float32, k int) ([]string, error)

func computeLatency(durs []time.Duration) LatencyStats {
	if len(durs) == 0 {
		return LatencyStats{}
	}
	sorted := make([]time.Duration, len(durs))
	copy(sorted, durs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	pct := func(p float64) float64 {
		idx := int(p/100*float64(len(sorted))) - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		return float64(sorted[idx]) / float64(time.Millisecond)
	}
	var sum time.Duration
	for _, d := range sorted {
		sum += d
	}
	return LatencyStats{
		MeanMs: float64(sum) / float64(len(sorted)) / float64(time.Millisecond),
		P50Ms:  pct(50),
		P90Ms:  pct(90),
		P95Ms:  pct(95),
		P99Ms:  pct(99),
		MaxMs:  float64(sorted[len(sorted)-1]) / float64(time.Millisecond),
	}
}

// benchSearch runs every query once, single-threaded, measuring per-query
// latency and recall against the exact ground truth.
func benchSearch(fn searchFn, ds *Dataset, gt [][]string, k, ef int) (SearchRun, error) {
	durs := make([]time.Duration, 0, len(ds.Queries))
	recallSum := 0.0
	start := time.Now()
	for qi, q := range ds.Queries {
		t0 := time.Now()
		ids, err := fn(q, k)
		durs = append(durs, time.Since(t0))
		if err != nil {
			return SearchRun{}, fmt.Errorf("search failed on query %d: %w", qi, err)
		}
		recallSum += recallAtK(ids, gt[qi], k)
	}
	wall := time.Since(start)
	return SearchRun{
		K:       k,
		Ef:      ef,
		Workers: 1,
		Recall:  recallSum / float64(len(ds.Queries)),
		QPS:     float64(len(ds.Queries)) / wall.Seconds(),
		Latency: computeLatency(durs),
	}, nil
}

// benchConcurrent hammers search from `workers` goroutines for totalOps
// operations to measure aggregate throughput under contention.
func benchConcurrent(fn searchFn, ds *Dataset, gt [][]string, k, ef, workers, totalOps int) (SearchRun, error) {
	if workers < 1 {
		workers = 1
	}
	var opIdx int64 = -1
	var firstErr error
	var errMu sync.Mutex

	workerDurs := make([][]time.Duration, workers)
	workerRecall := make([]float64, workers)
	workerOps := make([]int, workers)

	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			durs := make([]time.Duration, 0, totalOps/workers+1)
			for {
				op := atomic.AddInt64(&opIdx, 1)
				if op >= int64(totalOps) {
					break
				}
				qi := int(op) % len(ds.Queries)
				t0 := time.Now()
				ids, err := fn(ds.Queries[qi], k)
				durs = append(durs, time.Since(t0))
				if err != nil {
					errMu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					errMu.Unlock()
					break
				}
				workerRecall[w] += recallAtK(ids, gt[qi], k)
				workerOps[w]++
			}
			workerDurs[w] = durs
		}(w)
	}
	wg.Wait()
	wall := time.Since(start)
	if firstErr != nil {
		return SearchRun{}, firstErr
	}

	var all []time.Duration
	recallSum := 0.0
	ops := 0
	for w := 0; w < workers; w++ {
		all = append(all, workerDurs[w]...)
		recallSum += workerRecall[w]
		ops += workerOps[w]
	}
	return SearchRun{
		K:       k,
		Ef:      ef,
		Workers: workers,
		Recall:  recallSum / float64(ops),
		QPS:     float64(ops) / wall.Seconds(),
		Latency: computeLatency(all),
	}, nil
}

func heapInUse() uint64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// runIndexBench measures the HNSW index in isolation: pure ANN build and
// search performance with no persistence in the path.
func runIndexBench(p Params, ds *Dataset, gt [][]string) (*LevelResults, error) {
	res := &LevelResults{}

	fmt.Printf("\n--- index-level bench (pure HNSW, no persistence) ---\n")
	idx := index.NewHNSWIndex(p.Dim, nil)

	heapBefore := heapInUse()
	insertDurs := make([]time.Duration, 0, p.N)
	start := time.Now()
	for i, v := range ds.Base {
		t0 := time.Now()
		if err := idx.Add(ds.IDs[i], v); err != nil {
			return nil, fmt.Errorf("index add failed at %d: %w", i, err)
		}
		insertDurs = append(insertDurs, time.Since(t0))
		if (i+1)%10000 == 0 {
			elapsed := time.Since(start)
			fmt.Printf("  built %d/%d vectors (%.0f vec/s)\n",
				i+1, p.N, float64(i+1)/elapsed.Seconds())
		}
	}
	buildTime := time.Since(start)
	heapAfter := heapInUse()

	lat := computeLatency(insertDurs)
	heapDelta := uint64(0)
	if heapAfter > heapBefore {
		heapDelta = heapAfter - heapBefore
	}
	res.Build = &BuildStats{
		TotalSec:    buildTime.Seconds(),
		VecPerSec:   float64(p.N) / buildTime.Seconds(),
		InsertLat:   &lat,
		HeapBytes:   heapDelta,
		BytesPerVec: float64(heapDelta) / float64(p.N),
	}
	fmt.Printf("  build done in %.1fs (%.0f vec/s), heap +%s (%.0f B/vec)\n",
		buildTime.Seconds(), res.Build.VecPerSec, humanBytes(heapDelta), res.Build.BytesPerVec)

	fn := func(q []float32, k int) ([]string, error) {
		rs, err := idx.Search(q, k)
		if err != nil {
			return nil, err
		}
		ids := make([]string, len(rs))
		for i, r := range rs {
			ids[i] = r.ID
		}
		return ids, nil
	}

	// warmup
	for i := 0; i < p.Warmup && i < len(ds.Queries); i++ {
		fn(ds.Queries[i], 10)
	}

	// k sweep at default ef
	for _, k := range p.KList {
		run, err := benchSearch(fn, ds, gt, k, defaultEfSearch)
		if err != nil {
			return nil, err
		}
		res.Searches = append(res.Searches, run)
		fmt.Printf("  k=%-4d ef=%-4d recall=%.3f qps=%-7.0f p50=%.2fms p99=%.2fms\n",
			run.K, run.Ef, run.Recall, run.QPS, run.Latency.P50Ms, run.Latency.P99Ms)
	}

	// ef sweep at k=10 (recall/latency trade-off curve)
	for _, ef := range p.EfList {
		idx.SetSearchEf(ef)
		run, err := benchSearch(fn, ds, gt, 10, ef)
		if err != nil {
			return nil, err
		}
		res.Searches = append(res.Searches, run)
		fmt.Printf("  k=%-4d ef=%-4d recall=%.3f qps=%-7.0f p50=%.2fms p99=%.2fms\n",
			run.K, run.Ef, run.Recall, run.QPS, run.Latency.P50Ms, run.Latency.P99Ms)
	}
	idx.SetSearchEf(defaultEfSearch)

	// concurrent throughput at k=10, default ef
	conc, err := benchConcurrent(fn, ds, gt, 10, defaultEfSearch, p.Workers, p.ConcurrentOps)
	if err != nil {
		return nil, err
	}
	res.Concurrent = append(res.Concurrent, conc)
	fmt.Printf("  concurrent x%d: qps=%.0f p50=%.2fms p99=%.2fms recall=%.3f\n",
		conc.Workers, conc.QPS, conc.Latency.P50Ms, conc.Latency.P99Ms, conc.Recall)

	return res, nil
}

// runEngineBench measures the full engine path: BadgerDB persistence +
// HNSW indexing on ingest, and index search + storage hydration on query.
func runEngineBench(p Params, ds *Dataset, gt [][]string, dataDir string, restart bool) (*LevelResults, error) {
	res := &LevelResults{}

	fmt.Printf("\n--- engine-level bench (HNSW + BadgerDB persistence) ---\n")

	var dir string
	var err error
	if dataDir != "" {
		dir = filepath.Join(dataDir, fmt.Sprintf("bench-%d", time.Now().UnixNano()))
		if err = os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	} else {
		dir, err = os.MkdirTemp("", "vectordb-bench-*")
		if err != nil {
			return nil, err
		}
	}
	defer os.RemoveAll(dir)

	cfg := &config.Config{
		Index:    config.IndexConfig{Type: "hnsw", Dimensions: p.Dim},
		Badger:   config.BadgerConfig{Path: dir},
		Database: config.DatabaseConfig{MaxVectors: p.N * 2},
	}
	log, err := logger.New(&logger.Config{
		Level:       "error",
		Encoding:    "console",
		OutputPaths: []string{"stderr"},
	})
	if err != nil {
		return nil, err
	}

	eng, err := engine.NewEngine(cfg, log)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	if err := eng.Start(ctx); err != nil {
		return nil, err
	}

	// ingest via BatchInsert in chunks
	const chunk = 1000
	start := time.Now()
	for i := 0; i < p.N; i += chunk {
		end := i + chunk
		if end > p.N {
			end = p.N
		}
		batch := make([]types.Vector, 0, end-i)
		for j := i; j < end; j++ {
			batch = append(batch, types.Vector{
				ID:        ds.IDs[j],
				Embedding: ds.Base[j],
				Metadata:  map[string]interface{}{"seq": j},
			})
		}
		if err := eng.BatchInsert(batch); err != nil {
			return nil, fmt.Errorf("batch insert failed at %d: %w", i, err)
		}
		if end%10000 == 0 {
			elapsed := time.Since(start)
			fmt.Printf("  ingested %d/%d vectors (%.0f vec/s)\n",
				end, p.N, float64(end)/elapsed.Seconds())
		}
	}
	ingestTime := time.Since(start)
	res.Build = &BuildStats{
		TotalSec:  ingestTime.Seconds(),
		VecPerSec: float64(p.N) / ingestTime.Seconds(),
	}
	fmt.Printf("  ingest done in %.1fs (%.0f vec/s)\n", ingestTime.Seconds(), res.Build.VecPerSec)

	fn := func(q []float32, k int) ([]string, error) {
		rs, err := eng.Search(types.Vector{Embedding: q}, engine.SearchParams{
			K:         k,
			Threshold: -1, // disable threshold filtering
		})
		if err != nil {
			return nil, err
		}
		ids := make([]string, len(rs))
		for i, r := range rs {
			ids[i] = r.Vector.ID
		}
		return ids, nil
	}

	for i := 0; i < p.Warmup && i < len(ds.Queries); i++ {
		fn(ds.Queries[i], 10)
	}

	for _, k := range p.KList {
		run, err := benchSearch(fn, ds, gt, k, defaultEfSearch)
		if err != nil {
			return nil, err
		}
		res.Searches = append(res.Searches, run)
		fmt.Printf("  k=%-4d ef=%-4d recall=%.3f qps=%-7.0f p50=%.2fms p99=%.2fms\n",
			run.K, run.Ef, run.Recall, run.QPS, run.Latency.P50Ms, run.Latency.P99Ms)
	}

	conc, err := benchConcurrent(fn, ds, gt, 10, defaultEfSearch, p.Workers, p.ConcurrentOps)
	if err != nil {
		return nil, err
	}
	res.Concurrent = append(res.Concurrent, conc)
	fmt.Printf("  concurrent x%d: qps=%.0f p50=%.2fms p99=%.2fms recall=%.3f\n",
		conc.Workers, conc.QPS, conc.Latency.P50Ms, conc.Latency.P99Ms, conc.Recall)

	if restart {
		fmt.Printf("  measuring restart (badger reopen + full HNSW rebuild)...\n")
		if err := eng.Stop(); err != nil {
			return nil, err
		}
		t0 := time.Now()
		eng2, err := engine.NewEngine(cfg, log)
		if err != nil {
			return nil, err
		}
		if err := eng2.Start(ctx); err != nil {
			return nil, err
		}
		res.RestartSec = time.Since(t0).Seconds()
		fmt.Printf("  restart took %.1fs\n", res.RestartSec)
		return res, eng2.Stop()
	}

	return res, eng.Stop()
}

func humanBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

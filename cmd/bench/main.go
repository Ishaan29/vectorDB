// Command bench is an end-to-end benchmark harness for vectorDB.
//
// It generates a deterministic dataset, computes exact ground truth by
// brute force, then measures the index level (pure HNSW) and the engine
// level (HNSW + BadgerDB) separately:
//
//   - build/ingest throughput and per-insert latency
//   - search latency percentiles (p50/p90/p95/p99) and single-thread QPS
//   - recall@k against exact ground truth, across k and an efSearch sweep
//   - concurrent throughput under parallel load
//   - index memory footprint, optional restart (rebuild) time
//
// Results are printed and written as JSON so runs can be diffed across
// versions. Keep -seed fixed so every version sees identical data.
//
// Usage:
//
//	go run ./cmd/bench                        # full 100k baseline
//	go run ./cmd/bench -n 10000 -queries 200  # quick smoke run
//	go run ./cmd/bench -mode index            # skip the engine/Badger phase
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ishaan29/vectorDB/pkg/vectormath"
)

type Params struct {
	N              int    `json:"n"`
	Dim            int    `json:"dim"`
	Queries        int    `json:"queries"`
	KList          []int  `json:"k_list"`
	EfList         []int  `json:"ef_list"`
	Dist           string `json:"dist"`
	Clusters       int    `json:"clusters"`
	Seed           int64  `json:"seed"`
	Workers        int    `json:"workers"`
	ConcurrentOps  int    `json:"concurrent_ops"`
	Warmup         int    `json:"warmup"`
	M              int    `json:"hnsw_m"`
	EfConstruction int    `json:"hnsw_ef_construction"`
	DefaultEf      int    `json:"hnsw_ef_search_default"`
}

type MachineInfo struct {
	GoVersion   string `json:"go_version"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	CPUs        int    `json:"cpus"`
	CPUModel    string `json:"cpu_model,omitempty"`
	MathBackend string `json:"math_backend"`
}

type Results struct {
	Label     string        `json:"label"`
	Timestamp string        `json:"timestamp"`
	GitCommit string        `json:"git_commit,omitempty"`
	Machine   MachineInfo   `json:"machine"`
	Params    Params        `json:"params"`
	Index     *LevelResults `json:"index,omitempty"`
	Engine    *LevelResults `json:"engine,omitempty"`
}

func main() {
	var (
		n        = flag.Int("n", 100000, "number of base vectors")
		dim      = flag.Int("dim", 128, "vector dimensions")
		queries  = flag.Int("queries", 1000, "number of held-out query vectors")
		kCSV     = flag.String("k", "1,10,100", "comma-separated k values (run at default efSearch)")
		efCSV    = flag.String("ef", "16,32,64,128,256", "comma-separated efSearch values (run at k=10)")
		dist     = flag.String("dist", "clustered", "data distribution: clustered|uniform")
		clusters = flag.Int("clusters", 100, "cluster count for -dist clustered")
		seed     = flag.Int64("seed", 42, "RNG seed; keep fixed to compare versions on identical data")
		mode     = flag.String("mode", "all", "what to bench: all|index|engine")
		workers  = flag.Int("workers", runtime.NumCPU(), "goroutines for the concurrent phase")
		concOps  = flag.Int("concurrent-ops", 5000, "total searches in the concurrent phase")
		warmup   = flag.Int("warmup", 50, "warmup queries before timed runs")
		restart  = flag.Bool("restart", false, "also measure engine restart (badger reopen + index rebuild)")
		label    = flag.String("label", "baseline", "label recorded in the result file")
		outDir   = flag.String("out", "bench/results", "directory for JSON result files")
		dataDir  = flag.String("data", "", "parent dir for the engine's badger data (default: system temp)")
	)
	flag.Parse()

	p := Params{
		N: *n, Dim: *dim, Queries: *queries,
		KList: parseInts(*kCSV), EfList: parseInts(*efCSV),
		Dist: *dist, Clusters: *clusters, Seed: *seed,
		Workers: *workers, ConcurrentOps: *concOps, Warmup: *warmup,
		M: hnswM, EfConstruction: hnswEfConstruction, DefaultEf: defaultEfSearch,
	}

	kmax := 10
	for _, k := range p.KList {
		if k > kmax {
			kmax = k
		}
	}
	if kmax > p.N {
		fatal("k (%d) cannot exceed n (%d)", kmax, p.N)
	}

	res := &Results{
		Label:     *label,
		Timestamp: time.Now().Format(time.RFC3339),
		GitCommit: gitCommit(),
		Machine: MachineInfo{
			GoVersion:   runtime.Version(),
			OS:          runtime.GOOS,
			Arch:        runtime.GOARCH,
			CPUs:        runtime.NumCPU(),
			CPUModel:    cpuModel(),
			MathBackend: vectormath.Backend(),
		},
		Params: p,
	}

	fmt.Printf("vectorDB bench | n=%d dim=%d queries=%d dist=%s seed=%d | math=%s cpus=%d\n",
		p.N, p.Dim, p.Queries, p.Dist, p.Seed, res.Machine.MathBackend, res.Machine.CPUs)

	t0 := time.Now()
	ds := GenerateDataset(p.N, p.Dim, p.Queries, p.Dist, p.Clusters, p.Seed)
	fmt.Printf("dataset generated in %.1fs\n", time.Since(t0).Seconds())

	t0 = time.Now()
	gt := ComputeGroundTruth(ds, kmax, runtime.NumCPU())
	fmt.Printf("exact ground truth (top-%d per query, brute force) in %.1fs\n", kmax, time.Since(t0).Seconds())

	var err error
	if *mode == "all" || *mode == "index" {
		res.Index, err = runIndexBench(p, ds, gt)
		if err != nil {
			fatal("index bench: %v", err)
		}
	}
	if *mode == "all" || *mode == "engine" {
		res.Engine, err = runEngineBench(p, ds, gt, *dataDir, *restart)
		if err != nil {
			fatal("engine bench: %v", err)
		}
	}

	path, err := writeResults(res, *outDir)
	if err != nil {
		fatal("writing results: %v", err)
	}

	printSummary(res)
	fmt.Printf("\nresults written to %s\n", path)
}

func writeResults(res *Results, outDir string) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s_%s.json", res.Label, time.Now().Format("20060102-150405"))
	path := filepath.Join(outDir, name)
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o644)
}

func printSummary(res *Results) {
	fmt.Printf("\n================ SUMMARY (%s) ================\n", res.Label)
	printLevel := func(name string, lr *LevelResults) {
		if lr == nil {
			return
		}
		fmt.Printf("\n[%s]\n", name)
		if lr.Build != nil {
			fmt.Printf("  build: %.1fs total, %.0f vec/s", lr.Build.TotalSec, lr.Build.VecPerSec)
			if lr.Build.InsertLat != nil {
				fmt.Printf(", insert p50=%.2fms p99=%.2fms", lr.Build.InsertLat.P50Ms, lr.Build.InsertLat.P99Ms)
			}
			if lr.Build.HeapBytes > 0 {
				fmt.Printf(", heap +%s (%.0f B/vec)", humanBytes(lr.Build.HeapBytes), lr.Build.BytesPerVec)
			}
			fmt.Println()
		}
		fmt.Printf("  %-6s %-6s %-8s %-9s %-9s %-9s %-9s %s\n",
			"k", "ef", "recall", "qps", "p50(ms)", "p95(ms)", "p99(ms)", "workers")
		for _, s := range append(append([]SearchRun{}, lr.Searches...), lr.Concurrent...) {
			fmt.Printf("  %-6d %-6d %-8.3f %-9.0f %-9.2f %-9.2f %-9.2f %d\n",
				s.K, s.Ef, s.Recall, s.QPS, s.Latency.P50Ms, s.Latency.P95Ms, s.Latency.P99Ms, s.Workers)
		}
		if lr.RestartSec > 0 {
			fmt.Printf("  restart (reopen + rebuild): %.1fs\n", lr.RestartSec)
		}
	}
	printLevel("index level — pure HNSW", res.Index)
	printLevel("engine level — HNSW + BadgerDB", res.Engine)
}

func parseInts(csv string) []int {
	var out []int
	for _, part := range strings.Split(csv, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		v, err := strconv.Atoi(part)
		if err != nil || v < 1 {
			fatal("invalid integer list entry %q", part)
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		fatal("empty integer list %q", csv)
	}
	return out
}

func gitCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func cpuModel() string {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return ""
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "bench: "+format+"\n", args...)
	os.Exit(1)
}

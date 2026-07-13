package main

import (
	"container/heap"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
)

// Dataset holds the generated base vectors and held-out query vectors.
// Generation is fully deterministic for a given seed so different DB
// versions can be compared on identical data.
type Dataset struct {
	Base    [][]float32
	IDs     []string
	Queries [][]float32
}

func GenerateDataset(n, dim, queries int, dist string, clusters int, seed int64) *Dataset {
	rng := rand.New(rand.NewSource(seed))

	var centers [][]float32
	if dist == "clustered" {
		centers = make([][]float32, clusters)
		for i := range centers {
			c := make([]float32, dim)
			for d := range c {
				c[d] = rng.Float32()*2 - 1
			}
			centers[i] = c
		}
	}

	gen := func() []float32 {
		v := make([]float32, dim)
		if dist == "clustered" {
			c := centers[rng.Intn(len(centers))]
			for d := range v {
				v[d] = c[d] + float32(rng.NormFloat64())*0.15
			}
		} else {
			for d := range v {
				v[d] = rng.Float32()*2 - 1
			}
		}
		return v
	}

	ds := &Dataset{
		Base:    make([][]float32, n),
		IDs:     make([]string, n),
		Queries: make([][]float32, queries),
	}
	for i := 0; i < n; i++ {
		ds.Base[i] = gen()
		ds.IDs[i] = fmt.Sprintf("v%07d", i)
	}
	for i := 0; i < queries; i++ {
		ds.Queries[i] = gen()
	}
	return ds
}

type simPair struct {
	idx int
	sim float32
}

// min-heap by similarity, used for exact top-k selection
type simHeap []simPair

func (h simHeap) Len() int            { return len(h) }
func (h simHeap) Less(i, j int) bool  { return h[i].sim < h[j].sim }
func (h simHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *simHeap) Push(x interface{}) { *h = append(*h, x.(simPair)) }
func (h *simHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

func l2norm(v []float32) float32 {
	var sum float32
	for _, x := range v {
		sum += x * x
	}
	return float32(math.Sqrt(float64(sum)))
}

// ComputeGroundTruth brute-forces the exact top-kmax neighbors (by cosine
// similarity) for every query, parallelized across workers. The result is
// the reference against which recall is measured.
func ComputeGroundTruth(ds *Dataset, kmax, workers int) [][]string {
	norms := make([]float32, len(ds.Base))
	for i, v := range ds.Base {
		norms[i] = l2norm(v)
	}

	gt := make([][]string, len(ds.Queries))
	jobs := make(chan int)
	var wg sync.WaitGroup
	if workers < 1 {
		workers = 1
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for qi := range jobs {
				gt[qi] = exactTopK(ds, norms, ds.Queries[qi], kmax)
			}
		}()
	}
	for qi := range ds.Queries {
		jobs <- qi
	}
	close(jobs)
	wg.Wait()
	return gt
}

func exactTopK(ds *Dataset, norms []float32, q []float32, k int) []string {
	qn := l2norm(q)
	h := make(simHeap, 0, k)
	heap.Init(&h)
	for i, v := range ds.Base {
		var dot float32
		for d := range q {
			dot += q[d] * v[d]
		}
		sim := dot / (qn * norms[i])
		if len(h) < k {
			heap.Push(&h, simPair{i, sim})
		} else if sim > h[0].sim {
			h[0] = simPair{i, sim}
			heap.Fix(&h, 0)
		}
	}
	sort.Slice(h, func(a, b int) bool { return h[a].sim > h[b].sim })
	ids := make([]string, len(h))
	for i, p := range h {
		ids[i] = ds.IDs[p.idx]
	}
	return ids
}

// recallAtK = |returned top-k ∩ exact top-k| / k
func recallAtK(got, exact []string, k int) float64 {
	if k > len(exact) {
		k = len(exact)
	}
	if k == 0 {
		return 0
	}
	set := make(map[string]struct{}, k)
	for _, id := range exact[:k] {
		set[id] = struct{}{}
	}
	n := k
	if len(got) < n {
		n = len(got)
	}
	hits := 0
	for _, id := range got[:n] {
		if _, ok := set[id]; ok {
			hits++
		}
	}
	return float64(hits) / float64(k)
}

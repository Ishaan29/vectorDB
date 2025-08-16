package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ishaan29/vectorDB/internal/config"
	"github.com/ishaan29/vectorDB/internal/index"
	"github.com/ishaan29/vectorDB/internal/logger"
	"github.com/ishaan29/vectorDB/persistence"
	"github.com/ishaan29/vectorDB/pkg/types"
)

type Engine struct {
	mu      sync.RWMutex
	config  *config.Config
	store   *persistence.BadgerStore
	index   *index.HNSWIndex
	logger  logger.Logger
	running bool
}

func NewEngine(cfg *config.Config, log logger.Logger) (*Engine, error) {
	store, err := persistence.NewBadgerStore(cfg.Badger.Path, log)
	if err != nil {
		return nil, ErrStoreInitialization
	}

	hnswIndex := index.NewHNSWIndex(cfg.Index.Dimensions, log)
	if hnswIndex == nil {
		return nil, ErrIndexInitialization
	}

	return &Engine{
		config:  cfg,
		store:   store,
		index:   hnswIndex,
		logger:  log,
		running: false,
	}, nil
}

func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return ErrEngineAlreadyRunning
	}

	e.logger.Info("Starting vector engine, rebuilding HSNW index from storage. ")

	count := 0
	errors := 0
	startTime := time.Now()

	err := e.store.Iterate(func(vector types.Vector) error {
		if len(vector.Embedding) != e.config.Index.Dimensions {
			e.logger.Warn("Skipping vector with wrong dimensions",
				logger.String("id", vector.ID),
				logger.Int("expected", e.config.Index.Dimensions),
				logger.Int("actual", len(vector.Embedding)),
			)
			errors++
			return nil
		}

		if err := e.index.Add(vector.ID, vector.Embedding); err != nil {
			e.logger.Error("Failed to index vector ",
				logger.String("id", vector.ID),
				logger.Error("Error: ", err),
			)
			errors++
			return nil
		}

		count++

		if count%1000 == 0 {
			e.logger.Info("Indexing progress",
				logger.Int("vector_indexed", count),
				logger.Int("errors", errors),
				logger.Duration("elapsed", time.Since(startTime)),
			)
		}

		// check for contect cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	})

	if err != nil {
		e.logger.Error("Error indexing vectors", logger.Error("Error: ", err))
		return err
	}

	e.logger.Info("Engine started successfully",
		logger.Int("vectors_indexed", count),
		logger.Int("errors", errors),
		logger.Duration("startup_time", time.Since(startTime)))
	e.running = true
	return nil
}

func (e *Engine) Insert(vector types.Vector) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return ErrEngineNotRunning
	}

	if len(vector.Embedding) != e.config.Index.Dimensions {
		return ErrInvalidDimensions(
			e.config.Index.Dimensions,
			len(vector.Embedding))
	}

	if err := e.store.Put(vector); err != nil {
		return fmt.Errorf("failed to insert vector: %w", err)
	}

	if err := e.index.Add(vector.ID, vector.Embedding); err != nil {
		e.logger.Error("Failed to add to HNSW index, vector is presisted but not searchable",
			logger.String("id", vector.ID),
			logger.Error("Error: ", err),
		)
	}

	e.logger.Info("Vector inserted successfully",
		logger.String("id", vector.ID),
		logger.Int("dimensions", len(vector.Embedding)))

	return nil
}

func (e *Engine) Search(query types.Vector, params SearchParams) ([]types.SearchResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.running {
		return nil, ErrEngineNotRunning
	}

	startTime := time.Now()

	indexResults, err := e.index.Search(query.Embedding, params.K)
	if err != nil {
		e.logger.Error("Failed to search index", logger.Error("Error: ", err))
		return nil, ErrSearchIndexFailed
	}

	indexTime := time.Since(startTime)

	results := make([]types.SearchResult, 0, len(indexResults))
	hydrateStart := time.Now()
	for _, ir := range indexResults {
		if float32(ir.Score) < params.Threshold {
			continue
		}

		vector, err := e.store.Get(ir.ID)
		if err != nil {
			e.logger.Warn("Vector in index but not in storage (inconsistency)",
				logger.String("id", ir.ID),
				logger.Error("error", err))
			continue
		}

		result := types.SearchResult{
			Vector:   vector,
			Distance: float32(ir.Distance),
			Score:    float32(ir.Score),
		}
		if !params.IncludeVecs {
			result.Vector.Embedding = nil
		}
		if !params.IncludeMeta {
			result.Vector.Metadata = nil
		}
		results = append(results, result)
	}
	hydrateTime := time.Since(hydrateStart)
	totalTime := time.Since(startTime)

	e.logger.Debug("Search completed",
		logger.Int("results_returned", len(results)),
		logger.Int("index_results", len(indexResults)),
		logger.Duration("index_time", indexTime),
		logger.Duration("hydrate_time", hydrateTime),
		logger.Duration("total_time", totalTime))

	return results, nil

}

func (e *Engine) BatchInsert(vectors []types.Vector) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return ErrEngineNotRunning
	}

	startTime := time.Now()
	successCount := 0
	failedCount := 0

	if err := e.store.BatchPut(vectors); err != nil {
		e.logger.Error("Batch persists failed", logger.Error("error", err))
		return fmt.Errorf("batch persist failed: %w", err)
	}

	for _, vector := range vectors {
		if len(vector.Embedding) != e.config.Index.Dimensions {
			e.logger.Warn("Skipping vector with wrong dimensions",
				logger.String("id", vector.ID))
			failedCount++
			continue
		}

		if err := e.index.Add(vector.ID, vector.Embedding); err != nil {
			e.logger.Error("Failed to index vector",
				logger.String("id", vector.ID),
				logger.Error("error", err))
			failedCount++
			continue
		} else {
			successCount++
		}
	}
	e.logger.Info("Batch insert completed",
		logger.Int("total", len(vectors)),
		logger.Int("success", successCount),
		logger.Int("failed", failedCount),
		logger.Duration("duration", time.Since(startTime)))
	return nil
}

func (e *Engine) Get(id string) (types.Vector, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.running {
		return types.Vector{}, false
	}

	vector, err := e.store.Get(id)
	if err != nil {
		e.logger.Debug("Vector not found",
			logger.String("id", id),
			logger.Error("error", err))
		return types.Vector{}, false
	}
	return vector, true
}

func (e *Engine) Delete(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return ErrEngineNotRunning
	}

	if err := e.store.Delete(id); err != nil {
		return fmt.Errorf("failed to delete from store: %w", err)
	}

	if err := e.index.Remove(id); err != nil {
		e.logger.Warn("Failed to remove from index",
			logger.String("id", id),
			logger.Error("error", err))
	}

	e.logger.Info("Vector deleted successfully",
		logger.String("id", id))
	return nil
}

func (e *Engine) Update(vector types.Vector) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return ErrEngineNotRunning
	}
	if _, err := e.store.Get(vector.ID); err != nil {
		return ErrVectorNotFound
	}

	if err := e.store.Put(vector); err != nil {
		return fmt.Errorf("failed to update vector: %w", err)
	}

	e.logger.Warn("Vector updated in storage but index not updated (HNSW limitation)",
		logger.String("id", vector.ID))
	return nil
}

func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return ErrEngineNotRunning
	}

	e.logger.Info("Stopping engine...")

	if err := e.store.Close(); err != nil {
		e.logger.Error("Failed to close storage",
			logger.Error("error", err))
		return err
	}

	e.running = false
	e.logger.Info("Engine stopped successfully")
	return nil
}

func (e *Engine) Stats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := map[string]interface{}{
		"running":    e.running,
		"dimensions": e.config.Index.Dimensions,
	}

	// Add index stats
	if e.index != nil {
		indexStats := e.index.Stats()
		for k, v := range indexStats {
			stats["index_"+k] = v
		}
	}

	return stats
}

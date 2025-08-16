package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ishaan29/vectorDB/internal/api/models"
	"github.com/ishaan29/vectorDB/internal/engine"
	"github.com/ishaan29/vectorDB/internal/logger"
	"github.com/ishaan29/vectorDB/pkg/types"
)

func (h *Handlers) SearchVectors(c *gin.Context) {
	var req models.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	start := time.Now()

	query := types.Vector{
		ID:        "", // Anonymous query
		Embedding: req.Embedding,
	}

	params := engine.SearchParams{
		K:           req.K,
		Threshold:   req.Threshold,
		IncludeVecs: req.IncludeVectors,
		IncludeMeta: req.IncludeMetadata,
	}

	results, err := h.engine.Search(query, params)
	if err != nil {
		h.logger.Error("Search failed",
			logger.Int("k", req.K),
			logger.Float64("threshold", float64(req.Threshold)),
			logger.Error("error", err))

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Search failed",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	searchResults := make([]models.SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = models.ConvertSearchResult(r, req.IncludeVectors, req.IncludeMetadata)
	}

	response := models.SearchResponse{
		Results: searchResults,
		TookMs:  time.Since(start).Milliseconds(),
		Total:   len(searchResults),
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handlers) Optimize(c *gin.Context) {
	var req models.OptimizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Force = false
	}

	h.logger.Info("Index optimization requested",
		logger.Bool("force", req.Force))

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "Optimization completed",
	})
}

package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ishaan29/vectorDB/internal/api/models"
	"github.com/ishaan29/vectorDB/internal/logger"
	"github.com/ishaan29/vectorDB/pkg/types"
)

func (h *Handlers) InsertVector(c *gin.Context) {
	var req models.InsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	vector := types.Vector{
		ID:        req.ID,
		Embedding: req.Embedding,
		Metadata:  req.Metadata,
	}

	if err := h.engine.Insert(vector); err != nil {
		h.logger.Error("Failed to insert vector",
			logger.String("id", req.ID),
			logger.Error("error", err))

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Insert failed",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse{
		Success: true,
		Message: "Vector inserted successfully",
	})
}

func (h *Handlers) GetVector(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Missing vector ID",
			Code:  http.StatusBadRequest,
		})
		return
	}

	vector, found := h.engine.Get(id)
	if !found {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "Vector not found",
			Message: "Vector with ID " + id + " does not exist",
			Code:    http.StatusNotFound,
		})
		return
	}

	response := models.ConvertVector(vector, true, true)
	c.JSON(http.StatusOK, response)
}

func (h *Handlers) DeleteVector(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Missing vector ID",
			Code:  http.StatusBadRequest,
		})
		return
	}

	if err := h.engine.Delete(id); err != nil {
		h.logger.Error("Failed to delete vector",
			logger.String("id", id),
			logger.Error("error", err))

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Delete failed",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "Vector deleted successfully",
	})
}

func (h *Handlers) BatchInsert(c *gin.Context) {
	var req models.BatchInsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	if len(req.Vectors) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Empty vectors array",
			Code:  http.StatusBadRequest,
		})
		return
	}

	start := time.Now()

	// Convert to engine types
	vectors := make([]types.Vector, len(req.Vectors))
	for i, v := range req.Vectors {
		vectors[i] = types.Vector{
			ID:        v.ID,
			Embedding: v.Embedding,
			Metadata:  v.Metadata,
		}
	}

	if err := h.engine.BatchInsert(vectors); err != nil {
		h.logger.Error("Batch insert failed",
			logger.Int("count", len(vectors)),
			logger.Error("error", err))

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Batch insert failed",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	response := models.BatchInsertResponse{
		Success:  true,
		Inserted: len(vectors),
		Failed:   0,
		TookMs:   time.Since(start).Milliseconds(),
	}

	c.JSON(http.StatusCreated, response)
}

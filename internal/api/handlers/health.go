package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ishaan29/vectorDB/internal/api/models"
)

func (h *Handlers) Health(c *gin.Context) {
	stats := h.engine.Stats()

	status := "ok"
	if !stats["running"].(bool) {
		status = "engine_not_running"
	}

	response := models.HealthResponse{
		Status:  status,
		Engine:  "vectordb",
		Version: "1.0.0",
		Stats:   stats,
	}

	if status == "ok" {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

func (h *Handlers) Stats(c *gin.Context) {
	start := time.Now()

	stats := h.engine.Stats()

	response := models.StatsResponse{
		Stats:  stats,
		TookMs: time.Since(start).Milliseconds(),
	}

	c.JSON(http.StatusOK, response)
}

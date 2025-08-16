package handlers

import (
	"github.com/ishaan29/vectorDB/internal/engine"
	"github.com/ishaan29/vectorDB/internal/logger"
)

type Handlers struct {
	engine *engine.Engine
	logger logger.Logger
}

func NewHandlers(eng *engine.Engine, log logger.Logger) *Handlers {
	return &Handlers{
		engine: eng,
		logger: log,
	}
}

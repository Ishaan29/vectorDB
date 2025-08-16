package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ishaan29/vectorDB/internal/api/handlers"
	"github.com/ishaan29/vectorDB/internal/api/middleware"
	"github.com/ishaan29/vectorDB/internal/config"
	"github.com/ishaan29/vectorDB/internal/engine"
	"github.com/ishaan29/vectorDB/internal/logger"
)

type Server struct {
	engine     *engine.Engine
	logger     logger.Logger
	config     *config.Config
	httpServer *http.Server
	router     *gin.Engine
}

func NewServer(eng *engine.Engine, log logger.Logger, cfg *config.Config) *Server {
	s := &Server{
		engine: eng,
		logger: log,
		config: cfg,
	}

	s.setupRouter()
	return s
}

func (s *Server) setupRouter() {
	if s.config.Logging.DevMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(middleware.Logger(s.logger))
	r.Use(middleware.Recovery(s.logger))
	r.Use(middleware.CORS())

	h := handlers.NewHandlers(s.engine, s.logger)

	r.GET("/health", h.Health)
	r.GET("/stats", h.Stats)

	v1 := r.Group("/api/v1")
	{

		v1.POST("/vectors", h.InsertVector)
		v1.POST("/vectors/batch", h.BatchInsert)
		v1.GET("/vectors/:id", h.GetVector)
		v1.DELETE("/vectors/:id", h.DeleteVector)

		v1.POST("/search", h.SearchVectors)

		v1.POST("/optimize", h.Optimize)
	}

	s.router = r
}

func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting HTTP server",
		logger.String("address", addr),
		logger.String("mode", gin.Mode()))

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server failed", logger.Error("error", err))
		}
	}()

	<-ctx.Done()

	s.logger.Info("Shutting down HTTP server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("HTTP server shutdown failed", logger.Error("error", err))
		return err
	}

	s.logger.Info("HTTP server stopped")
	return nil
}

func (s *Server) Stop() error {
	if s.httpServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

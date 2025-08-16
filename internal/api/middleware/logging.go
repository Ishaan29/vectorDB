package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/ishaan29/vectorDB/internal/logger"
)

func Logger(log logger.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		log.Info("HTTP Request",
			logger.String("method", param.Method),
			logger.String("path", param.Path),
			logger.String("query", param.Request.URL.RawQuery),
			logger.String("ip", param.ClientIP),
			logger.String("user_agent", param.Request.UserAgent()),
			logger.Int("status", param.StatusCode),
			logger.Duration("latency", param.Latency),
			logger.String("error", param.ErrorMessage),
		)
		return ""
	})
}

func Recovery(log logger.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			log.Error("API panic recovered",
				logger.String("error", err),
				logger.String("path", c.Request.URL.Path),
				logger.String("method", c.Request.Method))
		} else if err, ok := recovered.(error); ok {
			log.Error("API panic recovered",
				logger.Error("error", err),
				logger.String("path", c.Request.URL.Path),
				logger.String("method", c.Request.Method))
		} else {
			log.Error("API panic recovered",
				logger.String("error", "unknown panic"),
				logger.String("path", c.Request.URL.Path),
				logger.String("method", c.Request.Method))
		}

		c.JSON(500, gin.H{
			"error": "Internal server error",
			"code":  500,
		})
	})
}

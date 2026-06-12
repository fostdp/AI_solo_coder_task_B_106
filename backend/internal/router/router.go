package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"stone-relic-monitor/internal/config"
	"time"
)

func SetupBasicRouter(cfg *config.Config) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"version": "2.0.0",
			"modules": []string{"ethercat_ingest", "scaling_model", "laser_threshold", "alert_ws"},
		})
	})

	return r
}

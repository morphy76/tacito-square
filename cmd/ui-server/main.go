package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Initialize structured logging using zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(os.Stdout)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dir := os.Getenv("STATIC_DIR")
	if dir == "" {
		dir = "/app/dist"
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Health and readiness check endpoints
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Static asset routing and fallback for React SPA client-side routing
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		safePath := filepath.Clean(path)
		if strings.HasPrefix(safePath, "..") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
			return
		}

		targetFile := filepath.Join(dir, safePath)
		if fileExists(targetFile) {
			c.File(targetFile)
			return
		}

		// Fallback to index.html for client-side routing (e.g. React Router)
		c.File(filepath.Join(dir, "index.html"))
	})

	log.Info().
		Str("dir", dir).
		Str("port", port).
		Msg("starting static UI server")

	if err := r.Run(":" + port); err != nil {
		log.Fatal().Err(err).Msg("static server failed to start")
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

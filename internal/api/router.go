package api

import (
	"io"
	"io/fs"
	"net/http"
	"time"

	"rocksdb-cli/internal/api/handlers"
	"rocksdb-cli/internal/api/middleware"
	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/service"

	"github.com/gin-gonic/gin"
)

// SetupRouter configures and returns a Gin router with all API routes
func SetupRouter(database db.KeyValueDB) *gin.Engine {
	// Set Gin to release mode in production
	// gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// Global middleware
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())
	r.Use(gin.Recovery()) // Recover from panics

	// Create services
	dbService := service.NewDatabaseService(database)
	scanService := service.NewScanService(database)
	searchService := service.NewSearchService(database)
	statsService := service.NewStatsService(database)

	// Create handlers
	dbHandler := handlers.NewDatabaseHandler(dbService)
	scanHandler := handlers.NewScanHandler(scanService)
	searchHandler := handlers.NewSearchHandler(searchService)
	statsHandler := handlers.NewStatsHandler(statsService)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"version": "1.0.0",
			})
		})

		// Database info routes
		v1.GET("/cf", dbHandler.ListColumnFamilies)
		v1.GET("/stats", statsHandler.GetDatabaseStats)

		// Column family routes
		cf := v1.Group("/cf/:cf")
		{
			// Basic operations
			cf.GET("/get/:key", dbHandler.GetValue)
			cf.POST("/put", dbHandler.PutValue)
			cf.DELETE("/delete/:key", dbHandler.DeleteValue)
			cf.GET("/last", dbHandler.GetLastEntry)

			// Scan operations
			cf.POST("/scan", scanHandler.Scan)
			cf.POST("/prefix", scanHandler.PrefixScan)

			// Search operations
			cf.POST("/search", searchHandler.Search)
			cf.POST("/jsonquery", searchHandler.JSONQuery)

			// Stats
			cf.GET("/stats", statsHandler.GetColumnFamilyStats)
		}
	}

	// Root route
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":    "RocksDB CLI API",
			"version": "1.0.0",
			"docs":    "/api/v1",
		})
	})

	return r
}

// SetupRouterWithUI configures and returns a Gin router with API routes and embedded Web UI
func SetupRouterWithUI(database db.KeyValueDB, staticFS fs.FS) *gin.Engine {
	r := gin.New()

	// Global middleware
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())
	r.Use(gin.Recovery())

	// Create services
	dbService := service.NewDatabaseService(database)
	scanService := service.NewScanService(database)
	searchService := service.NewSearchService(database)
	statsService := service.NewStatsService(database)

	// Create handlers
	dbHandler := handlers.NewDatabaseHandler(dbService)
	scanHandler := handlers.NewScanHandler(scanService)
	searchHandler := handlers.NewSearchHandler(searchService)
	statsHandler := handlers.NewStatsHandler(statsService)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"version": "1.0.0",
			})
		})

		// Database info routes
		v1.GET("/cf", dbHandler.ListColumnFamilies)
		v1.GET("/stats", statsHandler.GetDatabaseStats)

		// Column family routes
		cf := v1.Group("/cf/:cf")
		{
			// Basic operations
			cf.GET("/get/:key", dbHandler.GetValue)
			cf.POST("/put", dbHandler.PutValue)
			cf.DELETE("/delete/:key", dbHandler.DeleteValue)
			cf.GET("/last", dbHandler.GetLastEntry)

			// Scan operations
			cf.POST("/scan", scanHandler.Scan)
			cf.POST("/prefix", scanHandler.PrefixScan)

			// Search operations
			cf.POST("/search", searchHandler.Search)
			cf.POST("/jsonquery", searchHandler.JSONQuery)

			// Stats
			cf.GET("/stats", statsHandler.GetColumnFamilyStats)
		}
	}

	// Serve embedded static files
	// Use NoRoute to serve static files for all non-API routes
	r.NoRoute(func(c *gin.Context) {
		// Serve index.html for root and unknown routes (SPA routing)
		path := c.Request.URL.Path

		// Try to open the requested file
		file, err := staticFS.Open(path[1:]) // Remove leading slash
		if err != nil {
			// File not found, serve index.html for SPA routing
			indexFile, err := staticFS.Open("index.html")
			if err != nil {
				c.JSON(404, gin.H{"error": "Web UI not found"})
				return
			}
			defer indexFile.Close()

			c.Header("Content-Type", "text/html; charset=utf-8")
			http.ServeContent(c.Writer, c.Request, "index.html", time.Now(), indexFile.(io.ReadSeeker))
			return
		}
		defer file.Close()

		// Serve the requested file
		http.ServeContent(c.Writer, c.Request, path, time.Now(), file.(io.ReadSeeker))
	})

	return r
}

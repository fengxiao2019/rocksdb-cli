package api

import (
	"io"
	"net/http"
	"time"

	"rocksdb-cli/internal/api/handlers"
	"rocksdb-cli/internal/api/middleware"
	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/service"
	"rocksdb-cli/internal/webui"

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
func SetupRouterWithUI(dbManager *service.DBManager) *gin.Engine {
	r := gin.New()

	// Global middleware
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())
	r.Use(gin.Recovery())

	// Create database manager handler
	dbManagerHandler := handlers.NewDBManagerHandler(dbManager)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", func(c *gin.Context) {
			isConnected := dbManager.IsConnected()
			response := gin.H{
				"status":    "ok",
				"version":   "1.0.0",
				"connected": isConnected,
			}

			if isConnected {
				if info, err := dbManager.GetCurrentInfo(); err == nil && info != nil {
					response["database"] = info.Path
				}
			}

			c.JSON(200, response)
		})

		// Database management routes
		databases := v1.Group("/databases")
		{
			databases.POST("/connect", dbManagerHandler.Connect)
			databases.POST("/disconnect", dbManagerHandler.Disconnect)
			databases.GET("/current", dbManagerHandler.GetCurrent)
			databases.GET("/list", dbManagerHandler.ListAvailable)
			databases.POST("/validate", dbManagerHandler.Validate)
			databases.GET("/status", dbManagerHandler.GetStatus)
		}

		// Tools routes (no database connection required)
		tools := v1.Group("/tools")
		{
			ticksHandler := handlers.NewTicksHandler()
			ticks := tools.Group("/ticks")
			{
				ticks.POST("/from-datetime", ticksHandler.ConvertDateTimeToTicks)
				ticks.POST("/to-datetime", ticksHandler.ConvertTicksToDateTime)
			}
		}

		// Database operation routes (require active connection)
		// Use middleware to check connection
		connected := v1.Group("")
		connected.Use(func(c *gin.Context) {
			if !dbManager.IsConnected() {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error": "No database connected. Please connect to a database first.",
				})
				c.Abort()
				return
			}
			c.Next()
		})
		{
			// Get current database for operations
			getCurrentDB := func() (db.KeyValueDB, error) {
				return dbManager.GetCurrentDB()
			}

			// Create services with current database
			connected.GET("/cf", func(c *gin.Context) {
				rdb, _ := getCurrentDB() // Already validated by middleware
				dbService := service.NewDatabaseService(rdb)
				dbHandler := handlers.NewDatabaseHandler(dbService)
				dbHandler.ListColumnFamilies(c)
			})

			connected.GET("/stats", func(c *gin.Context) {
				rdb, _ := getCurrentDB() // Already validated by middleware
				statsService := service.NewStatsService(rdb)
				statsHandler := handlers.NewStatsHandler(statsService)
				statsHandler.GetDatabaseStats(c)
			})

			// Column family routes
			cf := connected.Group("/cf/:cf")
			{
				// Basic operations
				cf.GET("/get/:key", func(c *gin.Context) {
					rdb, _ := getCurrentDB()
					dbService := service.NewDatabaseService(rdb)
					dbHandler := handlers.NewDatabaseHandler(dbService)
					dbHandler.GetValue(c)
				})
				cf.POST("/put", func(c *gin.Context) {
					rdb, _ := getCurrentDB()
					dbService := service.NewDatabaseService(rdb)
					dbHandler := handlers.NewDatabaseHandler(dbService)
					dbHandler.PutValue(c)
				})
				cf.DELETE("/delete/:key", func(c *gin.Context) {
					rdb, _ := getCurrentDB()
					dbService := service.NewDatabaseService(rdb)
					dbHandler := handlers.NewDatabaseHandler(dbService)
					dbHandler.DeleteValue(c)
				})
				cf.GET("/last", func(c *gin.Context) {
					rdb, _ := getCurrentDB()
					dbService := service.NewDatabaseService(rdb)
					dbHandler := handlers.NewDatabaseHandler(dbService)
					dbHandler.GetLastEntry(c)
				})

				// Scan operations
				cf.POST("/scan", func(c *gin.Context) {
					rdb, _ := getCurrentDB()
					scanService := service.NewScanService(rdb)
					scanHandler := handlers.NewScanHandler(scanService)
					scanHandler.Scan(c)
				})
				cf.POST("/prefix", func(c *gin.Context) {
					rdb, _ := getCurrentDB()
					scanService := service.NewScanService(rdb)
					scanHandler := handlers.NewScanHandler(scanService)
					scanHandler.PrefixScan(c)
				})

				// Search operations
				cf.POST("/search", func(c *gin.Context) {
					rdb, _ := getCurrentDB()
					searchService := service.NewSearchService(rdb)
					searchHandler := handlers.NewSearchHandler(searchService)
					searchHandler.Search(c)
				})
				cf.POST("/jsonquery", func(c *gin.Context) {
					rdb, _ := getCurrentDB()
					searchService := service.NewSearchService(rdb)
					searchHandler := handlers.NewSearchHandler(searchService)
					searchHandler.JSONQuery(c)
				})

				// Stats
				cf.GET("/stats", func(c *gin.Context) {
					rdb, _ := getCurrentDB()
					statsService := service.NewStatsService(rdb)
					statsHandler := handlers.NewStatsHandler(statsService)
					statsHandler.GetColumnFamilyStats(c)
				})
			}
		}
	}

	// Serve embedded static files
	// Load embedded web UI
	staticFS, err := webui.GetDistFS()
	if err != nil {
		// If web UI is not available, just serve API
		r.GET("/", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"name":    "RocksDB Web Server",
				"version": "1.0.0",
				"message": "Web UI not embedded. API available at /api/v1",
			})
		})
	} else {
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
	}

	return r
}

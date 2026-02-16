package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

func main() {
	// Parse command-line flags
	dbPath := flag.String("db", os.Getenv("DB_PATH"), "Path to the CPG SQLite database")
	port := flag.String("port", "8080", "HTTP server port")
	staticDir := flag.String("static", "", "Static files directory (optional)")
	flag.Parse()

	// Validate database path
	if *dbPath == "" {
		log.Fatal("Database path is required. Use -db flag or DB_PATH environment variable.")
	}

	absPath, err := filepath.Abs(*dbPath)
	if err != nil {
		log.Fatalf("Invalid database path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Fatalf("Database file does not exist: %s", absPath)
	}

	// Initialize database connection
	db, err := NewDB(absPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database by getting stats
	stats, err := db.GetStats()
	if err != nil {
		log.Fatalf("Failed to query database: %v", err)
	}
	log.Printf("Database loaded successfully: %v", stats)

	// Create API handler
	api := NewAPI(db)

	// Create router
	r := mux.NewRouter()

	// API routes
	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/health", api.HealthHandler).Methods("GET")
	apiRouter.HandleFunc("/stats", api.StatsHandler).Methods("GET")
	apiRouter.HandleFunc("/search", api.SearchHandler).Methods("GET")
	// Register more specific routes first to avoid conflicts
	apiRouter.HandleFunc("/functions/{id:.+}/callgraph", api.GetCallGraphHandler).Methods("GET")
	apiRouter.HandleFunc("/functions/{id:.+}", api.GetFunctionHandler).Methods("GET")
	apiRouter.HandleFunc("/packages/graph", api.GetPackageGraphHandler).Methods("GET")
	apiRouter.HandleFunc("/packages/{name}/functions", api.GetPackageFunctionsHandler).Methods("GET")
	apiRouter.HandleFunc("/packages", api.GetPackagesHandler).Methods("GET")
	apiRouter.HandleFunc("/source", api.GetSourceHandler).Methods("GET")

	// Serve static files if provided
	if *staticDir != "" {
		log.Printf("Serving static files from: %s", *staticDir)
		r.PathPrefix("/").Handler(http.FileServer(http.Dir(*staticDir)))
	} else {
		// Default handler for root
		r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>CPG Explorer API</title>
    <style>
        body { font-family: sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        .endpoint { background: #f5f5f5; padding: 10px; margin: 10px 0; border-left: 4px solid #007bff; }
        code { background: #e9ecef; padding: 2px 6px; border-radius: 3px; }
    </style>
</head>
<body>
    <h1>CPG Explorer API</h1>
    <p>The API is running. Available endpoints:</p>

    <div class="endpoint">
        <strong>GET /api/health</strong><br>
        Health check
    </div>

    <div class="endpoint">
        <strong>GET /api/stats</strong><br>
        Database statistics
    </div>

    <div class="endpoint">
        <strong>GET /api/search?q={query}&kind={kind}&limit={limit}</strong><br>
        Search symbols by name
    </div>

    <div class="endpoint">
        <strong>GET /api/functions/{id}</strong><br>
        Get function details
    </div>

    <div class="endpoint">
        <strong>GET /api/functions/{id}/callgraph?depth={depth}</strong><br>
        Get call graph neighborhood (default depth=2)
    </div>

    <div class="endpoint">
        <strong>GET /api/packages</strong><br>
        List all packages
    </div>

    <div class="endpoint">
        <strong>GET /api/packages/graph</strong><br>
        Get package dependency graph
    </div>

    <div class="endpoint">
        <strong>GET /api/packages/{name}/functions?limit={limit}</strong><br>
        Get top functions in a package
    </div>

    <div class="endpoint">
        <strong>GET /api/source?file={path}</strong><br>
        Get source code for a file
    </div>
</body>
</html>
			`))
		}).Methods("GET")
	}

	// Apply middleware
	handler := CORSMiddleware(LoggingMiddleware(r))

	// Start server
	addr := ":" + *port
	log.Printf("Starting server on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

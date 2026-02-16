package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// API wraps the database and provides HTTP handlers
type API struct {
	db *DB
}

// NewAPI creates a new API instance
func NewAPI(db *DB) *API {
	return &API{db: db}
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

// respondError writes an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// HealthHandler returns the health status
func (api *API) HealthHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// StatsHandler returns database statistics
func (api *API) StatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := api.db.GetStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, stats)
}

// SearchHandler searches for symbols
func (api *API) SearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		respondError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	kind := r.URL.Query().Get("kind")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	results, err := api.db.SearchSymbols(query, kind, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, results)
}

// GetFunctionHandler returns function details
func (api *API) GetFunctionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	functionID := vars["id"]

	function, err := api.db.GetFunction(functionID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, function)
}

// GetCallGraphHandler returns the call graph for a function
func (api *API) GetCallGraphHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	functionID := vars["id"]

	depthStr := r.URL.Query().Get("depth")
	depth := 2
	if depthStr != "" {
		if d, err := strconv.Atoi(depthStr); err == nil && d > 0 {
			depth = d
		}
	}

	graph, err := api.db.GetCallGraph(functionID, depth)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, graph)
}

// GetPackagesHandler returns all packages
func (api *API) GetPackagesHandler(w http.ResponseWriter, r *http.Request) {
	packages, err := api.db.GetPackages()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, packages)
}

// GetPackageGraphHandler returns the package dependency graph
func (api *API) GetPackageGraphHandler(w http.ResponseWriter, r *http.Request) {
	graph, err := api.db.GetPackageGraph()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, graph)
}

// GetPackageFunctionsHandler returns top functions in a package
func (api *API) GetPackageFunctionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	packageName := vars["name"]

	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	functions, err := api.db.GetPackageFunctions(packageName, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, functions)
}

// GetSourceHandler returns source code for a file
func (api *API) GetSourceHandler(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if file == "" {
		respondError(w, http.StatusBadRequest, "query parameter 'file' is required")
		return
	}

	content, err := api.db.GetSourceCode(file)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"file": file, "content": content})
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs all requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

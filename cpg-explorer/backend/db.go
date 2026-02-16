package main

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite connection with connection pooling
type DB struct {
	conn *sql.DB
	mu   sync.RWMutex
}

// NewDB creates a new database connection
func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro&cache=shared", dbPath))
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	// Set connection pool settings
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)

	log.Printf("Connected to database: %s", dbPath)

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// GetStats returns database statistics
func (db *DB) GetStats() (map[string]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	stats := make(map[string]interface{})

	// Query stats_overview table (pre-computed)
	row := db.conn.QueryRow(`
		SELECT total_nodes, total_edges, total_files, total_packages,
		       total_functions, total_types, total_metrics
		FROM stats_overview
	`)

	var totalNodes, totalEdges, totalFiles, totalPackages, totalFunctions, totalTypes, totalMetrics int
	err := row.Scan(&totalNodes, &totalEdges, &totalFiles, &totalPackages,
		&totalFunctions, &totalTypes, &totalMetrics)
	if err != nil {
		return nil, fmt.Errorf("query stats: %w", err)
	}

	stats["total_nodes"] = totalNodes
	stats["total_edges"] = totalEdges
	stats["total_files"] = totalFiles
	stats["total_packages"] = totalPackages
	stats["total_functions"] = totalFunctions
	stats["total_types"] = totalTypes
	stats["total_metrics"] = totalMetrics

	return stats, nil
}

// Node represents a CPG node
type Node struct {
	ID             string                 `json:"id"`
	Kind           string                 `json:"kind"`
	Name           string                 `json:"name"`
	File           *string                `json:"file,omitempty"`
	Line           *int                   `json:"line,omitempty"`
	Col            *int                   `json:"col,omitempty"`
	EndLine        *int                   `json:"end_line,omitempty"`
	Package        *string                `json:"package,omitempty"`
	ParentFunction *string                `json:"parent_function,omitempty"`
	TypeInfo       *string                `json:"type_info,omitempty"`
	Properties     map[string]interface{} `json:"properties,omitempty"`
}

// Edge represents a CPG edge
type Edge struct {
	Source     string                 `json:"source"`
	Target     string                 `json:"target"`
	Kind       string                 `json:"kind"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// Graph represents a subgraph with nodes and edges
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Function represents a function with metadata
type Function struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	Package              *string `json:"package,omitempty"`
	File                 *string `json:"file,omitempty"`
	Line                 *int    `json:"line,omitempty"`
	CyclomaticComplexity *int    `json:"cyclomatic_complexity,omitempty"`
	FanIn                *int    `json:"fan_in,omitempty"`
	FanOut               *int    `json:"fan_out,omitempty"`
	LOC                  *int    `json:"loc,omitempty"`
	NumParams            *int    `json:"num_params,omitempty"`
}

// Package represents a package with metadata
type Package struct {
	Name               string  `json:"name"`
	FileCount          int     `json:"file_count"`
	FunctionCount      int     `json:"function_count"`
	TotalLOC           int     `json:"total_loc"`
	TotalComplexity    int     `json:"total_complexity"`
	AvgComplexity      float64 `json:"avg_complexity"`
	MaxComplexity      int     `json:"max_complexity"`
	TypeCount          int     `json:"type_count"`
	InterfaceCount     int     `json:"interface_count"`
}

// PackageEdge represents a dependency between packages
type PackageEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Weight int    `json:"weight"`
}

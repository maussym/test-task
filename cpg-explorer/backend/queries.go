package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// SearchSymbols searches for symbols by name and optionally by kind
func (db *DB) SearchSymbols(query string, kind string, limit int) ([]Node, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	sqlQuery := `
		SELECT id, name, kind, package, file, line
		FROM symbol_index
		WHERE name LIKE ? || '%'
	`
	args := []interface{}{query}

	if kind != "" {
		sqlQuery += ` AND kind = ?`
		args = append(args, kind)
	}

	sqlQuery += ` ORDER BY name LIMIT ?`
	args = append(args, limit)

	rows, err := db.conn.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search symbols: %w", err)
	}
	defer rows.Close()

	var results []Node
	for rows.Next() {
		var n Node
		err := rows.Scan(&n.ID, &n.Name, &n.Kind, &n.Package, &n.File, &n.Line)
		if err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		results = append(results, n)
	}

	return results, nil
}

// GetFunction retrieves detailed information about a function
func (db *DB) GetFunction(functionID string) (*Function, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	sqlQuery := `
		SELECT
			n.id, n.name, n.package, n.file, n.line,
			m.cyclomatic_complexity, m.fan_in, m.fan_out, m.loc, m.num_params
		FROM nodes n
		LEFT JOIN metrics m ON m.function_id = n.id
		WHERE n.id = ? AND n.kind = 'function'
	`

	var f Function
	err := db.conn.QueryRow(sqlQuery, functionID).Scan(
		&f.ID, &f.Name, &f.Package, &f.File, &f.Line,
		&f.CyclomaticComplexity, &f.FanIn, &f.FanOut, &f.LOC, &f.NumParams,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("function not found: %s", functionID)
	}
	if err != nil {
		return nil, fmt.Errorf("get function: %w", err)
	}

	return &f, nil
}

// GetCallGraph retrieves the call graph neighborhood for a function
func (db *DB) GetCallGraph(functionID string, depth int) (*Graph, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if depth <= 0 {
		depth = 2
	}
	if depth > 5 {
		depth = 5
	}

	// Get the function node
	nodes := make(map[string]Node)
	edges := make(map[string]Edge)

	// Add the root function
	rootNode, err := db.getNodeByID(functionID)
	if err != nil {
		return nil, err
	}
	nodes[rootNode.ID] = *rootNode

	// Get callees (functions this function calls)
	callees, err := db.getCallees(functionID, depth)
	if err != nil {
		return nil, err
	}
	for _, n := range callees {
		nodes[n.ID] = n
	}

	// Get callers (functions that call this function)
	callers, err := db.getCallers(functionID, depth)
	if err != nil {
		return nil, err
	}
	for _, n := range callers {
		nodes[n.ID] = n
	}

	// Get all call edges between these nodes
	nodeIDs := make([]string, 0, len(nodes))
	for id := range nodes {
		nodeIDs = append(nodeIDs, id)
	}

	callEdges, err := db.getCallEdges(nodeIDs)
	if err != nil {
		return nil, err
	}
	for _, e := range callEdges {
		key := e.Source + "->" + e.Target
		edges[key] = e
	}

	// Convert maps to slices
	nodeSlice := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		nodeSlice = append(nodeSlice, n)
	}

	edgeSlice := make([]Edge, 0, len(edges))
	for _, e := range edges {
		edgeSlice = append(edgeSlice, e)
	}

	return &Graph{
		Nodes: nodeSlice,
		Edges: edgeSlice,
	}, nil
}

func (db *DB) getNodeByID(nodeID string) (*Node, error) {
	sqlQuery := `
		SELECT id, kind, name, file, line, col, end_line, package, parent_function, type_info, properties
		FROM nodes
		WHERE id = ?
	`

	var n Node
	var propsJSON sql.NullString
	err := db.conn.QueryRow(sqlQuery, nodeID).Scan(
		&n.ID, &n.Kind, &n.Name, &n.File, &n.Line, &n.Col,
		&n.EndLine, &n.Package, &n.ParentFunction, &n.TypeInfo, &propsJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	if propsJSON.Valid && propsJSON.String != "" {
		if err := json.Unmarshal([]byte(propsJSON.String), &n.Properties); err != nil {
			// Ignore JSON parse errors
			n.Properties = nil
		}
	}

	return &n, nil
}

func (db *DB) getCallees(functionID string, depth int) ([]Node, error) {
	sqlQuery := `
		WITH RECURSIVE chain(id, depth) AS (
			SELECT ?, 0
			UNION
			SELECT e.target, c.depth + 1
			FROM chain c
			JOIN edges e ON e.source = c.id
			JOIN nodes n ON n.id = e.target
			WHERE e.kind = 'call' AND c.depth < ? AND n.kind = 'function'
		)
		SELECT DISTINCT n.id, n.kind, n.name, n.file, n.line, n.package
		FROM chain c
		JOIN nodes n ON n.id = c.id
		WHERE n.kind = 'function' AND c.id != ?
		LIMIT 50
	`

	rows, err := db.conn.Query(sqlQuery, functionID, depth, functionID)
	if err != nil {
		return nil, fmt.Errorf("get callees: %w", err)
	}
	defer rows.Close()

	var results []Node
	for rows.Next() {
		var n Node
		err := rows.Scan(&n.ID, &n.Kind, &n.Name, &n.File, &n.Line, &n.Package)
		if err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		results = append(results, n)
	}

	return results, nil
}

func (db *DB) getCallers(functionID string, depth int) ([]Node, error) {
	sqlQuery := `
		WITH RECURSIVE chain(id, depth) AS (
			SELECT ?, 0
			UNION
			SELECT e.source, c.depth + 1
			FROM chain c
			JOIN edges e ON e.target = c.id
			JOIN nodes n ON n.id = e.source
			WHERE e.kind = 'call' AND c.depth < ? AND n.kind = 'function'
		)
		SELECT DISTINCT n.id, n.kind, n.name, n.file, n.line, n.package
		FROM chain c
		JOIN nodes n ON n.id = c.id
		WHERE n.kind = 'function' AND c.id != ?
		LIMIT 50
	`

	rows, err := db.conn.Query(sqlQuery, functionID, depth, functionID)
	if err != nil {
		return nil, fmt.Errorf("get callers: %w", err)
	}
	defer rows.Close()

	var results []Node
	for rows.Next() {
		var n Node
		err := rows.Scan(&n.ID, &n.Kind, &n.Name, &n.File, &n.Line, &n.Package)
		if err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		results = append(results, n)
	}

	return results, nil
}

func (db *DB) getCallEdges(nodeIDs []string) ([]Edge, error) {
	if len(nodeIDs) == 0 {
		return []Edge{}, nil
	}

	// Build parameterized query
	placeholders := ""
	args := make([]interface{}, len(nodeIDs)*2)
	for i, id := range nodeIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
		args[len(nodeIDs)+i] = id
	}

	sqlQuery := fmt.Sprintf(`
		SELECT e.source, e.target, e.kind, e.properties
		FROM edges e
		WHERE e.kind = 'call'
		  AND e.source IN (%s)
		  AND e.target IN (%s)
		LIMIT 200
	`, placeholders, placeholders)

	rows, err := db.conn.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("get call edges: %w", err)
	}
	defer rows.Close()

	var results []Edge
	for rows.Next() {
		var e Edge
		var propsJSON sql.NullString
		err := rows.Scan(&e.Source, &e.Target, &e.Kind, &propsJSON)
		if err != nil {
			return nil, fmt.Errorf("scan edge: %w", err)
		}

		if propsJSON.Valid && propsJSON.String != "" {
			if err := json.Unmarshal([]byte(propsJSON.String), &e.Properties); err != nil {
				e.Properties = nil
			}
		}

		results = append(results, e)
	}

	return results, nil
}

// GetPackages retrieves all packages with their metadata
func (db *DB) GetPackages() ([]Package, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	sqlQuery := `
		SELECT package, file_count, function_count, total_loc, total_complexity,
		       avg_complexity, max_complexity, type_count, interface_count
		FROM dashboard_package_treemap
		ORDER BY function_count DESC
	`

	rows, err := db.conn.Query(sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("get packages: %w", err)
	}
	defer rows.Close()

	var results []Package
	for rows.Next() {
		var p Package
		err := rows.Scan(&p.Name, &p.FileCount, &p.FunctionCount, &p.TotalLOC,
			&p.TotalComplexity, &p.AvgComplexity, &p.MaxComplexity,
			&p.TypeCount, &p.InterfaceCount)
		if err != nil {
			return nil, fmt.Errorf("scan package: %w", err)
		}
		results = append(results, p)
	}

	return results, nil
}

// GetPackageGraph retrieves the package dependency graph
func (db *DB) GetPackageGraph() (*Graph, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Get package nodes from dashboard_package_treemap
	nodeRows, err := db.conn.Query(`
		SELECT package, file_count, function_count, total_loc, total_complexity
		FROM dashboard_package_treemap
		ORDER BY function_count DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("get package nodes: %w", err)
	}
	defer nodeRows.Close()

	nodes := make([]Node, 0)
	for nodeRows.Next() {
		var name string
		var fileCount, funcCount, totalLoc, totalComplexity int
		err := nodeRows.Scan(&name, &fileCount, &funcCount, &totalLoc, &totalComplexity)
		if err != nil {
			return nil, fmt.Errorf("scan package node: %w", err)
		}

		nodes = append(nodes, Node{
			ID:      "pkg::" + name,
			Kind:    "package",
			Name:    name,
			Package: &name,
			Properties: map[string]interface{}{
				"file_count":   fileCount,
				"function_count": funcCount,
				"total_loc":    totalLoc,
				"total_complexity": totalComplexity,
			},
		})
	}

	// Get package edges from dashboard_package_graph
	edgeRows, err := db.conn.Query(`
		SELECT source, target, weight
		FROM dashboard_package_graph
		ORDER BY weight DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("get package edges: %w", err)
	}
	defer edgeRows.Close()

	edges := make([]Edge, 0)
	for edgeRows.Next() {
		var source, target string
		var weight int
		err := edgeRows.Scan(&source, &target, &weight)
		if err != nil {
			return nil, fmt.Errorf("scan package edge: %w", err)
		}

		edges = append(edges, Edge{
			Source: "pkg::" + source,
			Target: "pkg::" + target,
			Kind:   "depends",
			Properties: map[string]interface{}{
				"weight": weight,
			},
		})
	}

	return &Graph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// GetPackageFunctions retrieves top functions in a package
func (db *DB) GetPackageFunctions(packageName string, limit int) ([]Function, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	sqlQuery := `
		SELECT n.id, n.name, n.package, n.file, n.line,
		       m.cyclomatic_complexity, m.fan_in, m.fan_out, m.loc, m.num_params
		FROM nodes n
		LEFT JOIN metrics m ON m.function_id = n.id
		WHERE n.kind = 'function' AND n.package = ?
		ORDER BY m.cyclomatic_complexity DESC, m.loc DESC
		LIMIT ?
	`

	rows, err := db.conn.Query(sqlQuery, packageName, limit)
	if err != nil {
		return nil, fmt.Errorf("get package functions: %w", err)
	}
	defer rows.Close()

	var results []Function
	for rows.Next() {
		var f Function
		err := rows.Scan(&f.ID, &f.Name, &f.Package, &f.File, &f.Line,
			&f.CyclomaticComplexity, &f.FanIn, &f.FanOut, &f.LOC, &f.NumParams)
		if err != nil {
			return nil, fmt.Errorf("scan function: %w", err)
		}
		results = append(results, f)
	}

	return results, nil
}

// GetSourceCode retrieves source code for a file
func (db *DB) GetSourceCode(file string) (string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var content string
	err := db.conn.QueryRow(`SELECT content FROM sources WHERE file = ?`, file).Scan(&content)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("file not found: %s", file)
	}
	if err != nil {
		return "", fmt.Errorf("get source code: %w", err)
	}

	return content, nil
}

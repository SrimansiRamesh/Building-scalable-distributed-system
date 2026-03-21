package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Allowed table names for SELECT * (whitelist to avoid SQL injection).
var allowedTables = map[string]bool{
	"products":             true,
	"shopping_carts":       true,
	"shopping_cart_items":  true,
}

// getTableRows returns all rows from a given table (SELECT * FROM table).
// Used to dump table data when DB is available (e.g. for backup or when DB is not accessible elsewhere).
func getTableRows(c *gin.Context) {
	if appDB == nil {
		respondError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Database is not available", "")
		return
	}
	tableName := strings.TrimSpace(c.Param("tableName"))
	if tableName == "" {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Table name is required", "")
		return
	}
	if !allowedTables[tableName] {
		respondError(c, http.StatusBadRequest, "INVALID_TABLE", "Table not allowed", "Only products, shopping_carts, shopping_cart_items are allowed")
		return
	}

	query := "SELECT * FROM " + tableName
	rows, err := appDB.Query(query)
	if err != nil {
		log.Printf("getTableRows %q: %v", tableName, err)
		respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", err.Error())
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		log.Printf("getTableRows columns: %v", err)
		respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", err.Error())
		return
	}

	var result []map[string]interface{}
	valuePtrs := make([]interface{}, len(columns))
	values := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Printf("getTableRows scan: %v", err)
			respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", err.Error())
			return
		}
		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			v := values[i]
			if v == nil {
				row[col] = nil
				continue
			}
			switch x := v.(type) {
			case []byte:
				row[col] = string(x)
			default:
				row[col] = x
			}
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		log.Printf("getTableRows rows: %v", err)
		respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", err.Error())
		return
	}
	if result == nil {
		result = []map[string]interface{}{}
	}
	c.IndentedJSON(http.StatusOK, gin.H{"table": tableName, "rows": result})
}

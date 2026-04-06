package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetRequest is the JSON body for a set (write) request from a client.
type SetRequest struct {
	Value string `json:"value"`
}

// InternalSetRequest is used for node-to-node replication messages.
// Includes the version number so receiving nodes can detect staleness.
type InternalSetRequest struct {
	Value   string `json:"value"`
	Version int64  `json:"version"`
}

// KVResponse is the standard response for a get request.
type KVResponse struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version int64  `json:"version"`
}

// registerRoutes sets up all HTTP routes on the gin router.
// Routes are split into:
//   - Client-facing: /kv/:key (GET and PUT)
//   - Internal: /internal/kv/:key (node-to-node replication)
//   - Debug: /local/:key (raw local read, bypasses coordination)
//   - Health: /health
func registerRoutes(r *gin.Engine, node *Node) {
	// Health check — used by ALB and ECS
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "node_id": node.ID, "role": node.Role})
	})

	// Client-facing endpoints
	r.PUT("/kv/:key", func(c *gin.Context) {
		key := c.Param("key")
		if key == "" || key == "/" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "key cannot be empty"})
			return
		}

		var req SetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		if err := node.HandleSet(key, req.Value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusCreated)
	})

	r.GET("/kv/:key", func(c *gin.Context) {
		key := c.Param("key")
		val, version, found := node.HandleGet(key)
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
			return
		}
		c.JSON(http.StatusOK, KVResponse{Key: key, Value: val, Version: version})
	})

	// Internal replication endpoint — called by leader/coordinator to replicate writes
	// Followers sleep 100ms before writing to simulate storage delay
	r.PUT("/internal/kv/:key", func(c *gin.Context) {
		key := c.Param("key")
		var req InternalSetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		node.HandleInternalSet(key, req.Value, req.Version)
		c.Status(http.StatusOK)
	})

	// Internal read endpoint — called by leader when doing R>1 reads
	// Followers sleep 50ms before responding to simulate read delay
	r.GET("/internal/kv/:key", func(c *gin.Context) {
		key := c.Param("key")
		val, version, found := node.HandleInternalGet(key)
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
			return
		}
		c.JSON(http.StatusOK, KVResponse{Key: key, Value: val, Version: version})
	})

	// local_read — sneaky debug endpoint, bypasses all coordination
	// Returns raw local value immediately with no delays
	// Used in unit tests to observe inconsistency windows
	r.GET("/local/:key", func(c *gin.Context) {
		key := c.Param("key")
		entry, ok := node.Store.LocalGet(key)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
			return
		}
		c.JSON(http.StatusOK, KVResponse{Key: key, Value: entry.Value, Version: entry.Version})
	})
}
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Node represents a single node in the distributed KV cluster.
// The same binary runs on all 5 nodes — behavior is controlled by env vars.
type Node struct {
	ID    string // e.g. "node-0", "node-1"
	Role  string // "leader", "follower", or "leaderless"
	Peers []string // HTTP addresses of all OTHER nodes, e.g. ["http://node1:8080", ...]
	W     int    // write quorum
	R     int    // read quorum
	Store *Store
}

// HandleSet is the main dispatch for client write requests.
// Routes to the correct implementation based on role.
func (n *Node) HandleSet(key, value string) error {
	switch n.Role {
	case "leader":
		return n.leaderSet(key, value)
	case "leaderless":
		return n.leaderlessSet(key, value)
	case "follower":
		// Followers should not receive direct client writes in leader-follower mode.
		// If they do, return an error directing client to the leader.
		return fmt.Errorf("this node is a follower — send writes to the leader")
	default:
		return fmt.Errorf("unknown role: %s", n.Role)
	}
}

// HandleGet is the main dispatch for client read requests.
func (n *Node) HandleGet(key string) (string, int64, bool) {
	switch n.Role {
	case "leader":
		return n.leaderGet(key)
	case "leaderless":
		return n.leaderlessGet(key)
	case "follower":
		// Followers can serve reads directly from their local store
		// This is how clients read from followers in leader-follower mode
		entry, ok := n.Store.LocalGet(key)
		if !ok {
			return "", 0, false
		}
		return entry.Value, entry.Version, true
	default:
		return "", 0, false
	}
}

// HandleInternalSet is called when THIS node receives a replication message
// from a leader or write coordinator. Simulates follower write delay.
func (n *Node) HandleInternalSet(key, value string, version int64) {
	// Assignment requirement: follower sleeps 100ms before writing
	time.Sleep(100 * time.Millisecond)
	n.Store.LocalSet(key, value, version)
}

// HandleInternalGet is called when THIS node receives an internal read
// from the leader during an R>1 read. Simulates follower read delay.
func (n *Node) HandleInternalGet(key string) (string, int64, bool) {
	// Assignment requirement: follower sleeps 50ms before responding to read
	time.Sleep(50 * time.Millisecond)
	entry, ok := n.Store.LocalGet(key)
	if !ok {
		return "", 0, false
	}
	return entry.Value, entry.Version, true
}

// mustNewRequest creates an HTTP request with JSON content-type header.
// Panics on error since this should never fail with valid inputs.
func mustNewRequest(method, url string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}

func main() {
	// Read configuration from environment variables
	// This is how the same Docker image runs as leader, follower, or leaderless
	nodeID := getEnv("NODE_ID", "node-0")
	role := getEnv("ROLE", "leader")
	peersRaw := getEnv("PEERS", "")
	wStr := getEnv("W", "5")
	rStr := getEnv("R", "1")
	port := getEnv("PORT", "8080")

	w, err := strconv.Atoi(wStr)
	if err != nil {
		log.Fatalf("invalid W value: %s", wStr)
	}
	r, err := strconv.Atoi(rStr)
	if err != nil {
		log.Fatalf("invalid R value: %s", rStr)
	}

	// Parse peer list — comma-separated HTTP addresses of other nodes
	var peers []string
	if peersRaw != "" {
		for _, p := range strings.Split(peersRaw, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				peers = append(peers, p)
			}
		}
	}

	node := &Node{
		ID:    nodeID,
		Role:  role,
		Peers: peers,
		W:     w,
		R:     r,
		Store: NewStore(),
	}

	log.Printf("Starting node %s | role=%s | W=%d | R=%d | peers=%v",
		nodeID, role, w, r, peers)

	// Set up HTTP server using gin (same as previous assignments)
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	registerRoutes(router, node)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// getEnv reads an env var with a fallback default value.
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// replicateToPeerWithBody is a helper used by both leader and leaderless.
func replicateToPeerWithBody(url string, body []byte) error {
	resp, err := http.DefaultClient.Do(mustNewRequest(
		http.MethodPut, url, bytes.NewBuffer(body),
	))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
	return nil
}
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// KVResponse matches the response from the node API
type KVResponse struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version int64  `json:"version"`
}

// These addresses are set via env vars so tests work against real deployed nodes
// e.g. LEADER_ADDR=http://leader-alb:8080
var (
	leaderAddr    = getAddr("LEADER_ADDR", "http://localhost:8080")
	follower1Addr = getAddr("FOLLOWER1_ADDR", "http://localhost:8081")
	follower2Addr = getAddr("FOLLOWER2_ADDR", "http://localhost:8082")
	leaderless0   = getAddr("LEADERLESS0_ADDR", "http://localhost:9080")
	leaderless1   = getAddr("LEADERLESS1_ADDR", "http://localhost:9081")
	leaderless2   = getAddr("LEADERLESS2_ADDR", "http://localhost:9082")
)

func getAddr(env, def string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	return def
}

// kvSet sends a PUT /kv/:key to a node
func kvSet(addr, key, value string) error {
	body, _ := json.Marshal(map[string]string{"value": value})
	req, _ := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/kv/%s", addr, key), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("expected 201 got %d", resp.StatusCode)
	}
	return nil
}

// kvGet sends a GET /kv/:key to a node
func kvGet(addr, key string) (*KVResponse, error) {
	resp, err := http.Get(fmt.Sprintf("%s/kv/%s", addr, key))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	var r KVResponse
	json.NewDecoder(resp.Body).Decode(&r)
	return &r, nil
}

// localGet sends a GET /local/:key — bypasses coordination, raw local value
func localGet(addr, key string) (*KVResponse, error) {
	resp, err := http.Get(fmt.Sprintf("%s/local/%s", addr, key))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	var r KVResponse
	json.NewDecoder(resp.Body).Decode(&r)
	return &r, nil
}

// =============================================================================
// Leader-Follower Tests
// =============================================================================

// TestLeaderReadConsistency:
// Write to leader → read from leader → must be consistent
func TestLeaderReadConsistency(t *testing.T) {
	key := fmt.Sprintf("test-leader-%d", time.Now().UnixNano())
	value := "hello-from-leader"

	if err := kvSet(leaderAddr, key, value); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	r, err := kvGet(leaderAddr, key)
	if err != nil || r == nil {
		t.Fatalf("get from leader failed: %v", err)
	}
	if r.Value != value {
		t.Errorf("expected %q got %q", value, r.Value)
	}
	t.Logf("Leader read consistent: version=%d value=%s", r.Version, r.Value)
}

// TestFollowerReadConsistency:
// Write to leader → wait for replication → read from follower → must be consistent
func TestFollowerReadConsistency(t *testing.T) {
	key := fmt.Sprintf("test-follower-%d", time.Now().UnixNano())
	value := "replicated-value"

	if err := kvSet(leaderAddr, key, value); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	// Wait for replication to complete (W=5 config: leader + 4 followers)
	// Each follower takes 100ms + 200ms leader delay = ~300ms per follower
	time.Sleep(500 * time.Millisecond)

	r, err := kvGet(follower1Addr, key)
	if err != nil || r == nil {
		t.Fatalf("get from follower failed: %v", err)
	}
	if r.Value != value {
		t.Errorf("expected %q got %q", value, r.Value)
	}
	t.Logf("Follower read consistent after replication: version=%d", r.Version)
}

// TestFollowerInconsistencyWindow:
// Write to leader → immediately local_read from followers → may show stale data
// This proves the inconsistency window exists during replication
func TestFollowerInconsistencyWindow(t *testing.T) {
	key := fmt.Sprintf("test-window-%d", time.Now().UnixNano())
	value := "new-value"

	staleCount := 0
	totalRuns := 10

	for i := 0; i < totalRuns; i++ {
		uniqueKey := fmt.Sprintf("%s-%d", key, i)
		uniqueVal := fmt.Sprintf("%s-%d", value, i)

		// Send write to leader — don't wait for it to complete (async)
		go kvSet(leaderAddr, uniqueKey, uniqueVal)

		// Immediately read from followers via local_read
		// This should frequently return "not found" or stale data
		time.Sleep(50 * time.Millisecond) // small delay so write has started but not replicated

		r, _ := localGet(follower1Addr, uniqueKey)
		if r == nil || r.Value != uniqueVal {
			staleCount++
			t.Logf("Run %d: Caught inconsistency window — follower has stale/missing data", i)
		}

		time.Sleep(400 * time.Millisecond) // wait for replication before next run
	}

	t.Logf("Inconsistency window observed in %d/%d runs", staleCount, totalRuns)
	if staleCount == 0 {
		t.Log("WARNING: No inconsistency observed — try increasing load or reducing W")
	}
}

// =============================================================================
// Leaderless Tests
// =============================================================================

// TestLeaderlessWriteCoordinatorConsistency:
// Write to node 0 → read from node 0 after write completes → must be consistent
func TestLeaderlessWriteCoordinatorConsistency(t *testing.T) {
	key := fmt.Sprintf("test-leaderless-%d", time.Now().UnixNano())
	value := "leaderless-value"

	if err := kvSet(leaderless0, key, value); err != nil {
		t.Fatalf("leaderless set failed: %v", err)
	}

	// Read from coordinator — must be consistent (W=N means all nodes updated)
	r, err := kvGet(leaderless0, key)
	if err != nil || r == nil {
		t.Fatalf("get from coordinator failed: %v", err)
	}
	if r.Value != value {
		t.Errorf("coordinator inconsistent: expected %q got %q", value, r.Value)
	}
	t.Logf("Leaderless coordinator read consistent: version=%d", r.Version)

	// Read from another node — also must be consistent (W=N waited for all)
	r2, err := kvGet(leaderless1, key)
	if err != nil || r2 == nil {
		t.Fatalf("get from other leaderless node failed: %v", err)
	}
	if r2.Value != value {
		t.Errorf("other node inconsistent: expected %q got %q", value, r2.Value)
	}
	t.Logf("Leaderless other node read consistent: version=%d", r2.Version)
}

// TestLeaderlessInconsistencyWindow:
// Write to node 0 → during replication, read from node 1 → should show stale data
// This is the core demonstration of the leaderless inconsistency window
func TestLeaderlessInconsistencyWindow(t *testing.T) {
	key := fmt.Sprintf("test-leaderless-window-%d", time.Now().UnixNano())
	value := "coordinator-only-so-far"

	staleCount := 0
	totalRuns := 10

	for i := 0; i < totalRuns; i++ {
		uniqueKey := fmt.Sprintf("%s-%d", key, i)
		uniqueVal := fmt.Sprintf("%s-%d", value, i)

		// Send write to node 0 (becomes coordinator) — don't wait
		go kvSet(leaderless0, uniqueKey, uniqueVal)

		// Immediately read from node 1 — replication hasn't finished yet
		// R=1 so node 1 just returns its own local value
		time.Sleep(30 * time.Millisecond)

		r, _ := localGet(leaderless1, uniqueKey)
		if r == nil || r.Value != uniqueVal {
			staleCount++
			t.Logf("Run %d: Caught leaderless inconsistency — node 1 has stale data", i)
		}

		time.Sleep(600 * time.Millisecond) // wait for full replication before next run
	}

	t.Logf("Leaderless inconsistency window observed in %d/%d runs", staleCount, totalRuns)
}
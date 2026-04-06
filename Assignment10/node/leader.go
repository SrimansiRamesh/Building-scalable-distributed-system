package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

// leaderSet handles a write request arriving at the leader node.
// Behavior depends on W (write quorum):
//   W=1: write locally, return immediately, replicate async
//   W=3: write locally + wait for 2 followers to confirm (quorum)
//   W=5: write locally + wait for ALL 4 followers to confirm
func (n *Node) leaderSet(key, value string) error {
	version := n.Store.NextVersion(key)
	// Always write to leader first
	n.Store.LocalSet(key, value, version)

	needed := n.W - 1 // how many followers we need to confirm (-1 for leader itself)
	if needed <= 0 {
		// W=1: fire-and-forget replication to all followers asynchronously
		go n.replicateToAll(key, value, version)
		return nil
	}

	// W>1: need to wait for 'needed' followers to confirm before responding
	// Use a channel to collect results from concurrent goroutines
	results := make(chan error, len(n.Peers))

	for _, peer := range n.Peers {
		go func(peer string) {
			results <- n.replicateToPeer(peer, key, value, version)
		}(peer)
	}

	// Collect results — return as soon as we have enough confirmations
	confirmed := 0
	errors := 0
	for i := 0; i < len(n.Peers); i++ {
		err := <-results
		if err == nil {
			confirmed++
			if confirmed >= needed {
				// Got enough — don't need to wait for remaining peers
				// Remaining goroutines will finish in background
				return nil
			}
		} else {
			errors++
			// If too many failures make quorum impossible, fail fast
			remaining := len(n.Peers) - i - 1
			if confirmed+remaining < needed {
				return fmt.Errorf("not enough nodes available for W=%d", n.W)
			}
		}
	}
	return nil
}

// replicateToPeer sends a single replication message to one follower.
// The leader sleeps 200ms after sending each message (as required by assignment).
// The follower will sleep 100ms before responding (handled on follower side).
func (n *Node) replicateToPeer(peer, key, value string, version int64) error {
	body, _ := json.Marshal(InternalSetRequest{Value: value, Version: version})
	url := fmt.Sprintf("%s/internal/kv/%s", peer, key)

	resp, err := http.DefaultClient.Do(mustNewRequest(http.MethodPut, url, bytes.NewBuffer(body)))
	if err != nil {
		return err
	}
	resp.Body.Close()

	// Assignment requirement: leader sleeps 200ms after each follower message
	time.Sleep(200 * time.Millisecond)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer %s returned %d", peer, resp.StatusCode)
	}
	return nil
}

// replicateToAll sends replication to all peers concurrently (fire-and-forget).
// Used for W=1 where we don't wait for confirmations.
func (n *Node) replicateToAll(key, value string, version int64) {
	var wg sync.WaitGroup
	for _, peer := range n.Peers {
		wg.Add(1)
		go func(peer string) {
			defer wg.Done()
			n.replicateToPeer(peer, key, value, version)
		}(peer)
	}
	wg.Wait()
}

// leaderGet handles a read request arriving at the leader node.
// Behavior depends on R (read quorum):
//   R=1: return leader's own value immediately (no follower contact)
//   R=3: fetch from leader + 2 followers, return most recent version
//   R=5: fetch from all 5 nodes, return most recent version
func (n *Node) leaderGet(key string) (string, int64, bool) {
	if n.R == 1 {
		// R=1: just return what the leader has
		entry, ok := n.Store.LocalGet(key)
		if !ok {
			return "", 0, false
		}
		return entry.Value, entry.Version, true
	}

	// R>1: need to fetch from multiple nodes and return most recent
	// Collect from leader + followers concurrently
	type result struct {
		entry Entry
		found bool
	}

	results := make(chan result, len(n.Peers)+1)

	// Include leader's own value
	go func() {
		entry, ok := n.Store.LocalGet(key)
		results <- result{entry, ok}
	}()

	// Fetch from all followers concurrently
	for _, peer := range n.Peers {
		go func(peer string) {
			entry, found := n.fetchFromPeer(peer, key)
			results <- result{entry, found}
		}(peer)
	}

	// Collect R results and return the one with highest version
	needed := n.R
	collected := 0
	var best Entry
	bestFound := false

	for i := 0; i < len(n.Peers)+1 && collected < needed; i++ {
		r := <-results
		collected++
		if r.found {
			if !bestFound || r.entry.Version > best.Version {
				best = r.entry
				bestFound = true
			}
		}
	}

	if !bestFound {
		return "", 0, false
	}
	return best.Value, best.Version, true
}

// fetchFromPeer fetches a value from a follower's internal read endpoint.
// Followers sleep 50ms before responding (handled on follower side).
func (n *Node) fetchFromPeer(peer, key string) (Entry, bool) {
	url := fmt.Sprintf("%s/internal/kv/%s", peer, key)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode == http.StatusNotFound {
		return Entry{}, false
	}
	defer resp.Body.Close()

	var kv KVResponse
	if err := json.NewDecoder(resp.Body).Decode(&kv); err != nil {
		return Entry{}, false
	}
	return Entry{Value: kv.Value, Version: kv.Version}, true
}

// mostRecent returns the entry with the highest version from a slice.
// Used for R>1 reads where we need to pick the most up-to-date value.
func mostRecent(entries []Entry) (Entry, bool) {
	if len(entries) == 0 {
		return Entry{}, false
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Version > entries[j].Version
	})
	return entries[0], true
}
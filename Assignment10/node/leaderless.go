package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// leaderlessSet handles a write arriving at any node in leaderless mode.
// The receiving node becomes the Write Coordinator for this write.
// W=N means ALL nodes must be updated before returning 201 to the client.
//
// Flow:
//  1. Write locally first
//  2. Send replication to ALL other nodes
//  3. Wait for ALL to confirm
//  4. Only then return success to client
//
// This creates an intentional inconsistency window:
// between step 1 and step 3, other nodes don't have the data yet.
// A concurrent read to one of those nodes will return stale data.
func (n *Node) leaderlessSet(key, value string) error {
	version := n.Store.NextVersion(key)
	// Write locally first — this node is now ahead of the others
	n.Store.LocalSet(key, value, version)

	// Replicate to ALL peers and wait for all to confirm (W=N)
	errCh := make(chan error, len(n.Peers))
	for _, peer := range n.Peers {
		go func(peer string) {
			errCh <- n.leaderlessReplicateToPeer(peer, key, value, version)
		}(peer)
	}

	// Collect ALL results — must wait for every node
	var errs []error
	for i := 0; i < len(n.Peers); i++ {
		if err := <-errCh; err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("replication failed to %d nodes", len(errs))
	}
	return nil
}

// leaderlessReplicateToPeer sends a replication message to one peer.
// Same 200ms delay as leader-follower replication.
func (n *Node) leaderlessReplicateToPeer(peer, key, value string, version int64) error {
	body, _ := json.Marshal(InternalSetRequest{Value: value, Version: version})
	url := fmt.Sprintf("%s/internal/kv/%s", peer, key)

	resp, err := http.DefaultClient.Do(mustNewRequest(http.MethodPut, url, bytes.NewBuffer(body)))
	if err != nil {
		return err
	}
	resp.Body.Close()

	// Assignment requirement: coordinator sleeps 200ms after each peer message
	time.Sleep(200 * time.Millisecond)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer %s returned %d", peer, resp.StatusCode)
	}
	return nil
}

// leaderlessGet handles a read arriving at any node in leaderless mode.
// R=1 means just return this node's own local value immediately.
// This is what creates the inconsistency window —
// if this node hasn't received the replication yet, it returns stale data.
func (n *Node) leaderlessGet(key string) (string, int64, bool) {
	entry, ok := n.Store.LocalGet(key)
	if !ok {
		return "", 0, false
	}
	return entry.Value, entry.Version, true
}
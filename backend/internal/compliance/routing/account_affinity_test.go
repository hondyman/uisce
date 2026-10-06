package routing

import (
	"testing"

	"github.com/google/uuid"
)

func TestConsistentHashRouter_DeterministicMapping(t *testing.T) {
	router := NewConsistentHashRouter(100)
	router.AddNode("gateway-pod-1")
	router.AddNode("gateway-pod-2")
	router.AddNode("gateway-pod-3")

	accountA := uuid.MustParse("018f2d5e-7a42-7000-8000-000000000001")
	accountB := uuid.MustParse("018f2d5e-7a42-7000-8000-000000000002")

	nodeA1, err := router.RouteAccount(accountA)
	if err != nil {
		t.Fatalf("RouteAccount failed: %v", err)
	}

	nodeB1, err := router.RouteAccount(accountB)
	if err != nil {
		t.Fatalf("RouteAccount failed: %v", err)
	}

	// Re-route 10,000 times: must deterministically return identical nodes
	for i := 0; i < 10000; i++ {
		nodeA, _ := router.RouteAccount(accountA)
		nodeB, _ := router.RouteAccount(accountB)
		if nodeA != nodeA1 {
			t.Fatalf("Non-deterministic routing for Account A on iteration %d: got %s, want %s", i, nodeA, nodeA1)
		}
		if nodeB != nodeB1 {
			t.Fatalf("Non-deterministic routing for Account B on iteration %d: got %s, want %s", i, nodeB, nodeB1)
		}
	}
}

func TestConsistentHashRouter_DistributionAndRebalance(t *testing.T) {
	router := NewConsistentHashRouter(100)
	nodes := []string{"pod-1", "pod-2", "pod-3"}
	for _, n := range nodes {
		router.AddNode(n)
	}

	// Generate 1,000 accounts and track routing
	numAccounts := 1000
	initialAssignments := make(map[uuid.UUID]string, numAccounts)
	nodeCounts := make(map[string]int)

	for i := 0; i < numAccounts; i++ {
		accID := uuid.New()
		node, err := router.RouteAccount(accID)
		if err != nil {
			t.Fatalf("RouteAccount error: %v", err)
		}
		initialAssignments[accID] = node
		nodeCounts[node]++
	}

	for node, count := range nodeCounts {
		t.Logf("Initial Node %s owns %d accounts (%.1f%%)", node, count, float64(count)/float64(numAccounts)*100)
		if count == 0 {
			t.Errorf("Node %s received 0 accounts", node)
		}
	}

	// Scale up: Add pod-4
	router.AddNode("pod-4")
	migrated := 0
	for accID, oldNode := range initialAssignments {
		newNode, _ := router.RouteAccount(accID)
		if newNode != oldNode {
			migrated++
			if newNode != "pod-4" {
				t.Errorf("Account migrated to existing node %s instead of new node pod-4", newNode)
			}
		}
	}

	// Bound migration: adding 1 node to 3 should migrate ~25% of keys
	migrationPct := float64(migrated) / float64(numAccounts) * 100
	t.Logf("Migration upon adding pod-4: %d accounts (%.1f%%)", migrated, migrationPct)
	if migrationPct < 15 || migrationPct > 35 {
		t.Errorf("Migration percentage %.1f%% outside expected bound [15%%, 35%%]", migrationPct)
	}
}

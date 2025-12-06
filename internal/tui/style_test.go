package tui

import (
	"testing"
	"time"

	"github.com/bit2swaz/crisismesh/internal/store"
)

func TestFlashLogic(t *testing.T) {
	// Test 1: Now should flash
	if !ShouldFlash(time.Now()) {
		t.Error("Expected ShouldFlash(Now) to be true")
	}

	// Test 2: Old message should not flash
	oldTime := time.Now().Add(-2 * time.Second)
	if ShouldFlash(oldTime) {
		t.Error("Expected ShouldFlash(OldTime) to be false")
	}
}

func TestPeerSorting(t *testing.T) {
	peers := []store.Peer{
		{Nick: "Zebra", IsActive: true},
		{Nick: "Alpha", IsActive: false},
		{Nick: "Beta", IsActive: true},
	}

	sortPeers(peers)

	// Expected order:
	// 1. Beta (Active, B < Z)
	// 2. Zebra (Active)
	// 3. Alpha (Inactive)

	if peers[0].Nick != "Beta" {
		t.Errorf("Expected first peer to be Beta, got %s", peers[0].Nick)
	}
	if peers[1].Nick != "Zebra" {
		t.Errorf("Expected second peer to be Zebra, got %s", peers[1].Nick)
	}
	if peers[2].Nick != "Alpha" {
		t.Errorf("Expected third peer to be Alpha, got %s", peers[2].Nick)
	}
}

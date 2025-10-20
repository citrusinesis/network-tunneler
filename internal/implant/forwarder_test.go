package implant

import (
	"testing"
	"time"

	testutil "network-tunneler/internal/testing"
)

func TestPacketForwarder_Count(t *testing.T) {
	log := testutil.NewTestLogger()
	forwarder := NewPacketForwarder(ForwarderParams{Logger: log})

	if forwarder.Count() != 0 {
		t.Errorf("expected 0 connections, got %d", forwarder.Count())
	}
}

func TestPacketForwarder_RemoveConnection(t *testing.T) {
	log := testutil.NewTestLogger()
	forwarder := NewPacketForwarder(ForwarderParams{Logger: log})

	mockConn := testutil.NewMockNetConn()

	state := &ConnectionState{
		ConnectionID: "test-conn-1",
		TargetAddr:   "192.168.1.1:80",
		TargetConn:   mockConn,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	forwarder.connections["test-conn-1"] = state

	if forwarder.Count() != 1 {
		t.Errorf("expected 1 connection, got %d", forwarder.Count())
	}

	forwarder.removeConnection("test-conn-1")

	if forwarder.Count() != 0 {
		t.Errorf("expected 0 connections after removal, got %d", forwarder.Count())
	}

	if !mockConn.Closed {
		t.Error("expected connection to be closed")
	}
}

func TestPacketForwarder_Cleanup(t *testing.T) {
	log := testutil.NewTestLogger()
	forwarder := NewPacketForwarder(ForwarderParams{Logger: log})

	mockConn1 := testutil.NewMockNetConn()
	mockConn2 := testutil.NewMockNetConn()
	mockConn3 := testutil.NewMockNetConn()

	now := time.Now()

	forwarder.connections["conn-1"] = &ConnectionState{
		ConnectionID: "conn-1",
		TargetAddr:   "192.168.1.1:80",
		TargetConn:   mockConn1,
		CreatedAt:    now.Add(-10 * time.Minute),
		LastActivity: now.Add(-10 * time.Minute),
	}

	forwarder.connections["conn-2"] = &ConnectionState{
		ConnectionID: "conn-2",
		TargetAddr:   "192.168.1.2:80",
		TargetConn:   mockConn2,
		CreatedAt:    now.Add(-8 * time.Minute),
		LastActivity: now.Add(-8 * time.Minute),
	}

	forwarder.connections["conn-3"] = &ConnectionState{
		ConnectionID: "conn-3",
		TargetAddr:   "192.168.1.3:80",
		TargetConn:   mockConn3,
		CreatedAt:    now,
		LastActivity: now,
	}

	removed := forwarder.Cleanup(5 * time.Minute)

	if removed != 2 {
		t.Errorf("expected 2 connections to be removed, got %d", removed)
	}

	if forwarder.Count() != 1 {
		t.Errorf("expected 1 connection remaining, got %d", forwarder.Count())
	}

	if !mockConn1.Closed {
		t.Error("expected conn-1 to be closed")
	}

	if !mockConn2.Closed {
		t.Error("expected conn-2 to be closed")
	}

	if mockConn3.Closed {
		t.Error("expected conn-3 to remain open")
	}

	_, exists := forwarder.connections["conn-3"]
	if !exists {
		t.Error("expected conn-3 to still exist")
	}
}

func TestPacketForwarder_CleanupNoStale(t *testing.T) {
	log := testutil.NewTestLogger()
	forwarder := NewPacketForwarder(ForwarderParams{Logger: log})

	mockConn := testutil.NewMockNetConn()

	forwarder.connections["conn-1"] = &ConnectionState{
		ConnectionID: "conn-1",
		TargetAddr:   "192.168.1.1:80",
		TargetConn:   mockConn,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	removed := forwarder.Cleanup(5 * time.Minute)

	if removed != 0 {
		t.Errorf("expected 0 connections to be removed, got %d", removed)
	}

	if forwarder.Count() != 1 {
		t.Errorf("expected 1 connection remaining, got %d", forwarder.Count())
	}

	if mockConn.Closed {
		t.Error("expected connection to remain open")
	}
}

func TestPacketForwarder_MultipleCleanupCycles(t *testing.T) {
	log := testutil.NewTestLogger()
	forwarder := NewPacketForwarder(ForwarderParams{Logger: log})

	now := time.Now()

	for i := 0; i < 5; i++ {
		mockConn := testutil.NewMockNetConn()
		forwarder.connections[string(rune(i))] = &ConnectionState{
			ConnectionID: string(rune(i)),
			TargetAddr:   "192.168.1.1:80",
			TargetConn:   mockConn,
			CreatedAt:    now.Add(-10 * time.Minute),
			LastActivity: now.Add(-10 * time.Minute),
		}
	}

	removed := forwarder.Cleanup(5 * time.Minute)
	if removed != 5 {
		t.Errorf("first cleanup: expected 5 connections removed, got %d", removed)
	}

	removed = forwarder.Cleanup(5 * time.Minute)
	if removed != 0 {
		t.Errorf("second cleanup: expected 0 connections removed, got %d", removed)
	}
}

func BenchmarkPacketForwarder_Count(b *testing.B) {
	log := testutil.NewTestLogger()
	forwarder := NewPacketForwarder(ForwarderParams{Logger: log})

	for i := 0; i < 100; i++ {
		mockConn := testutil.NewMockNetConn()
		forwarder.connections[string(rune(i))] = &ConnectionState{
			ConnectionID: string(rune(i)),
			TargetAddr:   "192.168.1.1:80",
			TargetConn:   mockConn,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		forwarder.Count()
	}
}

func BenchmarkPacketForwarder_Cleanup(b *testing.B) {
	log := testutil.NewTestLogger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		forwarder := NewPacketForwarder(ForwarderParams{Logger: log})
		now := time.Now()

		for j := 0; j < 100; j++ {
			mockConn := testutil.NewMockNetConn()
			age := time.Duration(j) * time.Minute
			forwarder.connections[string(rune(j))] = &ConnectionState{
				ConnectionID: string(rune(j)),
				TargetAddr:   "192.168.1.1:80",
				TargetConn:   mockConn,
				CreatedAt:    now.Add(-age),
				LastActivity: now.Add(-age),
			}
		}
		b.StartTimer()

		forwarder.Cleanup(5 * time.Minute)
	}
}

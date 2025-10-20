package agent

import (
	"testing"
	"time"

	testutil "network-tunneler/internal/testing"
)

func TestConnectionTracker_Track(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	mockConn := testutil.NewMockNetConn()
	connID := "test-conn-1"
	originalDest := "192.168.1.1:80"

	tracker.Track(connID, originalDest, mockConn)

	state, exists := tracker.Get(connID)
	if !exists {
		t.Fatal("expected connection to be tracked")
	}

	if state.ConnectionID != connID {
		t.Errorf("expected connection ID %s, got %s", connID, state.ConnectionID)
	}

	if state.OriginalDest != originalDest {
		t.Errorf("expected original dest %s, got %s", originalDest, state.OriginalDest)
	}

	if state.LocalConn != mockConn {
		t.Error("expected local connection to match")
	}

	if state.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if state.LastActivity.IsZero() {
		t.Error("expected LastActivity to be set")
	}
}

func TestConnectionTracker_Get(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	mockConn := testutil.NewMockNetConn()
	connID := "test-conn-1"

	tracker.Track(connID, "192.168.1.1:80", mockConn)

	state, exists := tracker.Get(connID)
	if !exists {
		t.Fatal("expected connection to exist")
	}

	if state.ConnectionID != connID {
		t.Errorf("expected connection ID %s, got %s", connID, state.ConnectionID)
	}

	_, exists = tracker.Get("non-existent")
	if exists {
		t.Error("expected non-existent connection to not exist")
	}
}

func TestConnectionTracker_UpdateActivity(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	mockConn := testutil.NewMockNetConn()
	connID := "test-conn-1"

	tracker.Track(connID, "192.168.1.1:80", mockConn)

	state, _ := tracker.Get(connID)
	initialActivity := state.LastActivity

	time.Sleep(10 * time.Millisecond)

	tracker.UpdateActivity(connID)

	state, _ = tracker.Get(connID)
	if !state.LastActivity.After(initialActivity) {
		t.Error("expected LastActivity to be updated")
	}
}

func TestConnectionTracker_Remove(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	mockConn := testutil.NewMockNetConn()
	connID := "test-conn-1"

	tracker.Track(connID, "192.168.1.1:80", mockConn)

	_, exists := tracker.Get(connID)
	if !exists {
		t.Fatal("expected connection to exist before removal")
	}

	tracker.Remove(connID)

	_, exists = tracker.Get(connID)
	if exists {
		t.Error("expected connection to be removed")
	}

	if !mockConn.Closed {
		t.Error("expected connection to be closed")
	}
}

func TestConnectionTracker_Cleanup(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	mockConn1 := testutil.NewMockNetConn()
	mockConn2 := testutil.NewMockNetConn()
	mockConn3 := testutil.NewMockNetConn()

	tracker.Track("conn-1", "192.168.1.1:80", mockConn1)
	tracker.Track("conn-2", "192.168.1.2:80", mockConn2)
	tracker.Track("conn-3", "192.168.1.3:80", mockConn3)

	time.Sleep(50 * time.Millisecond)

	tracker.UpdateActivity("conn-3")

	removed := tracker.Cleanup(30 * time.Millisecond)

	if removed != 2 {
		t.Errorf("expected 2 connections to be removed, got %d", removed)
	}

	if tracker.Count() != 1 {
		t.Errorf("expected 1 connection remaining, got %d", tracker.Count())
	}

	_, exists := tracker.Get("conn-3")
	if !exists {
		t.Error("expected conn-3 to still exist")
	}

	_, exists = tracker.Get("conn-1")
	if exists {
		t.Error("expected conn-1 to be removed")
	}

	_, exists = tracker.Get("conn-2")
	if exists {
		t.Error("expected conn-2 to be removed")
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
}

func TestConnectionTracker_DeliverResponse(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	mockConn := testutil.NewMockNetConn()
	connID := "test-conn-1"

	tracker.Track(connID, "192.168.1.1:80", mockConn)

	responseData := []byte("response from server")

	err := tracker.DeliverResponse(connID, responseData)
	if err != nil {
		t.Fatalf("DeliverResponse failed: %v", err)
	}

	writtenData := mockConn.WriteBuf.Bytes()
	if string(writtenData) != string(responseData) {
		t.Errorf("expected data %s, got %s", responseData, writtenData)
	}

	state, _ := tracker.Get(connID)
	if state.LastActivity.IsZero() {
		t.Error("expected LastActivity to be updated")
	}
}

func TestConnectionTracker_DeliverResponse_NotFound(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	responseData := []byte("response from server")

	err := tracker.DeliverResponse("non-existent", responseData)
	if err == nil {
		t.Error("expected error for non-existent connection")
	}
}

func TestConnectionTracker_Count(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	if tracker.Count() != 0 {
		t.Errorf("expected 0 connections, got %d", tracker.Count())
	}

	mockConn1 := testutil.NewMockNetConn()
	mockConn2 := testutil.NewMockNetConn()

	tracker.Track("conn-1", "192.168.1.1:80", mockConn1)
	tracker.Track("conn-2", "192.168.1.2:80", mockConn2)

	if tracker.Count() != 2 {
		t.Errorf("expected 2 connections, got %d", tracker.Count())
	}

	tracker.Remove("conn-1")

	if tracker.Count() != 1 {
		t.Errorf("expected 1 connection, got %d", tracker.Count())
	}
}

func TestConnectionTracker_ConcurrentAccess(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	done := make(chan struct{})
	numGoroutines := 10
	numOpsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOpsPerGoroutine; j++ {
				connID := "conn-" + string(rune(id*numOpsPerGoroutine+j))
				mockConn := testutil.NewMockNetConn()

				tracker.Track(connID, "192.168.1.1:80", mockConn)

				tracker.Get(connID)

				tracker.UpdateActivity(connID)

				tracker.DeliverResponse(connID, []byte("test"))

				tracker.Remove(connID)
			}
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	if tracker.Count() != 0 {
		t.Errorf("expected 0 connections after cleanup, got %d", tracker.Count())
	}
}

func TestConnectionTracker_MultipleCleanupCycles(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	for i := 0; i < 5; i++ {
		mockConn := testutil.NewMockNetConn()
		tracker.Track("conn-"+string(rune(i)), "192.168.1.1:80", mockConn)
	}

	time.Sleep(50 * time.Millisecond)

	removed := tracker.Cleanup(30 * time.Millisecond)
	if removed != 5 {
		t.Errorf("first cleanup: expected 5 connections removed, got %d", removed)
	}

	removed = tracker.Cleanup(30 * time.Millisecond)
	if removed != 0 {
		t.Errorf("second cleanup: expected 0 connections removed, got %d", removed)
	}
}

func TestConnectionTracker_UpdateActivity_NonExistent(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	tracker.UpdateActivity("non-existent")

	if tracker.Count() != 0 {
		t.Error("expected no connections to be created")
	}
}

func TestConnectionTracker_Remove_NonExistent(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	tracker.Remove("non-existent")

	if tracker.Count() != 0 {
		t.Error("expected no connections")
	}
}

func BenchmarkConnectionTracker_Track(b *testing.B) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockConn := testutil.NewMockNetConn()
		tracker.Track("conn-"+string(rune(i)), "192.168.1.1:80", mockConn)
	}
}

func BenchmarkConnectionTracker_Get(b *testing.B) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	mockConn := testutil.NewMockNetConn()
	tracker.Track("test-conn", "192.168.1.1:80", mockConn)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.Get("test-conn")
	}
}

func BenchmarkConnectionTracker_DeliverResponse(b *testing.B) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	mockConn := testutil.NewMockNetConn()
	tracker.Track("test-conn", "192.168.1.1:80", mockConn)

	data := []byte("test response data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.DeliverResponse("test-conn", data)
	}
}

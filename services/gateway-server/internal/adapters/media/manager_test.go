package media

import (
	"fmt"
	"sync"
	"testing"
)

func TestAddRemoveStream(t *testing.T) {
	mgr := NewMediaManager()

	// Add a stream
	err := mgr.AddCamera("cam-01", "rtsp://localhost:8554/cam-01")
	if err != nil {
		t.Fatalf("AddCamera failed: %v", err)
	}

	// Verify stream exists
	status := mgr.GetStatus("cam-01")
	if status.ID != "cam-01" {
		t.Errorf("expected ID cam-01, got %s", status.ID)
	}
	if status.State != StreamStateInactive {
		t.Errorf("expected state INACTIVE, got %s", status.State)
	}

	// Remove the stream
	err = mgr.RemoveCamera("cam-01")
	if err != nil {
		t.Fatalf("RemoveCamera failed: %v", err)
	}

	// Verify stream is gone
	status = mgr.GetStatus("cam-01")
	if status.Error != "stream not found" {
		t.Error("expected stream not found after removal")
	}

	// Remove non-existent stream should error
	err = mgr.RemoveCamera("cam-99")
	if err == nil {
		t.Error("expected error removing non-existent stream")
	}
}

func TestIndependentFailure(t *testing.T) {
	mgr := NewMediaManager()

	// Add 3 streams
	for i := 1; i <= 3; i++ {
		err := mgr.AddCamera(fmt.Sprintf("cam-%02d", i), fmt.Sprintf("rtsp://localhost:8554/cam-%02d", i))
		if err != nil {
			t.Fatalf("AddCamera(cam-%02d) failed: %v", i, err)
		}
	}

	// Mark all active
	mgr.MarkStreamActive("cam-01")
	mgr.MarkStreamActive("cam-02")
	mgr.MarkStreamActive("cam-03")

	// Fail stream 2 independently
	mgr.MarkStreamError("cam-02", "connection timeout")

	// Verify stream 1 is still active
	s1 := mgr.GetStatus("cam-01")
	if s1.State != StreamStateActive {
		t.Errorf("cam-01 should still be ACTIVE, got %s", s1.State)
	}

	// Verify stream 2 is in error
	s2 := mgr.GetStatus("cam-02")
	if s2.State != StreamStateError {
		t.Errorf("cam-02 should be ERROR, got %s", s2.State)
	}
	if s2.Error != "connection timeout" {
		t.Errorf("cam-02 error message mismatch: %s", s2.Error)
	}

	// Verify stream 3 is still active
	s3 := mgr.GetStatus("cam-03")
	if s3.State != StreamStateActive {
		t.Errorf("cam-03 should still be ACTIVE, got %s", s3.State)
	}

	// Health should show 2 active, 1 error
	health := mgr.Health()
	if health.ActiveStreams != 2 {
		t.Errorf("expected 2 active streams, got %d", health.ActiveStreams)
	}
	if health.TotalStreams != 3 {
		t.Errorf("expected 3 total streams, got %d", health.TotalStreams)
	}
	if len(health.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(health.Errors))
	}
}

func TestReconnect(t *testing.T) {
	mgr := NewMediaManager()

	err := mgr.AddCamera("cam-01", "rtsp://localhost:8554/cam-01")
	if err != nil {
		t.Fatalf("AddCamera failed: %v", err)
	}

	// Activate then fail
	mgr.MarkStreamActive("cam-01")
	mgr.MarkStreamError("cam-01", "network error")

	// Verify error state
	status := mgr.GetStatus("cam-01")
	if status.State != StreamStateError {
		t.Errorf("expected ERROR state, got %s", status.State)
	}

	// Reconnect
	err = mgr.Reconnect("cam-01")
	if err != nil {
		t.Fatalf("Reconnect failed: %v", err)
	}

	// Verify stream is back to inactive (ready for reconnection)
	status = mgr.GetStatus("cam-01")
	if status.State != StreamStateInactive {
		t.Errorf("expected INACTIVE after reconnect, got %s", status.State)
	}
	if status.Error != "" {
		t.Errorf("expected empty error after reconnect, got: %s", status.Error)
	}

	// Reconnect non-existent stream should error
	err = mgr.Reconnect("cam-99")
	if err == nil {
		t.Error("expected error reconnecting non-existent stream")
	}
}

func TestMaxFourStreams(t *testing.T) {
	mgr := NewMediaManager()

	// Add 4 streams (should succeed)
	for i := 1; i <= 4; i++ {
		err := mgr.AddCamera(fmt.Sprintf("cam-%02d", i), fmt.Sprintf("rtsp://localhost:8554/cam-%02d", i))
		if err != nil {
			t.Fatalf("AddCamera(cam-%02d) failed: %v", i, err)
		}
	}

	if mgr.StreamCount() != 4 {
		t.Errorf("expected 4 streams, got %d", mgr.StreamCount())
	}

	// 5th stream should fail
	err := mgr.AddCamera("cam-05", "rtsp://localhost:8554/cam-05")
	if err == nil {
		t.Fatal("expected error when adding 5th stream, got nil")
	}

	// Verify still 4 streams
	if mgr.StreamCount() != 4 {
		t.Errorf("expected 4 streams after failed add, got %d", mgr.StreamCount())
	}

	// Remove one and add again should work
	err = mgr.RemoveCamera("cam-01")
	if err != nil {
		t.Fatalf("RemoveCamera failed: %v", err)
	}

	err = mgr.AddCamera("cam-05", "rtsp://localhost:8554/cam-05")
	if err != nil {
		t.Fatalf("AddCamera after removal failed: %v", err)
	}

	if mgr.StreamCount() != 4 {
		t.Errorf("expected 4 streams after replace, got %d", mgr.StreamCount())
	}
}

func TestDuplicateStreamID(t *testing.T) {
	mgr := NewMediaManager()

	err := mgr.AddCamera("cam-01", "rtsp://localhost:8554/cam-01")
	if err != nil {
		t.Fatalf("first AddCamera failed: %v", err)
	}

	// Adding same ID should fail
	err = mgr.AddCamera("cam-01", "rtsp://localhost:8554/cam-01-alt")
	if err == nil {
		t.Fatal("expected error adding duplicate stream ID")
	}
}

func TestValidation(t *testing.T) {
	mgr := NewMediaManager()

	// Empty ID
	err := mgr.AddStream(StreamConfig{
		ID:        "",
		SourceURL: "rtsp://localhost/test",
		Protocol:  ProtocolRTSP,
		Codec:     CodecH264,
	})
	if err == nil {
		t.Error("expected validation error for empty ID")
	}

	// Empty source URL
	err = mgr.AddStream(StreamConfig{
		ID:        "cam-01",
		SourceURL: "",
		Protocol:  ProtocolRTSP,
		Codec:     CodecH264,
	})
	if err == nil {
		t.Error("expected validation error for empty sourceURL")
	}

	// Empty protocol
	err = mgr.AddStream(StreamConfig{
		ID:        "cam-01",
		SourceURL: "rtsp://localhost/test",
		Protocol:  "",
		Codec:     CodecH264,
	})
	if err == nil {
		t.Error("expected validation error for empty protocol")
	}

	// Empty codec
	err = mgr.AddStream(StreamConfig{
		ID:        "cam-01",
		SourceURL: "rtsp://localhost/test",
		Protocol:  ProtocolRTSP,
		Codec:     "",
	})
	if err == nil {
		t.Error("expected validation error for empty codec")
	}
}

func TestGetAllStreams(t *testing.T) {
	mgr := NewMediaManager()

	// Empty list
	streams := mgr.GetAllStreams()
	if len(streams) != 0 {
		t.Errorf("expected 0 streams, got %d", len(streams))
	}

	// Add 2 streams
	_ = mgr.AddCamera("cam-01", "rtsp://localhost:8554/cam-01")
	_ = mgr.AddCamera("cam-02", "rtsp://localhost:8554/cam-02")

	streams = mgr.GetAllStreams()
	if len(streams) != 2 {
		t.Errorf("expected 2 streams, got %d", len(streams))
	}

	// Verify IDs are present
	ids := make(map[string]bool)
	for _, s := range streams {
		ids[s.ID] = true
	}
	if !ids["cam-01"] {
		t.Error("cam-01 missing from GetAllStreams")
	}
	if !ids["cam-02"] {
		t.Error("cam-02 missing from GetAllStreams")
	}
}

func TestGenerateConfig(t *testing.T) {
	mgr := NewMediaManager()

	_ = mgr.AddCamera("cam-01", "rtsp://localhost:8554/cam-01")
	_ = mgr.AddCamera("cam-02", "rtsp://localhost:8554/cam-02")

	data, err := mgr.GenerateConfig()
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty config output")
	}
}

func TestHealthReport(t *testing.T) {
	mgr := NewMediaManager()

	health := mgr.Health()
	if !health.ProcessRunning {
		t.Error("expected ProcessRunning to be true")
	}
	if health.TotalStreams != 0 {
		t.Errorf("expected 0 total streams, got %d", health.TotalStreams)
	}
	if health.ActiveStreams != 0 {
		t.Errorf("expected 0 active streams, got %d", health.ActiveStreams)
	}

	// Add and activate streams
	_ = mgr.AddCamera("cam-01", "rtsp://localhost:8554/cam-01")
	_ = mgr.AddCamera("cam-02", "rtsp://localhost:8554/cam-02")
	mgr.MarkStreamActive("cam-01")

	health = mgr.Health()
	if health.TotalStreams != 2 {
		t.Errorf("expected 2 total streams, got %d", health.TotalStreams)
	}
	if health.ActiveStreams != 1 {
		t.Errorf("expected 1 active stream, got %d", health.ActiveStreams)
	}
	if health.MemoryUsageMB <= 0 {
		t.Error("expected positive memory usage estimate")
	}
}

func TestBytesReceivedUpdate(t *testing.T) {
	mgr := NewMediaManager()

	_ = mgr.AddCamera("cam-01", "rtsp://localhost:8554/cam-01")
	mgr.MarkStreamActive("cam-01")

	mgr.UpdateBytesReceived("cam-01", 1024)
	mgr.UpdateBytesReceived("cam-01", 2048)

	status := mgr.GetStatus("cam-01")
	if status.BytesReceived != 3072 {
		t.Errorf("expected 3072 bytes received, got %d", status.BytesReceived)
	}
	if status.LastFrameAt.IsZero() {
		t.Error("expected LastFrameAt to be set")
	}
}

func TestConcurrentAccess(t *testing.T) {
	mgr := NewMediaManager()

	// Pre-add streams
	for i := 1; i <= 4; i++ {
		_ = mgr.AddCamera(fmt.Sprintf("cam-%02d", i), fmt.Sprintf("rtsp://localhost:8554/cam-%02d", i))
	}

	var wg sync.WaitGroup

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = mgr.GetAllStreams()
			_ = mgr.Health()
			_ = mgr.GetStatus("cam-01")
		}()
	}

	// Concurrent state updates
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("cam-%02d", (n%4)+1)
			if n%2 == 0 {
				mgr.MarkStreamActive(id)
			} else {
				mgr.UpdateBytesReceived(id, 100)
			}
		}(i)
	}

	wg.Wait()

	// No panic = pass. Also verify manager state is consistent.
	if mgr.StreamCount() != 4 {
		t.Errorf("expected 4 streams after concurrent access, got %d", mgr.StreamCount())
	}
}

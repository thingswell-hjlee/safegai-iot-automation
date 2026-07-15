package media

import (
	"fmt"
	"strings"
	"testing"
)

func TestSingleStreamConfig(t *testing.T) {
	cfg := NewMediaMTXConfig()
	err := cfg.AddStream(StreamConfig{
		ID:             "cam-01",
		Name:           "Zone A Camera 1",
		SourceURL:      "rtsp://localhost:8554/cam-01",
		Protocol:       ProtocolRTSP,
		Codec:          CodecH264,
		Resolution:     Resolution{Width: 1920, Height: 1080},
		FPS:            10,
		MaxBitrateKbps: 2048,
		OnDemand:       true,
	})
	if err != nil {
		t.Fatalf("AddStream failed: %v", err)
	}

	yaml, err := cfg.GenerateYAML()
	if err != nil {
		t.Fatalf("GenerateYAML failed: %v", err)
	}

	content := string(yaml)

	// Verify stream path exists
	if !strings.Contains(content, "cam-01:") {
		t.Error("expected cam-01 path in config")
	}

	// Verify WebRTC enabled
	if !strings.Contains(content, "webrtc: yes") {
		t.Error("expected WebRTC to be enabled")
	}

	// Verify HLS enabled (fallback)
	if !strings.Contains(content, "hls: yes") {
		t.Error("expected HLS to be enabled")
	}

	// Verify on-demand
	if !strings.Contains(content, "sourceOnDemand: yes") {
		t.Error("expected sourceOnDemand: yes")
	}

	// Verify source protocol
	if !strings.Contains(content, "sourceProtocol: tcp") {
		t.Error("expected sourceProtocol: tcp")
	}
}

func TestFourStreamConfig(t *testing.T) {
	cfg := NewMediaMTXConfig()

	streams := []StreamConfig{
		{ID: "cam-01", Name: "Zone A Camera 1", SourceURL: "rtsp://localhost:8554/cam-01", Protocol: ProtocolRTSP, Codec: CodecH264, Resolution: Resolution{Width: 640, Height: 360}, FPS: 10, MaxBitrateKbps: 768, OnDemand: true},
		{ID: "cam-02", Name: "Zone A Camera 2", SourceURL: "rtsp://localhost:8554/cam-02", Protocol: ProtocolRTSP, Codec: CodecH264, Resolution: Resolution{Width: 640, Height: 360}, FPS: 10, MaxBitrateKbps: 768, OnDemand: true},
		{ID: "cam-03", Name: "Zone B Camera 1", SourceURL: "rtsp://localhost:8554/cam-03", Protocol: ProtocolRTSP, Codec: CodecH264, Resolution: Resolution{Width: 640, Height: 360}, FPS: 10, MaxBitrateKbps: 768, OnDemand: true},
		{ID: "cam-04", Name: "Zone B Camera 2", SourceURL: "rtsp://localhost:8554/cam-04", Protocol: ProtocolRTSP, Codec: CodecH264, Resolution: Resolution{Width: 640, Height: 360}, FPS: 10, MaxBitrateKbps: 768, OnDemand: true},
	}

	for _, s := range streams {
		if err := cfg.AddStream(s); err != nil {
			t.Fatalf("AddStream(%s) failed: %v", s.ID, err)
		}
	}

	if cfg.StreamCount() != 4 {
		t.Errorf("expected 4 streams, got %d", cfg.StreamCount())
	}

	yaml, err := cfg.GenerateYAML()
	if err != nil {
		t.Fatalf("GenerateYAML failed: %v", err)
	}

	content := string(yaml)
	for _, s := range streams {
		if !strings.Contains(content, s.ID+":") {
			t.Errorf("expected stream path %s in config", s.ID)
		}
	}
}

func TestMaxStreamsEnforced(t *testing.T) {
	cfg := NewMediaMTXConfig()

	for i := 1; i <= MaxStreams; i++ {
		err := cfg.AddStream(StreamConfig{
			ID:             fmt.Sprintf("cam-%02d", i),
			Name:           fmt.Sprintf("Camera %d", i),
			SourceURL:      fmt.Sprintf("rtsp://localhost:8554/cam-%02d", i),
			Protocol:       ProtocolRTSP,
			Codec:          CodecH264,
			Resolution:     Resolution{Width: 640, Height: 360},
			FPS:            10,
			MaxBitrateKbps: 768,
			OnDemand:       true,
		})
		if err != nil {
			t.Fatalf("AddStream(%d) should succeed: %v", i, err)
		}
	}

	// 5th stream should fail
	err := cfg.AddStream(StreamConfig{
		ID:             "cam-05",
		Name:           "Camera 5",
		SourceURL:      "rtsp://localhost:8554/cam-05",
		Protocol:       ProtocolRTSP,
		Codec:          CodecH264,
		Resolution:     Resolution{Width: 640, Height: 360},
		FPS:            10,
		MaxBitrateKbps: 768,
		OnDemand:       true,
	})
	if err == nil {
		t.Fatal("expected error when adding 5th stream, got nil")
	}

	if !strings.Contains(err.Error(), "maximum streams") {
		t.Errorf("expected 'maximum streams' in error, got: %v", err)
	}
}

func TestNoRecordingInConfig(t *testing.T) {
	cfg := NewMediaMTXConfig()
	_ = cfg.AddStream(StreamConfig{
		ID:             "cam-01",
		Name:           "Test Camera",
		SourceURL:      "rtsp://localhost:8554/cam-01",
		Protocol:       ProtocolRTSP,
		Codec:          CodecH264,
		Resolution:     Resolution{Width: 640, Height: 360},
		FPS:            10,
		MaxBitrateKbps: 768,
		OnDemand:       true,
	})

	yaml, err := cfg.GenerateYAML()
	if err != nil {
		t.Fatalf("GenerateYAML failed: %v", err)
	}

	content := string(yaml)

	// Verify global recording disabled
	if !strings.Contains(content, "record: no") {
		t.Error("expected 'record: no' in config")
	}

	// Verify no recording path or directory references
	forbiddenRecording := []string{
		"recordPath:",
		"recordFormat:",
		"recordSegmentDuration:",
		"recordDeleteAfter:",
	}
	for _, forbidden := range forbiddenRecording {
		if strings.Contains(content, forbidden) {
			t.Errorf("found forbidden recording directive: %s", forbidden)
		}
	}
}

func TestNoTranscodingInConfig(t *testing.T) {
	cfg := NewMediaMTXConfig()
	_ = cfg.AddStream(StreamConfig{
		ID:             "cam-01",
		Name:           "Test Camera",
		SourceURL:      "rtsp://localhost:8554/cam-01",
		Protocol:       ProtocolRTSP,
		Codec:          CodecH264,
		Resolution:     Resolution{Width: 640, Height: 360},
		FPS:            10,
		MaxBitrateKbps: 768,
		OnDemand:       true,
	})

	yaml, err := cfg.GenerateYAML()
	if err != nil {
		t.Fatalf("GenerateYAML failed: %v", err)
	}

	content := string(yaml)

	// Verify no transcoding directives
	forbiddenTranscoding := []string{
		"ffmpeg",
		"runOnReady:",
		"runOnReadyRestart:",
		"runOnDemand:",
		"runOnInit:",
		"gstreamer",
		"-vcodec",
		"-acodec",
		"transcode",
	}
	for _, forbidden := range forbiddenTranscoding {
		if strings.Contains(strings.ToLower(content), strings.ToLower(forbidden)) {
			t.Errorf("found forbidden transcoding directive: %s", forbidden)
		}
	}
}

func TestOnDemandEnabled(t *testing.T) {
	cfg := NewMediaMTXConfig()
	_ = cfg.AddStream(StreamConfig{
		ID:             "cam-01",
		Name:           "Test Camera",
		SourceURL:      "rtsp://localhost:8554/cam-01",
		Protocol:       ProtocolRTSP,
		Codec:          CodecH264,
		Resolution:     Resolution{Width: 640, Height: 360},
		FPS:            10,
		MaxBitrateKbps: 768,
		OnDemand:       true,
	})

	yaml, err := cfg.GenerateYAML()
	if err != nil {
		t.Fatalf("GenerateYAML failed: %v", err)
	}

	content := string(yaml)

	if !strings.Contains(content, "sourceOnDemand: yes") {
		t.Error("expected on-demand connection to be enabled")
	}

	if !strings.Contains(content, "sourceOnDemandStartTimeout:") {
		t.Error("expected sourceOnDemandStartTimeout to be set")
	}

	if !strings.Contains(content, "sourceOnDemandCloseAfter:") {
		t.Error("expected sourceOnDemandCloseAfter to be set")
	}
}

func TestOnDemandDisabled(t *testing.T) {
	cfg := NewMediaMTXConfig()
	_ = cfg.AddStream(StreamConfig{
		ID:             "cam-01",
		Name:           "Test Camera",
		SourceURL:      "rtsp://localhost:8554/cam-01",
		Protocol:       ProtocolRTSP,
		Codec:          CodecH264,
		Resolution:     Resolution{Width: 640, Height: 360},
		FPS:            10,
		MaxBitrateKbps: 768,
		OnDemand:       false,
	})

	yaml, err := cfg.GenerateYAML()
	if err != nil {
		t.Fatalf("GenerateYAML failed: %v", err)
	}

	content := string(yaml)

	if !strings.Contains(content, "sourceOnDemand: no") {
		t.Error("expected on-demand connection to be disabled")
	}
}

func TestNoCredentialsInConfig(t *testing.T) {
	cfg := NewMediaMTXConfig()
	_ = cfg.AddStream(StreamConfig{
		ID:             "cam-01",
		Name:           "Test Camera",
		SourceURL:      "rtsp://localhost:8554/cam-01",
		Protocol:       ProtocolRTSP,
		Codec:          CodecH264,
		Resolution:     Resolution{Width: 640, Height: 360},
		FPS:            10,
		MaxBitrateKbps: 768,
		OnDemand:       true,
	})

	yaml, err := cfg.GenerateYAML()
	if err != nil {
		t.Fatalf("GenerateYAML failed: %v", err)
	}

	content := string(yaml)

	// Verify no credential patterns
	forbiddenCredentials := []string{
		"password",
		"passwd",
		"secret",
		"apikey",
		"api_key",
		"admin:",
	}
	for _, forbidden := range forbiddenCredentials {
		if strings.Contains(strings.ToLower(content), forbidden) {
			t.Errorf("found potential credential in config: %s", forbidden)
		}
	}

	// Verify comment about credential injection
	if !strings.Contains(content, "encrypted") {
		t.Error("expected reference to encrypted credential storage")
	}
}

func TestRemoveStream(t *testing.T) {
	cfg := NewMediaMTXConfig()
	_ = cfg.AddStream(StreamConfig{
		ID:             "cam-01",
		Name:           "Camera 1",
		SourceURL:      "rtsp://localhost:8554/cam-01",
		Protocol:       ProtocolRTSP,
		Codec:          CodecH264,
		Resolution:     Resolution{Width: 640, Height: 360},
		FPS:            10,
		MaxBitrateKbps: 768,
		OnDemand:       true,
	})
	_ = cfg.AddStream(StreamConfig{
		ID:             "cam-02",
		Name:           "Camera 2",
		SourceURL:      "rtsp://localhost:8554/cam-02",
		Protocol:       ProtocolRTSP,
		Codec:          CodecH264,
		Resolution:     Resolution{Width: 640, Height: 360},
		FPS:            10,
		MaxBitrateKbps: 768,
		OnDemand:       true,
	})

	if cfg.StreamCount() != 2 {
		t.Fatalf("expected 2 streams, got %d", cfg.StreamCount())
	}

	removed := cfg.RemoveStream("cam-01")
	if !removed {
		t.Error("expected RemoveStream to return true")
	}

	if cfg.StreamCount() != 1 {
		t.Errorf("expected 1 stream after removal, got %d", cfg.StreamCount())
	}

	// Remove non-existent stream
	removed = cfg.RemoveStream("cam-99")
	if removed {
		t.Error("expected RemoveStream to return false for non-existent stream")
	}

	// Can add new stream after removal
	err := cfg.AddStream(StreamConfig{
		ID:             "cam-03",
		Name:           "Camera 3",
		SourceURL:      "rtsp://localhost:8554/cam-03",
		Protocol:       ProtocolRTSP,
		Codec:          CodecH264,
		Resolution:     Resolution{Width: 640, Height: 360},
		FPS:            10,
		MaxBitrateKbps: 768,
		OnDemand:       true,
	})
	if err != nil {
		t.Fatalf("AddStream after removal failed: %v", err)
	}
}

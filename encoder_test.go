// SPDX-License-Identifier: Apache-2.0
// Copyright Contributors to the OpenTimelineIO project

package fcp7xml

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/mrjoshuak/gotio/opentime"
	"github.com/mrjoshuak/gotio/opentimelineio"
)

func TestEncoder_Encode(t *testing.T) {
	// Create a simple timeline
	timeline := opentimelineio.NewTimeline("Test Timeline", nil, nil)

	// Create a video track
	videoTrack := opentimelineio.NewTrack("Video 1", nil, opentimelineio.TrackKindVideo, nil, nil)

	// Create a clip
	mediaRef := opentimelineio.NewExternalReference(
		"test.mov",
		"file:///path/to/test.mov",
		&opentime.TimeRange{},
		nil,
	)
	sourceRange := opentime.NewTimeRange(
		opentime.NewRationalTime(0, 24),
		opentime.NewRationalTime(50, 24),
	)
	clip := opentimelineio.NewClip(
		"Test Clip",
		mediaRef,
		&sourceRange,
		nil,
		nil,
		nil,
		"",
		nil,
	)

	videoTrack.AppendChild(clip)
	timeline.Tracks().AppendChild(videoTrack)

	// Encode to XML
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	err := encoder.Encode(timeline)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	xmlString := buf.String()

	// Basic validation
	if !strings.Contains(xmlString, "<?xml version") {
		t.Error("Missing XML declaration")
	}
	if !strings.Contains(xmlString, "<!DOCTYPE xmeml>") {
		t.Error("Missing DOCTYPE declaration")
	}
	if !strings.Contains(xmlString, "<xmeml") {
		t.Error("Missing xmeml root element")
	}
	if !strings.Contains(xmlString, "<sequence>") {
		t.Error("Missing sequence element")
	}
	if !strings.Contains(xmlString, "Test Timeline") {
		t.Error("Missing timeline name")
	}
	if !strings.Contains(xmlString, "Test Clip") {
		t.Error("Missing clip name")
	}

	// Parse the XML to verify it's valid
	var xmeml XMEML
	err = xml.Unmarshal([]byte(xmlString), &xmeml)
	if err != nil {
		t.Fatalf("Generated XML is not valid: %v", err)
	}

	// Check structure
	if len(xmeml.Sequence) != 1 {
		t.Errorf("Expected 1 sequence, got %d", len(xmeml.Sequence))
	}

	if len(xmeml.Sequence) > 0 {
		seq := xmeml.Sequence[0]
		if seq.Name != "Test Timeline" {
			t.Errorf("Expected name 'Test Timeline', got '%s'", seq.Name)
		}

		if seq.Media.Video == nil {
			t.Fatal("No video tracks in sequence")
		}

		if len(seq.Media.Video.Track) != 1 {
			t.Errorf("Expected 1 video track, got %d", len(seq.Media.Video.Track))
		}

		if len(seq.Media.Video.Track) > 0 {
			track := seq.Media.Video.Track[0]
			if len(track.ClipItem) != 1 {
				t.Errorf("Expected 1 clip item, got %d", len(track.ClipItem))
			}

			if len(track.ClipItem) > 0 {
				clipItem := track.ClipItem[0]
				if clipItem.Name != "Test Clip" {
					t.Errorf("Expected clip name 'Test Clip', got '%s'", clipItem.Name)
				}
				if clipItem.Duration != 50 {
					t.Errorf("Expected duration 50, got %d", clipItem.Duration)
				}
			}
		}
	}
}

func TestEncoder_EncodeRoundTrip(t *testing.T) {
	// Create a timeline, encode it, then decode it back
	timeline := opentimelineio.NewTimeline("Round Trip Test", nil, nil)

	videoTrack := opentimelineio.NewTrack("Video 1", nil, opentimelineio.TrackKindVideo, nil, nil)
	audioTrack := opentimelineio.NewTrack("Audio 1", nil, opentimelineio.TrackKindAudio, nil, nil)

	// Add video clip
	videoSourceRange := opentime.NewTimeRange(
		opentime.NewRationalTime(0, 24),
		opentime.NewRationalTime(100, 24),
	)
	videoClip := opentimelineio.NewClip(
		"Video Clip",
		opentimelineio.NewExternalReference("video.mov", "file:///video.mov", nil, nil),
		&videoSourceRange,
		nil,
		nil,
		nil,
		"",
		nil,
	)
	videoTrack.AppendChild(videoClip)

	// Add audio clip
	audioSourceRange := opentime.NewTimeRange(
		opentime.NewRationalTime(0, 24),
		opentime.NewRationalTime(100, 24),
	)
	audioClip := opentimelineio.NewClip(
		"Audio Clip",
		opentimelineio.NewExternalReference("audio.wav", "file:///audio.wav", nil, nil),
		&audioSourceRange,
		nil,
		nil,
		nil,
		"",
		nil,
	)
	audioTrack.AppendChild(audioClip)

	timeline.Tracks().AppendChild(videoTrack)
	timeline.Tracks().AppendChild(audioTrack)

	// Encode
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	err := encoder.Encode(timeline)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	// Decode
	decoder := NewDecoder(&buf)
	decodedTimeline, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode() failed: %v", err)
	}

	// Verify
	if decodedTimeline.Name() != "Round Trip Test" {
		t.Errorf("Expected name 'Round Trip Test', got '%s'", decodedTimeline.Name())
	}

	videoTracks := decodedTimeline.VideoTracks()
	if len(videoTracks) != 1 {
		t.Errorf("Expected 1 video track, got %d", len(videoTracks))
	}

	audioTracks := decodedTimeline.AudioTracks()
	if len(audioTracks) != 1 {
		t.Errorf("Expected 1 audio track, got %d", len(audioTracks))
	}

	if len(videoTracks) > 0 && len(videoTracks[0].Children()) > 0 {
		clip := videoTracks[0].Children()[0].(*opentimelineio.Clip)
		if clip.Name() != "Video Clip" {
			t.Errorf("Expected clip name 'Video Clip', got '%s'", clip.Name())
		}
	}

	if len(audioTracks) > 0 && len(audioTracks[0].Children()) > 0 {
		clip := audioTracks[0].Children()[0].(*opentimelineio.Clip)
		if clip.Name() != "Audio Clip" {
			t.Errorf("Expected clip name 'Audio Clip', got '%s'", clip.Name())
		}
	}
}

func TestEncoder_EncodeNTSC(t *testing.T) {
	// Create a timeline with NTSC frame rate (29.97)
	timeline := opentimelineio.NewTimeline("NTSC Timeline", nil, nil)
	videoTrack := opentimelineio.NewTrack("Video 1", nil, opentimelineio.TrackKindVideo, nil, nil)

	// NTSC frame rate: 30000/1001
	ntscRate := 30.0 * 1000.0 / 1001.0

	sourceRange := opentime.NewTimeRange(
		opentime.NewRationalTime(0, ntscRate),
		opentime.NewRationalTime(30, ntscRate),
	)
	clip := opentimelineio.NewClip(
		"NTSC Clip",
		opentimelineio.NewMissingReference("", nil, nil),
		&sourceRange,
		nil,
		nil,
		nil,
		"",
		nil,
	)
	videoTrack.AppendChild(clip)
	timeline.Tracks().AppendChild(videoTrack)

	// Encode
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	err := encoder.Encode(timeline)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	// Parse and verify NTSC flag is set
	var xmeml XMEML
	xmlString := buf.String()
	err = xml.Unmarshal([]byte(xmlString), &xmeml)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if len(xmeml.Sequence) > 0 {
		if !xmeml.Sequence[0].Rate.NTSC {
			t.Error("Expected NTSC flag to be true")
		}
		if xmeml.Sequence[0].Rate.Timebase != 30 {
			t.Errorf("Expected timebase 30, got %d", xmeml.Sequence[0].Rate.Timebase)
		}
	}
}

func TestEncoder_EncodeNilTimeline(t *testing.T) {
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	err := encoder.Encode(nil)
	if err == nil {
		t.Error("Expected error for nil timeline, got nil")
	}
}

func TestEncoder_EncodeEmptyTimeline(t *testing.T) {
	timeline := opentimelineio.NewTimeline("Empty Timeline", nil, nil)

	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	err := encoder.Encode(timeline)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	// Should produce valid XML even with no tracks
	var xmeml XMEML
	xmlString := buf.String()
	err = xml.Unmarshal([]byte(xmlString), &xmeml)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if len(xmeml.Sequence) == 0 {
		t.Error("Expected at least one sequence")
	}
}

func TestEncoder_EncodeWithGaps(t *testing.T) {
	timeline := opentimelineio.NewTimeline("Timeline with Gaps", nil, nil)
	videoTrack := opentimelineio.NewTrack("Video 1", nil, opentimelineio.TrackKindVideo, nil, nil)

	// Add clip, gap, clip
	clip1SourceRange := opentime.NewTimeRange(
		opentime.NewRationalTime(0, 24),
		opentime.NewRationalTime(50, 24),
	)
	clip1 := opentimelineio.NewClip(
		"Clip 1",
		opentimelineio.NewMissingReference("", nil, nil),
		&clip1SourceRange,
		nil,
		nil,
		nil,
		"",
		nil,
	)

	gap := opentimelineio.NewGapWithDuration(opentime.NewRationalTime(25, 24))

	clip2SourceRange := opentime.NewTimeRange(
		opentime.NewRationalTime(0, 24),
		opentime.NewRationalTime(50, 24),
	)
	clip2 := opentimelineio.NewClip(
		"Clip 2",
		opentimelineio.NewMissingReference("", nil, nil),
		&clip2SourceRange,
		nil,
		nil,
		nil,
		"",
		nil,
	)

	videoTrack.AppendChild(clip1)
	videoTrack.AppendChild(gap)
	videoTrack.AppendChild(clip2)
	timeline.Tracks().AppendChild(videoTrack)

	// Encode
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	err := encoder.Encode(timeline)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	// Parse and verify we have 2 clips (gaps are skipped in FCP7 XML)
	var xmeml XMEML
	xmlString := buf.String()
	err = xml.Unmarshal([]byte(xmlString), &xmeml)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if len(xmeml.Sequence) > 0 && xmeml.Sequence[0].Media.Video != nil {
		track := xmeml.Sequence[0].Media.Video.Track[0]
		if len(track.ClipItem) != 2 {
			t.Errorf("Expected 2 clip items (gaps excluded), got %d", len(track.ClipItem))
		}

		// Verify the second clip's start position accounts for the gap
		if len(track.ClipItem) >= 2 {
			// First clip: 0-50
			// Gap: 25 frames
			// Second clip should start at 75
			if track.ClipItem[1].Start != 75 {
				t.Errorf("Expected second clip to start at 75, got %d", track.ClipItem[1].Start)
			}
		}
	}
}

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with spaces", "with_spaces"},
		{"with-dashes", "with-dashes"},
		{"with_underscores", "with_underscores"},
		{"with123numbers", "with123numbers"},
		{"with!@#special", "withspecial"},
		{"", "file"},
	}

	for _, tt := range tests {
		result := sanitizeID(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeID(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsNTSCRate(t *testing.T) {
	tests := []struct {
		rate     float64
		expected bool
	}{
		{23.976, true},
		{29.97, true},
		{59.94, true},
		{24.0, false},
		{25.0, false},
		{30.0, false},
		{60.0, false},
	}

	for _, tt := range tests {
		result := isNTSCRate(tt.rate)
		if result != tt.expected {
			t.Errorf("isNTSCRate(%f) = %v, want %v", tt.rate, result, tt.expected)
		}
	}
}

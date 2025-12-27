// SPDX-License-Identifier: Apache-2.0
// Copyright Contributors to the OpenTimelineIO project

package fcp7xml

import (
	"os"
	"testing"

	"github.com/mrjoshuak/gotio/opentimelineio"
)

func TestDecoder_DecodeWithMarkers(t *testing.T) {
	f, err := os.Open("testdata/features_test.xml")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	decoder := NewDecoder(f)
	timeline, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Check video track
	videoTracks := timeline.VideoTracks()
	if len(videoTracks) != 1 {
		t.Fatalf("Expected 1 video track, got %d", len(videoTracks))
	}

	track := videoTracks[0]
	children := track.Children()

	// Check first clip has markers
	if len(children) < 1 {
		t.Fatalf("Expected at least 1 item in track")
	}

	clip, ok := children[0].(*opentimelineio.Clip)
	if !ok {
		t.Fatalf("Expected first item to be a Clip")
	}

	markers := clip.Markers()
	if len(markers) != 2 {
		t.Fatalf("Expected 2 markers on clip, got %d", len(markers))
	}

	// Check marker details
	if markers[0].Name() != "Clip Marker 1" {
		t.Errorf("Expected marker name 'Clip Marker 1', got '%s'", markers[0].Name())
	}

	if markers[0].Comment() != "First marker" {
		t.Errorf("Expected marker comment 'First marker', got '%s'", markers[0].Comment())
	}

	// Check marker has color metadata
	metadata := markers[0].Metadata()
	if metadata == nil {
		t.Fatal("Expected marker metadata")
	}

	if _, ok := metadata["fcp7xml_color"]; !ok {
		t.Error("Expected fcp7xml_color in marker metadata")
	}
}

func TestDecoder_DecodeWithEffectsAndFilters(t *testing.T) {
	f, err := os.Open("testdata/features_test.xml")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	decoder := NewDecoder(f)
	timeline, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	videoTracks := timeline.VideoTracks()
	track := videoTracks[0]
	clip := track.Children()[0].(*opentimelineio.Clip)

	metadata := clip.Metadata()
	if metadata == nil {
		t.Fatal("Expected clip metadata")
	}

	// Check effects
	effects, ok := metadata["fcp7xml_effects"]
	if !ok {
		t.Error("Expected fcp7xml_effects in metadata")
	} else {
		effectsArray, ok := effects.([]opentimelineio.AnyDictionary)
		if !ok || len(effectsArray) != 1 {
			t.Errorf("Expected 1 effect, got %d", len(effectsArray))
		}
	}

	// Check filters
	filters, ok := metadata["fcp7xml_filters"]
	if !ok {
		t.Error("Expected fcp7xml_filters in metadata")
	} else {
		filtersArray, ok := filters.([]opentimelineio.AnyDictionary)
		if !ok || len(filtersArray) != 1 {
			t.Errorf("Expected 1 filter, got %d", len(filtersArray))
		}
	}
}

func TestDecoder_DecodeWithTransition(t *testing.T) {
	f, err := os.Open("testdata/features_test.xml")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	decoder := NewDecoder(f)
	timeline, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	videoTracks := timeline.VideoTracks()
	track := videoTracks[0]
	children := track.Children()

	// Second item should be a transition
	if len(children) < 2 {
		t.Fatalf("Expected at least 2 items in track")
	}

	transition, ok := children[1].(*opentimelineio.Transition)
	if !ok {
		t.Fatalf("Expected second item to be a Transition, got %T", children[1])
	}

	if transition.Name() != "Cross Dissolve" {
		t.Errorf("Expected transition name 'Cross Dissolve', got '%s'", transition.Name())
	}

	// Check alignment in metadata
	metadata := transition.Metadata()
	if metadata == nil {
		t.Fatal("Expected transition metadata")
	}

	alignment, ok := metadata["fcp7xml_alignment"].(string)
	if !ok || alignment != "center" {
		t.Errorf("Expected alignment 'center', got '%s'", alignment)
	}
}

func TestDecoder_DecodeWithGenerator(t *testing.T) {
	f, err := os.Open("testdata/features_test.xml")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	decoder := NewDecoder(f)
	timeline, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	videoTracks := timeline.VideoTracks()
	track := videoTracks[0]
	children := track.Children()

	// Third item should be a generator (converted to clip)
	if len(children) < 3 {
		t.Fatalf("Expected at least 3 items in track")
	}

	clip, ok := children[2].(*opentimelineio.Clip)
	if !ok {
		t.Fatalf("Expected third item to be a Clip (generator), got %T", children[2])
	}

	if clip.Name() != "Slug" {
		t.Errorf("Expected generator name 'Slug', got '%s'", clip.Name())
	}

	// Check generator metadata
	metadata := clip.Metadata()
	if metadata == nil {
		t.Fatal("Expected generator metadata")
	}

	isGen, ok := metadata["fcp7xml_generator"].(bool)
	if !ok || !isGen {
		t.Error("Expected fcp7xml_generator to be true")
	}

	// Check generator has markers
	markers := clip.Markers()
	if len(markers) != 1 {
		t.Fatalf("Expected 1 marker on generator, got %d", len(markers))
	}

	if markers[0].Name() != "Generator Marker" {
		t.Errorf("Expected marker name 'Generator Marker', got '%s'", markers[0].Name())
	}

	// Check media reference is GeneratorReference
	mediaRef := clip.MediaReference()
	if _, ok := mediaRef.(*opentimelineio.GeneratorReference); !ok {
		t.Errorf("Expected GeneratorReference, got %T", mediaRef)
	}
}

func TestDecoder_DecodeWithImageSequence(t *testing.T) {
	f, err := os.Open("testdata/features_test.xml")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	decoder := NewDecoder(f)
	timeline, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	videoTracks := timeline.VideoTracks()
	track := videoTracks[0]
	children := track.Children()

	// Fourth item should be an image sequence
	if len(children) < 4 {
		t.Fatalf("Expected at least 4 items in track")
	}

	clip, ok := children[3].(*opentimelineio.Clip)
	if !ok {
		t.Fatalf("Expected fourth item to be a Clip, got %T", children[3])
	}

	if clip.Name() != "Image Sequence" {
		t.Errorf("Expected clip name 'Image Sequence', got '%s'", clip.Name())
	}

	// Check media reference is ImageSequenceReference
	mediaRef := clip.MediaReference()
	imgSeqRef, ok := mediaRef.(*opentimelineio.ImageSequenceReference)
	if !ok {
		t.Fatalf("Expected ImageSequenceReference, got %T", mediaRef)
	}

	// Check image sequence properties
	if imgSeqRef.Name() != "frame_####.png" {
		t.Errorf("Expected name 'frame_####.png', got '%s'", imgSeqRef.Name())
	}
}

func TestEncoder_EncodeWithNewFeatures(t *testing.T) {
	// First decode a file with all features
	f, err := os.Open("testdata/features_test.xml")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}

	decoder := NewDecoder(f)
	timeline, err := decoder.Decode()
	f.Close()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Now encode it
	outFile, err := os.CreateTemp("", "fcp7xml_test_*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(outFile.Name())

	encoder := NewEncoder(outFile)
	if err := encoder.Encode(timeline); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	outFile.Close()

	// Decode the encoded file
	outFile, err = os.Open(outFile.Name())
	if err != nil {
		t.Fatalf("Failed to open encoded file: %v", err)
	}
	defer outFile.Close()

	decoder2 := NewDecoder(outFile)
	timeline2, err := decoder2.Decode()
	if err != nil {
		t.Fatalf("Decode of encoded file failed: %v", err)
	}

	// Verify markers survived round trip
	videoTracks := timeline2.VideoTracks()
	if len(videoTracks) != 1 {
		t.Fatalf("Expected 1 video track after round trip, got %d", len(videoTracks))
	}

	track := videoTracks[0]
	clip := track.Children()[0].(*opentimelineio.Clip)
	markers := clip.Markers()

	if len(markers) != 2 {
		t.Fatalf("Expected 2 markers after round trip, got %d", len(markers))
	}

	if markers[0].Name() != "Clip Marker 1" {
		t.Errorf("Marker name not preserved after round trip: got '%s'", markers[0].Name())
	}
}

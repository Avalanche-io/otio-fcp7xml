// SPDX-License-Identifier: Apache-2.0
// Copyright Contributors to the OpenTimelineIO project

package fcp7xml_test

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/Avalanche-io/gotio/opentime"
	"github.com/Avalanche-io/gotio/opentimelineio"
	"github.com/mrjoshuak/otio-fcp7xml"
)

// ExampleDecoder demonstrates how to decode FCP7 XML into an OTIO Timeline.
func ExampleDecoder() {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE xmeml>
<xmeml version="5">
  <sequence>
    <name>Example Sequence</name>
    <rate>
      <timebase>24</timebase>
      <ntsc>false</ntsc>
    </rate>
    <media>
      <video>
        <track>
          <clipitem>
            <name>My Clip</name>
            <duration>100</duration>
            <rate>
              <timebase>24</timebase>
              <ntsc>false</ntsc>
            </rate>
            <start>0</start>
            <end>100</end>
            <in>0</in>
            <out>100</out>
            <file id="file-1">
              <name>video.mov</name>
              <pathurl>file:///path/to/video.mov</pathurl>
            </file>
          </clipitem>
        </track>
      </video>
    </media>
  </sequence>
</xmeml>`

	decoder := fcp7xml.NewDecoder(strings.NewReader(xmlData))
	timeline, err := decoder.Decode()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Timeline: %s\n", timeline.Name())
	fmt.Printf("Video Tracks: %d\n", len(timeline.VideoTracks()))

	if len(timeline.VideoTracks()) > 0 {
		track := timeline.VideoTracks()[0]
		fmt.Printf("Clips in track: %d\n", len(track.Children()))

		if len(track.Children()) > 0 {
			if clip, ok := track.Children()[0].(*opentimelineio.Clip); ok {
				fmt.Printf("First clip: %s\n", clip.Name())
			}
		}
	}

	// Output:
	// Timeline: Example Sequence
	// Video Tracks: 1
	// Clips in track: 1
	// First clip: My Clip
}

// ExampleEncoder demonstrates how to encode an OTIO Timeline to FCP7 XML.
func ExampleEncoder() {
	// Create a timeline
	timeline := opentimelineio.NewTimeline("My Timeline", nil, nil)

	// Create a video track
	videoTrack := opentimelineio.NewTrack("V1", nil, opentimelineio.TrackKindVideo, nil, nil)

	// Create a clip with a media reference
	mediaRef := opentimelineio.NewExternalReference(
		"video.mov",
		"file:///path/to/video.mov",
		nil,
		nil,
	)

	sourceRange := opentime.NewTimeRange(
		opentime.NewRationalTime(0, 24),
		opentime.NewRationalTime(100, 24),
	)

	clip := opentimelineio.NewClip(
		"My Clip",
		mediaRef,
		&sourceRange,
		nil,
		nil,
		nil,
		"",
		nil,
	)

	// Add clip to track, track to timeline
	videoTrack.AppendChild(clip)
	timeline.Tracks().AppendChild(videoTrack)

	// Encode to FCP7 XML
	var buf bytes.Buffer
	encoder := fcp7xml.NewEncoder(&buf)
	err := encoder.Encode(timeline)
	if err != nil {
		log.Fatal(err)
	}

	// The output is FCP7 XML
	xmlOutput := buf.String()
	fmt.Println("Generated FCP7 XML:")
	fmt.Println(strings.Contains(xmlOutput, "<xmeml"))
	fmt.Println(strings.Contains(xmlOutput, "My Timeline"))
	fmt.Println(strings.Contains(xmlOutput, "My Clip"))

	// Output:
	// Generated FCP7 XML:
	// true
	// true
	// true
}

// ExampleEncoder_roundTrip demonstrates encoding and decoding a timeline.
func ExampleEncoder_roundTrip() {
	// Create a timeline
	original := opentimelineio.NewTimeline("Round Trip", nil, nil)
	videoTrack := opentimelineio.NewTrack("V1", nil, opentimelineio.TrackKindVideo, nil, nil)

	sourceRange := opentime.NewTimeRange(
		opentime.NewRationalTime(0, 24),
		opentime.NewRationalTime(50, 24),
	)

	clip := opentimelineio.NewClip(
		"Test Clip",
		opentimelineio.NewMissingReference("", nil, nil),
		&sourceRange,
		nil,
		nil,
		nil,
		"",
		nil,
	)

	videoTrack.AppendChild(clip)
	original.Tracks().AppendChild(videoTrack)

	// Encode
	var buf bytes.Buffer
	encoder := fcp7xml.NewEncoder(&buf)
	encoder.Encode(original)

	// Decode
	decoder := fcp7xml.NewDecoder(&buf)
	decoded, err := decoder.Decode()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Original: %s\n", original.Name())
	fmt.Printf("Decoded: %s\n", decoded.Name())
	fmt.Printf("Match: %t\n", original.Name() == decoded.Name())

	// Output:
	// Original: Round Trip
	// Decoded: Round Trip
	// Match: true
}

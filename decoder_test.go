// SPDX-License-Identifier: Apache-2.0
// Copyright Contributors to the OpenTimelineIO project

package fcp7xml

import (
	"strings"
	"testing"

	"github.com/mrjoshuak/gotio/opentimelineio"
)

func TestDecoder_Decode(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE xmeml>
<xmeml version="5">
  <sequence>
    <name>Test Sequence</name>
    <duration>100</duration>
    <rate>
      <timebase>24</timebase>
      <ntsc>false</ntsc>
    </rate>
    <media>
      <video>
        <track>
          <enabled>true</enabled>
          <clipitem id="clip1">
            <name>Test Clip</name>
            <enabled>true</enabled>
            <duration>50</duration>
            <rate>
              <timebase>24</timebase>
              <ntsc>false</ntsc>
            </rate>
            <start>0</start>
            <end>50</end>
            <in>0</in>
            <out>50</out>
            <file id="file-1">
              <name>test.mov</name>
              <pathurl>file:///path/to/test.mov</pathurl>
              <duration>100</duration>
            </file>
          </clipitem>
        </track>
      </video>
      <audio>
        <track>
          <enabled>true</enabled>
          <clipitem id="clip2">
            <name>Audio Clip</name>
            <enabled>true</enabled>
            <duration>50</duration>
            <rate>
              <timebase>24</timebase>
              <ntsc>false</ntsc>
            </rate>
            <start>0</start>
            <end>50</end>
            <in>0</in>
            <out>50</out>
          </clipitem>
        </track>
      </audio>
    </media>
  </sequence>
</xmeml>`

	decoder := NewDecoder(strings.NewReader(xmlData))
	timeline, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode() failed: %v", err)
	}

	if timeline == nil {
		t.Fatal("Decode() returned nil timeline")
	}

	if timeline.Name() != "Test Sequence" {
		t.Errorf("Expected timeline name 'Test Sequence', got '%s'", timeline.Name())
	}

	// Check video tracks
	videoTracks := timeline.VideoTracks()
	if len(videoTracks) != 1 {
		t.Errorf("Expected 1 video track, got %d", len(videoTracks))
	}

	if len(videoTracks) > 0 {
		videoTrack := videoTracks[0]
		if len(videoTrack.Children()) != 1 {
			t.Errorf("Expected 1 clip in video track, got %d", len(videoTrack.Children()))
		}

		if len(videoTrack.Children()) > 0 {
			clip, ok := videoTrack.Children()[0].(*opentimelineio.Clip)
			if !ok {
				t.Error("First child is not a Clip")
			} else {
				if clip.Name() != "Test Clip" {
					t.Errorf("Expected clip name 'Test Clip', got '%s'", clip.Name())
				}

				// Check media reference
				mediaRef := clip.MediaReference()
				if mediaRef == nil {
					t.Error("Clip has no media reference")
				} else {
					if extRef, ok := mediaRef.(*opentimelineio.ExternalReference); ok {
						if extRef.TargetURL() != "file:///path/to/test.mov" {
							t.Errorf("Expected URL 'file:///path/to/test.mov', got '%s'", extRef.TargetURL())
						}
					} else {
						t.Error("Media reference is not an ExternalReference")
					}
				}
			}
		}
	}

	// Check audio tracks
	audioTracks := timeline.AudioTracks()
	if len(audioTracks) != 1 {
		t.Errorf("Expected 1 audio track, got %d", len(audioTracks))
	}

	if len(audioTracks) > 0 {
		audioTrack := audioTracks[0]
		if len(audioTrack.Children()) != 1 {
			t.Errorf("Expected 1 clip in audio track, got %d", len(audioTrack.Children()))
		}
	}
}

func TestDecoder_DecodeNTSC(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE xmeml>
<xmeml version="5">
  <sequence>
    <name>NTSC Sequence</name>
    <rate>
      <timebase>30</timebase>
      <ntsc>true</ntsc>
    </rate>
    <media>
      <video>
        <track>
          <clipitem>
            <name>NTSC Clip</name>
            <duration>30</duration>
            <rate>
              <timebase>30</timebase>
              <ntsc>true</ntsc>
            </rate>
            <start>0</start>
            <end>30</end>
            <in>0</in>
            <out>30</out>
          </clipitem>
        </track>
      </video>
    </media>
  </sequence>
</xmeml>`

	decoder := NewDecoder(strings.NewReader(xmlData))
	timeline, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode() failed: %v", err)
	}

	if timeline == nil {
		t.Fatal("Decode() returned nil timeline")
	}

	videoTracks := timeline.VideoTracks()
	if len(videoTracks) > 0 && len(videoTracks[0].Children()) > 0 {
		clip, ok := videoTracks[0].Children()[0].(*opentimelineio.Clip)
		if !ok {
			t.Fatal("First child is not a Clip")
		}

		// Check that the frame rate is NTSC (29.97)
		dur, err := clip.Duration()
		if err != nil {
			t.Fatalf("Failed to get duration: %v", err)
		}

		// NTSC 30fps = 30000/1001 = 29.97...
		expectedRate := 30.0 * 1000.0 / 1001.0
		if abs(dur.Rate()-expectedRate) > 0.01 {
			t.Errorf("Expected NTSC rate ~29.97, got %f", dur.Rate())
		}
	}
}

func TestDecoder_DecodeEmpty(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE xmeml>
<xmeml version="5">
  <sequence>
    <name>Empty Sequence</name>
    <rate>
      <timebase>24</timebase>
      <ntsc>false</ntsc>
    </rate>
    <media>
      <video>
        <track>
        </track>
      </video>
    </media>
  </sequence>
</xmeml>`

	decoder := NewDecoder(strings.NewReader(xmlData))
	timeline, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode() failed: %v", err)
	}

	if timeline == nil {
		t.Fatal("Decode() returned nil timeline")
	}

	videoTracks := timeline.VideoTracks()
	if len(videoTracks) != 1 {
		t.Errorf("Expected 1 video track, got %d", len(videoTracks))
	}

	if len(videoTracks) > 0 && len(videoTracks[0].Children()) != 0 {
		t.Errorf("Expected empty video track, got %d children", len(videoTracks[0].Children()))
	}
}

func TestDecoder_DecodeInvalidXML(t *testing.T) {
	xmlData := `<?xml version="1.0"?>
<invalid>
  This is not valid FCP XML
</invalid>`

	decoder := NewDecoder(strings.NewReader(xmlData))
	timeline, err := decoder.Decode()
	if err == nil {
		t.Error("Expected error for invalid XML, got nil")
	}

	if timeline != nil {
		t.Error("Expected nil timeline for invalid XML")
	}
}

func TestDecoder_DecodeNoSequence(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE xmeml>
<xmeml version="5">
</xmeml>`

	decoder := NewDecoder(strings.NewReader(xmlData))
	timeline, err := decoder.Decode()
	if err == nil {
		t.Error("Expected error for missing sequence, got nil")
	}

	if timeline != nil {
		t.Error("Expected nil timeline for missing sequence")
	}
}

func TestDecoder_DecodeMultipleTracks(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE xmeml>
<xmeml version="5">
  <sequence>
    <name>Multi-track Sequence</name>
    <rate>
      <timebase>24</timebase>
      <ntsc>false</ntsc>
    </rate>
    <media>
      <video>
        <track>
          <clipitem>
            <name>Video Clip 1</name>
            <duration>50</duration>
            <rate>
              <timebase>24</timebase>
              <ntsc>false</ntsc>
            </rate>
            <start>0</start>
            <end>50</end>
            <in>0</in>
            <out>50</out>
          </clipitem>
        </track>
        <track>
          <clipitem>
            <name>Video Clip 2</name>
            <duration>30</duration>
            <rate>
              <timebase>24</timebase>
              <ntsc>false</ntsc>
            </rate>
            <start>0</start>
            <end>30</end>
            <in>0</in>
            <out>30</out>
          </clipitem>
        </track>
      </video>
    </media>
  </sequence>
</xmeml>`

	decoder := NewDecoder(strings.NewReader(xmlData))
	timeline, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode() failed: %v", err)
	}

	videoTracks := timeline.VideoTracks()
	if len(videoTracks) != 2 {
		t.Errorf("Expected 2 video tracks, got %d", len(videoTracks))
	}

	if len(videoTracks) >= 2 {
		if len(videoTracks[0].Children()) != 1 {
			t.Errorf("Expected 1 clip in first track, got %d", len(videoTracks[0].Children()))
		}
		if len(videoTracks[1].Children()) != 1 {
			t.Errorf("Expected 1 clip in second track, got %d", len(videoTracks[1].Children()))
		}
	}
}

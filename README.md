# otio-fcp7xml

A Go adapter for reading and writing Final Cut Pro 7 XML (XMEML) files with OpenTimelineIO.

## Overview

This package provides encoding and decoding support for Final Cut Pro 7 XML format, allowing you to convert between FCP7 XML and OpenTimelineIO Timeline objects.

## Installation

```bash
go get github.com/mrjoshuak/otio-fcp7xml
```

### Command-Line Tool

Install the CLI tool:

```bash
go install github.com/mrjoshuak/otio-fcp7xml/cmd/fcp7xml@latest
```

Or build from source:

```bash
cd otio-fcp7xml
go build -o bin/fcp7xml ./cmd/fcp7xml
```

## Usage

### Command-Line Tool

Read and validate an FCP7 XML file:

```bash
fcp7xml -i sequence.xml
```

Convert and normalize an FCP7 XML file:

```bash
fcp7xml -i input.xml -o output.xml
```

### Library Usage

#### Decoding FCP7 XML

```go
package main

import (
    "log"
    "os"

    "github.com/mrjoshuak/otio-fcp7xml"
)

func main() {
    // Open FCP7 XML file
    file, err := os.Open("sequence.xml")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    // Create decoder and decode
    decoder := fcp7xml.NewDecoder(file)
    timeline, err := decoder.Decode()
    if err != nil {
        log.Fatal(err)
    }

    // Use the timeline
    log.Printf("Loaded timeline: %s", timeline.Name())
}
```

#### Encoding to FCP7 XML

```go
package main

import (
    "log"
    "os"

    "github.com/mrjoshuak/gotio/opentime"
    "github.com/mrjoshuak/gotio/opentimelineio"
    "github.com/mrjoshuak/otio-fcp7xml"
)

func main() {
    // Create a timeline
    timeline := opentimelineio.NewTimeline("My Sequence", nil, nil)

    // Create a video track
    videoTrack := opentimelineio.NewTrack(
        "Video 1",
        nil,
        opentimelineio.TrackKindVideo,
        nil,
        nil,
    )

    // Create a clip
    mediaRef := opentimelineio.NewExternalReference(
        "clip.mov",
        "file:///path/to/clip.mov",
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

    // Build the timeline
    videoTrack.AppendChild(clip)
    timeline.Tracks().AppendChild(videoTrack)

    // Encode to FCP7 XML
    file, err := os.Create("output.xml")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    encoder := fcp7xml.NewEncoder(file)
    if err := encoder.Encode(timeline); err != nil {
        log.Fatal(err)
    }
}
```

## Features

### Supported

- Sequences with multiple video and audio tracks
- Clips with media references
- Frame rate handling (both standard and NTSC rates)
- External file references with paths
- Basic clip metadata
- Enabled/disabled state for tracks and clips
- Gaps in timelines

### Not Yet Supported

- Transitions and effects
- Speed effects (time warps)
- Nested sequences
- Markers
- Color information
- Advanced metadata
- Image sequences

## FCP7 XML Format

The adapter handles Final Cut Pro 7's XMEML format (version 5), which includes:

- `<xmeml>` - Root element
- `<sequence>` - Timeline/sequence container
- `<rate>` - Frame rate information (timebase and NTSC flag)
- `<media>` - Container for video and audio tracks
- `<track>` - Individual video or audio track
- `<clipitem>` - Clips with timing information (start, end, in, out)
- `<file>` - Media file references

### Frame Rate Handling

The adapter properly handles both standard and NTSC (drop-frame) rates:

- **Standard rates**: 24, 25, 30, 60 fps
- **NTSC rates**: 23.976, 29.97, 59.94 fps (calculated as timebase * 1000/1001)

## API

### Decoder

```go
type Decoder struct {
    // contains filtered or unexported fields
}

func NewDecoder(r io.Reader) *Decoder
func (d *Decoder) Decode() (*opentimelineio.Timeline, error)
```

### Encoder

```go
type Encoder struct {
    // contains filtered or unexported fields
}

func NewEncoder(w io.Writer) *Encoder
func (e *Encoder) Encode(t *opentimelineio.Timeline) error
```

## Testing

Run the test suite:

```bash
go test -v ./...
```

Run examples:

```bash
go test -v -run Example
```

## License

Apache-2.0 - See LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

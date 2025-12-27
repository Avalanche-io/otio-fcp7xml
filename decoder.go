// SPDX-License-Identifier: Apache-2.0
// Copyright Contributors to the OpenTimelineIO project

package fcp7xml

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/Avalanche-io/gotio/opentime"
	"github.com/Avalanche-io/gotio/opentimelineio"
)

// Decoder decodes Final Cut Pro 7 XML into OTIO Timeline.
type Decoder struct {
	r io.Reader
}

// NewDecoder creates a new FCP7 XML decoder.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// Decode parses FCP7 XML and returns an OTIO Timeline.
func (d *Decoder) Decode() (*opentimelineio.Timeline, error) {
	var xmeml XMEML
	decoder := xml.NewDecoder(d.r)
	if err := decoder.Decode(&xmeml); err != nil {
		return nil, fmt.Errorf("failed to decode XML: %w", err)
	}

	if len(xmeml.Sequence) == 0 {
		return nil, fmt.Errorf("no sequence found in FCP7 XML")
	}

	// For now, convert the first sequence
	// In the future, we might want to handle multiple sequences
	return d.convertSequence(&xmeml.Sequence[0])
}

// convertSequence converts an FCP7 Sequence to an OTIO Timeline.
func (d *Decoder) convertSequence(seq *Sequence) (*opentimelineio.Timeline, error) {
	timeline := opentimelineio.NewTimeline(seq.Name, nil, nil)

	// Convert video tracks
	if seq.Media.Video != nil {
		for i, fcpTrack := range seq.Media.Video.Track {
			track, err := d.convertTrack(&fcpTrack, &seq.Rate, opentimelineio.TrackKindVideo, i)
			if err != nil {
				return nil, fmt.Errorf("failed to convert video track %d: %w", i, err)
			}
			if err := timeline.Tracks().AppendChild(track); err != nil {
				return nil, fmt.Errorf("failed to append video track: %w", err)
			}
		}
	}

	// Convert audio tracks
	if seq.Media.Audio != nil {
		for i, fcpTrack := range seq.Media.Audio.Track {
			track, err := d.convertTrack(&fcpTrack, &seq.Rate, opentimelineio.TrackKindAudio, i)
			if err != nil {
				return nil, fmt.Errorf("failed to convert audio track %d: %w", i, err)
			}
			if err := timeline.Tracks().AppendChild(track); err != nil {
				return nil, fmt.Errorf("failed to append audio track: %w", err)
			}
		}
	}

	return timeline, nil
}

// trackItem represents any item in a track with its start time.
type trackItem struct {
	start      int64
	itemType   string // "clip", "transition", "generator"
	clipItem   *ClipItem
	transition *TransitionItem
	generator  *GeneratorItem
}

// convertTrack converts an FCP7 Track to an OTIO Track.
func (d *Decoder) convertTrack(fcpTrack *Track, rate *Rate, kind string, index int) (*opentimelineio.Track, error) {
	trackName := fmt.Sprintf("%s %d", kind, index+1)
	track := opentimelineio.NewTrack(trackName, nil, kind, nil, nil)

	// Set enabled state if specified
	if fcpTrack.Enabled != nil && !*fcpTrack.Enabled {
		track.SetEnabled(false)
	}

	// Collect all items with their start times
	var items []trackItem

	for i := range fcpTrack.ClipItem {
		items = append(items, trackItem{
			start:    fcpTrack.ClipItem[i].Start,
			itemType: "clip",
			clipItem: &fcpTrack.ClipItem[i],
		})
	}

	for i := range fcpTrack.TransitionItem {
		items = append(items, trackItem{
			start:      fcpTrack.TransitionItem[i].Start,
			itemType:   "transition",
			transition: &fcpTrack.TransitionItem[i],
		})
	}

	for i := range fcpTrack.GeneratorItem {
		items = append(items, trackItem{
			start:     fcpTrack.GeneratorItem[i].Start,
			itemType:  "generator",
			generator: &fcpTrack.GeneratorItem[i],
		})
	}

	// Sort by start time
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].start < items[i].start {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	// Convert items in order
	for i, item := range items {
		switch item.itemType {
		case "clip":
			composable, err := d.convertClipItem(item.clipItem, rate)
			if err != nil {
				return nil, fmt.Errorf("failed to convert clip %d: %w", i, err)
			}
			if err := track.AppendChild(composable); err != nil {
				return nil, fmt.Errorf("failed to append clip: %w", err)
			}

		case "transition":
			trans, err := d.convertTransition(item.transition, rate)
			if err != nil {
				return nil, fmt.Errorf("failed to convert transition %d: %w", i, err)
			}
			if err := track.AppendChild(trans); err != nil {
				return nil, fmt.Errorf("failed to append transition: %w", err)
			}

		case "generator":
			gen, err := d.convertGenerator(item.generator, rate)
			if err != nil {
				return nil, fmt.Errorf("failed to convert generator %d: %w", i, err)
			}
			if err := track.AppendChild(gen); err != nil {
				return nil, fmt.Errorf("failed to append generator: %w", err)
			}
		}
	}

	return track, nil
}

// convertClipItem converts an FCP7 ClipItem to an OTIO Clip.
func (d *Decoder) convertClipItem(item *ClipItem, sequenceRate *Rate) (opentimelineio.Composable, error) {
	// Calculate the frame rate
	rate := item.Rate
	frameRate := float64(rate.Timebase)
	if rate.NTSC {
		// NTSC uses a drop frame rate (e.g., 29.97 instead of 30)
		frameRate = frameRate * 1000.0 / 1001.0
	}

	// Check for nested sequence
	if item.Sequence != nil {
		// Calculate source range for nested sequence
		sourceStart := opentime.NewRationalTime(float64(item.In), frameRate)
		sourceDuration := opentime.NewRationalTime(float64(item.Out-item.In), frameRate)
		sourceRange := opentime.NewTimeRange(sourceStart, sourceDuration)

		// Create a clip referencing the nested timeline
		metadata := make(opentimelineio.AnyDictionary)
		metadata["fcp7xml_nested_sequence"] = true
		metadata["fcp7xml_sequence_name"] = item.Sequence.Name

		clip := opentimelineio.NewClip(
			item.Name,
			opentimelineio.NewMissingReference("", nil, nil),
			&sourceRange,
			metadata,
			nil, nil, "", nil,
		)
		return clip, nil
	}

	// Convert frame numbers to rational times
	// In FCP7 XML:
	// - start/end: position in the timeline
	// - in/out: range in the source media
	// - duration: length of the clip

	// Source range is from in to out point
	sourceStart := opentime.NewRationalTime(float64(item.In), frameRate)
	sourceDuration := opentime.NewRationalTime(float64(item.Out-item.In), frameRate)
	sourceRange := opentime.NewTimeRange(sourceStart, sourceDuration)

	// Create media reference
	var mediaRef opentimelineio.MediaReference
	if item.File != nil && item.File.PathURL != "" {
		// Check for image sequence
		mediaRef = d.createMediaReference(item.File, frameRate)
	} else {
		// No file reference - create missing reference
		mediaRef = opentimelineio.NewMissingReference("", nil, nil)
	}

	// Create metadata
	metadata := make(opentimelineio.AnyDictionary)
	if item.ID != "" {
		metadata["fcp7xml_id"] = item.ID
	}

	// Store effects and filters as metadata
	if len(item.Effect) > 0 {
		metadata["fcp7xml_effects"] = d.effectsToMetadata(item.Effect)
	}
	if len(item.Filter) > 0 {
		metadata["fcp7xml_filters"] = d.filtersToMetadata(item.Filter)
	}

	// Convert markers
	var markers []*opentimelineio.Marker
	for _, m := range item.Marker {
		marker := d.convertMarker(&m, frameRate)
		markers = append(markers, marker)
	}

	// Create the clip
	clip := opentimelineio.NewClip(
		item.Name,
		mediaRef,
		&sourceRange,
		metadata,
		nil,     // effects
		markers, // markers
		"",      // active media reference key
		nil,     // color
	)

	// Set enabled state if specified
	if item.Enabled != nil && !*item.Enabled {
		clip.SetEnabled(false)
	}

	return clip, nil
}

// convertTransition converts an FCP7 TransitionItem to an OTIO Transition.
func (d *Decoder) convertTransition(item *TransitionItem, sequenceRate *Rate) (*opentimelineio.Transition, error) {
	frameRate := rateToFrameRate(&item.Rate)

	metadata := make(opentimelineio.AnyDictionary)
	metadata["fcp7xml_alignment"] = item.Alignment
	if item.Effect != nil {
		metadata["fcp7xml_effect"] = d.effectToMetadata(item.Effect)
	}

	// Split duration between in and out offset (typically 50/50 for center alignment)
	halfDuration := opentime.NewRationalTime(float64(item.End-item.Start)/2.0, frameRate)

	transition := opentimelineio.NewTransition(
		item.Name,
		opentimelineio.TransitionTypeCustom,
		halfDuration,
		halfDuration,
		metadata,
	)

	return transition, nil
}

// convertGenerator converts an FCP7 GeneratorItem to an OTIO Clip.
func (d *Decoder) convertGenerator(item *GeneratorItem, sequenceRate *Rate) (*opentimelineio.Clip, error) {
	frameRate := rateToFrameRate(&item.Rate)

	// Calculate source range
	sourceStart := opentime.NewRationalTime(float64(item.In), frameRate)
	sourceDuration := opentime.NewRationalTime(float64(item.Duration), frameRate)
	sourceRange := opentime.NewTimeRange(sourceStart, sourceDuration)

	// Create metadata to preserve generator type
	metadata := make(opentimelineio.AnyDictionary)
	metadata["fcp7xml_generator"] = true
	metadata["fcp7xml_generator_name"] = item.Name

	if item.Effect != nil {
		metadata["fcp7xml_effect"] = d.effectToMetadata(item.Effect)
	}
	if len(item.Filter) > 0 {
		metadata["fcp7xml_filters"] = d.filtersToMetadata(item.Filter)
	}

	// Convert markers
	var markers []*opentimelineio.Marker
	for _, m := range item.Marker {
		marker := d.convertMarker(&m, frameRate)
		markers = append(markers, marker)
	}

	// Generators don't have file references
	mediaRef := opentimelineio.NewGeneratorReference(
		item.Name,
		item.Name, // generator kind
		nil,       // parameters
		nil,       // available range
		nil,       // metadata
	)

	clip := opentimelineio.NewClip(
		item.Name,
		mediaRef,
		&sourceRange,
		metadata,
		nil,
		markers,
		"",
		nil,
	)

	if item.Enabled != nil && !*item.Enabled {
		clip.SetEnabled(false)
	}

	return clip, nil
}

// convertMarker converts an FCP7 Marker to an OTIO Marker.
func (d *Decoder) convertMarker(m *Marker, frameRate float64) *opentimelineio.Marker {
	markedRange := opentime.NewTimeRange(
		opentime.NewRationalTime(float64(m.In), frameRate),
		opentime.NewRationalTime(float64(m.Out-m.In), frameRate),
	)

	metadata := make(opentimelineio.AnyDictionary)
	if m.Comment != "" {
		metadata["comment"] = m.Comment
	}

	// Store FCP7 color in metadata
	if m.Color != nil {
		metadata["fcp7xml_color"] = map[string]int{
			"red":   m.Color.Red,
			"green": m.Color.Green,
			"blue":  m.Color.Blue,
			"alpha": m.Color.Alpha,
		}
	}

	// Use default marker color
	markerColor := opentimelineio.MarkerColorGreen
	comment := m.Comment

	return opentimelineio.NewMarker(m.Name, markedRange, markerColor, comment, metadata)
}

// createMediaReference creates the appropriate MediaReference, detecting image sequences.
func (d *Decoder) createMediaReference(file *File, frameRate float64) opentimelineio.MediaReference {
	availableRange := opentime.NewTimeRange(
		opentime.NewRationalTime(0, frameRate),
		opentime.NewRationalTime(float64(file.Duration), frameRate),
	)

	// Detect image sequence patterns (e.g., file.####.ext or file.%04d.ext)
	name := file.Name
	pathURL := file.PathURL

	// Common image sequence patterns
	isImageSequence := false
	if len(name) > 0 {
		// Check for hash pattern (####) or printf pattern (%04d)
		for i := 0; i < len(name)-3; i++ {
			if name[i:i+4] == "####" {
				isImageSequence = true
				break
			}
		}
		// Check for printf-style patterns
		if !isImageSequence && len(name) > 4 {
			for i := 0; i < len(name)-4; i++ {
				if name[i] == '%' && name[i+1] >= '0' && name[i+1] <= '9' {
					if name[i+3] == 'd' || name[i+4] == 'd' {
						isImageSequence = true
						break
					}
				}
			}
		}
	}

	if isImageSequence {
		metadata := make(opentimelineio.AnyDictionary)
		metadata["fcp7xml_file_id"] = file.ID

		// Parse image sequence pattern - basic implementation
		// For more complex patterns, would need more sophisticated parsing
		namePrefix := ""
		nameSuffix := ""
		startFrame := 0
		frameZeroPadding := 4

		return opentimelineio.NewImageSequenceReference(
			name,
			pathURL,
			namePrefix,
			nameSuffix,
			startFrame,
			1, // frame step
			frameRate,
			frameZeroPadding,
			&availableRange,
			metadata,
			opentimelineio.MissingFramePolicyError,
		)
	}

	// Regular external reference
	return opentimelineio.NewExternalReference(
		name,
		pathURL,
		&availableRange,
		nil,
	)
}

// effectToMetadata converts an Effect to metadata dictionary.
func (d *Decoder) effectToMetadata(effect *Effect) opentimelineio.AnyDictionary {
	metadata := make(opentimelineio.AnyDictionary)
	metadata["name"] = effect.Name
	metadata["effectid"] = effect.EffectID
	metadata["effecttype"] = effect.EffectType
	metadata["mediatype"] = effect.MediaType

	if effect.EffectCategory != "" {
		metadata["effectcategory"] = effect.EffectCategory
	}
	if effect.Duration > 0 {
		metadata["duration"] = effect.Duration
	}
	if effect.StartRatio != nil {
		metadata["startratio"] = *effect.StartRatio
	}
	if effect.EndRatio != nil {
		metadata["endratio"] = *effect.EndRatio
	}
	if effect.Reverse != nil {
		metadata["reverse"] = *effect.Reverse
	}

	if len(effect.Parameter) > 0 {
		params := make([]opentimelineio.AnyDictionary, len(effect.Parameter))
		for i, p := range effect.Parameter {
			params[i] = d.parameterToMetadata(&p)
		}
		metadata["parameters"] = params
	}

	return metadata
}

// effectsToMetadata converts multiple Effects to metadata.
func (d *Decoder) effectsToMetadata(effects []Effect) []opentimelineio.AnyDictionary {
	result := make([]opentimelineio.AnyDictionary, len(effects))
	for i, e := range effects {
		result[i] = d.effectToMetadata(&e)
	}
	return result
}

// filtersToMetadata converts Filters to metadata.
func (d *Decoder) filtersToMetadata(filters []Filter) []opentimelineio.AnyDictionary {
	result := make([]opentimelineio.AnyDictionary, len(filters))
	for i, f := range filters {
		filterMeta := make(opentimelineio.AnyDictionary)
		if f.Enabled != nil {
			filterMeta["enabled"] = *f.Enabled
		}
		if f.Start > 0 {
			filterMeta["start"] = f.Start
		}
		if f.End > 0 {
			filterMeta["end"] = f.End
		}
		if f.Effect != nil {
			filterMeta["effect"] = d.effectToMetadata(f.Effect)
		}
		result[i] = filterMeta
	}
	return result
}

// parameterToMetadata converts a Parameter to metadata.
func (d *Decoder) parameterToMetadata(p *Parameter) opentimelineio.AnyDictionary {
	metadata := make(opentimelineio.AnyDictionary)

	if p.ParameterID != "" {
		metadata["parameterid"] = p.ParameterID
	}
	if p.Name != "" {
		metadata["name"] = p.Name
	}
	if p.Value != "" {
		metadata["value"] = p.Value
	}
	if p.ValueID != "" {
		metadata["valueid"] = p.ValueID
	}
	if p.ValueMin != nil {
		metadata["valuemin"] = *p.ValueMin
	}
	if p.ValueMax != nil {
		metadata["valuemax"] = *p.ValueMax
	}
	if p.ValueList != "" {
		metadata["valuelist"] = p.ValueList
	}

	return metadata
}

// rateToFrameRate converts an FCP7 Rate to a float64 frame rate.
func rateToFrameRate(rate *Rate) float64 {
	frameRate := float64(rate.Timebase)
	if rate.NTSC {
		// NTSC uses a drop frame rate
		frameRate = frameRate * 1000.0 / 1001.0
	}
	return frameRate
}

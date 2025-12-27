// SPDX-License-Identifier: Apache-2.0
// Copyright Contributors to the OpenTimelineIO project

package fcp7xml

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"path/filepath"

	"github.com/Avalanche-io/gotio/opentime"
	"github.com/Avalanche-io/gotio/opentimelineio"
)

// Encoder encodes OTIO Timeline into Final Cut Pro 7 XML.
type Encoder struct {
	w io.Writer
}

// NewEncoder creates a new FCP7 XML encoder.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode converts an OTIO Timeline to FCP7 XML and writes it.
func (e *Encoder) Encode(timeline *opentimelineio.Timeline) error {
	if timeline == nil {
		return fmt.Errorf("timeline cannot be nil")
	}

	xmeml, err := e.convertTimeline(timeline)
	if err != nil {
		return fmt.Errorf("failed to convert timeline: %w", err)
	}

	// Write XML header
	if _, err := e.w.Write([]byte(xml.Header)); err != nil {
		return fmt.Errorf("failed to write XML header: %w", err)
	}

	// Write DOCTYPE
	if _, err := e.w.Write([]byte("<!DOCTYPE xmeml>\n")); err != nil {
		return fmt.Errorf("failed to write DOCTYPE: %w", err)
	}

	// Encode the XMEML
	encoder := xml.NewEncoder(e.w)
	encoder.Indent("", "  ")
	if err := encoder.Encode(xmeml); err != nil {
		return fmt.Errorf("failed to encode XML: %w", err)
	}

	if _, err := e.w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// convertTimeline converts an OTIO Timeline to FCP7 XMEML.
func (e *Encoder) convertTimeline(timeline *opentimelineio.Timeline) (*XMEML, error) {
	// Determine the frame rate from the first track
	frameRate := 24.0 // default
	isNTSC := false

	if timeline.Tracks() != nil && len(timeline.Tracks().Children()) > 0 {
		for _, child := range timeline.Tracks().Children() {
			if track, ok := child.(*opentimelineio.Track); ok {
				if len(track.Children()) > 0 {
					if clip, ok := track.Children()[0].(*opentimelineio.Clip); ok {
						dur, err := clip.Duration()
						if err == nil && dur.Rate() > 0 {
							frameRate = dur.Rate()
							// Check if this is an NTSC rate
							isNTSC = isNTSCRate(frameRate)
							break
						}
					}
				}
			}
		}
	}

	// Create the sequence
	sequence, err := e.convertTracks(timeline, frameRate, isNTSC)
	if err != nil {
		return nil, err
	}

	return &XMEML{
		Version:  "5",
		Sequence: []Sequence{*sequence},
	}, nil
}

// convertTracks converts OTIO tracks to an FCP7 Sequence.
func (e *Encoder) convertTracks(timeline *opentimelineio.Timeline, frameRate float64, isNTSC bool) (*Sequence, error) {
	timebase := int(frameRate)
	if isNTSC {
		// Round up for NTSC rates (e.g., 29.97 -> 30)
		timebase = int(frameRate*1001.0/1000.0 + 0.5)
	}

	rate := Rate{
		Timebase: timebase,
		NTSC:     isNTSC,
	}

	// Calculate duration
	duration, err := timeline.Duration()
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline duration: %w", err)
	}
	durationFrames := int64(duration.Value())

	sequence := &Sequence{
		Name:     timeline.Name(),
		Duration: durationFrames,
		Rate:     rate,
		Media:    Media{},
	}

	// Convert video tracks
	var videoTracks []Track
	for _, track := range timeline.VideoTracks() {
		fcpTrack, err := e.convertTrack(track, &rate)
		if err != nil {
			return nil, fmt.Errorf("failed to convert video track: %w", err)
		}
		videoTracks = append(videoTracks, *fcpTrack)
	}
	if len(videoTracks) > 0 {
		sequence.Media.Video = &Video{Track: videoTracks}
	}

	// Convert audio tracks
	var audioTracks []Track
	for _, track := range timeline.AudioTracks() {
		fcpTrack, err := e.convertTrack(track, &rate)
		if err != nil {
			return nil, fmt.Errorf("failed to convert audio track: %w", err)
		}
		audioTracks = append(audioTracks, *fcpTrack)
	}
	if len(audioTracks) > 0 {
		sequence.Media.Audio = &Audio{Track: audioTracks}
	}

	return sequence, nil
}

// convertTrack converts an OTIO Track to an FCP7 Track.
func (e *Encoder) convertTrack(track *opentimelineio.Track, rate *Rate) (*Track, error) {
	fcpTrack := &Track{
		ClipItem:       make([]ClipItem, 0),
		TransitionItem: make([]TransitionItem, 0),
		GeneratorItem:  make([]GeneratorItem, 0),
	}

	// Set enabled state
	enabled := track.Enabled()
	fcpTrack.Enabled = &enabled

	// Track position in frames for start time
	var currentPosition int64 = 0

	// Convert each child
	for _, child := range track.Children() {
		switch item := child.(type) {
		case *opentimelineio.Clip:
			// Check if it's a generator
			if isGenerator, genItem := e.convertToGenerator(item, rate, currentPosition); isGenerator {
				fcpTrack.GeneratorItem = append(fcpTrack.GeneratorItem, *genItem)
			} else {
				clipItem, err := e.convertClip(item, rate, currentPosition)
				if err != nil {
					return nil, fmt.Errorf("failed to convert clip: %w", err)
				}
				fcpTrack.ClipItem = append(fcpTrack.ClipItem, *clipItem)
			}

			// Update position
			dur, err := item.Duration()
			if err != nil {
				return nil, fmt.Errorf("failed to get clip duration: %w", err)
			}
			currentPosition += int64(dur.Value())

		case *opentimelineio.Transition:
			transItem, err := e.convertTransitionToItem(item, rate, currentPosition)
			if err != nil {
				return nil, fmt.Errorf("failed to convert transition: %w", err)
			}
			fcpTrack.TransitionItem = append(fcpTrack.TransitionItem, *transItem)

			// Update position
			dur := item.InOffset().Add(item.OutOffset())
			currentPosition += int64(dur.Value())

		case *opentimelineio.Gap:
			// Gaps represent empty space in the timeline
			// In FCP7, we can skip them or represent them differently
			dur, err := item.Duration()
			if err != nil {
				return nil, fmt.Errorf("failed to get gap duration: %w", err)
			}
			currentPosition += int64(dur.Value())

		default:
			// Skip unsupported types
			continue
		}
	}

	return fcpTrack, nil
}

// convertClip converts an OTIO Clip to an FCP7 ClipItem.
func (e *Encoder) convertClip(clip *opentimelineio.Clip, rate *Rate, startPosition int64) (*ClipItem, error) {
	// Get source range
	var sourceRange opentime.TimeRange
	if clip.SourceRange() != nil {
		sourceRange = *clip.SourceRange()
	} else {
		// Use available range if no source range
		ar, err := clip.AvailableRange()
		if err != nil {
			return nil, fmt.Errorf("failed to get available range: %w", err)
		}
		sourceRange = ar
	}

	// Convert to frames
	inPoint := int64(sourceRange.StartTime().Value())
	outPoint := inPoint + int64(sourceRange.Duration().Value())
	duration := int64(sourceRange.Duration().Value())

	clipItem := &ClipItem{
		Name:     clip.Name(),
		Duration: duration,
		Rate:     *rate,
		Start:    startPosition,
		End:      startPosition + duration,
		In:       inPoint,
		Out:      outPoint,
	}

	// Set enabled state
	enabled := clip.Enabled()
	clipItem.Enabled = &enabled

	// Get ID from metadata if available
	if metadata := clip.Metadata(); metadata != nil {
		if id, ok := metadata["fcp7xml_id"].(string); ok {
			clipItem.ID = id
		}

		// Restore effects from metadata
		if effects, ok := metadata["fcp7xml_effects"].([]opentimelineio.AnyDictionary); ok {
			clipItem.Effect = e.metadataToEffects(effects)
		}

		// Restore filters from metadata
		if filters, ok := metadata["fcp7xml_filters"].([]opentimelineio.AnyDictionary); ok {
			clipItem.Filter = e.metadataToFilters(filters)
		}
	}

	// Convert markers
	for _, marker := range clip.Markers() {
		fcpMarker := e.convertMarkerToFCP(marker)
		clipItem.Marker = append(clipItem.Marker, fcpMarker)
	}

	// Convert media reference
	mediaRef := clip.MediaReference()
	if mediaRef != nil {
		file, err := e.convertMediaReference(mediaRef, rate)
		if err != nil {
			return nil, fmt.Errorf("failed to convert media reference: %w", err)
		}
		clipItem.File = file
	}

	return clipItem, nil
}

// convertMediaReference converts an OTIO MediaReference to an FCP7 File.
func (e *Encoder) convertMediaReference(ref opentimelineio.MediaReference, rate *Rate) (*File, error) {
	// Generate a file ID based on the reference name
	fileID := "file-" + sanitizeID(ref.Name())

	file := &File{
		ID:   fileID,
		Name: ref.Name(),
		Rate: *rate,
	}

	// Handle different types of references
	switch r := ref.(type) {
	case *opentimelineio.ExternalReference:
		// Convert URL
		targetURL := r.TargetURL()
		if targetURL != "" {
			// Ensure it's a proper file:// URL
			if !isFileURL(targetURL) {
				// Convert file path to file:// URL
				absPath, err := filepath.Abs(targetURL)
				if err == nil {
					fileURL := url.URL{
						Scheme: "file",
						Path:   absPath,
					}
					targetURL = fileURL.String()
				}
			}
			file.PathURL = targetURL
		}

		// Get available range
		if ar := r.AvailableRange(); ar != nil {
			file.Duration = int64(ar.Duration().Value())
		}

	case *opentimelineio.MissingReference:
		// Missing reference - no path URL
		file.PathURL = ""

	default:
		// For other reference types, just use the name
		file.PathURL = ""
	}

	return file, nil
}

// isNTSCRate checks if a frame rate is an NTSC rate.
func isNTSCRate(rate float64) bool {
	// Common NTSC rates: 23.976, 29.97, 59.94
	ntscRates := []float64{
		23.976023976023978, // 24000/1001
		29.97002997002997,  // 30000/1001
		59.94005994005994,  // 60000/1001
	}

	for _, ntsc := range ntscRates {
		if abs(rate-ntsc) < 0.01 {
			return true
		}
	}
	return false
}

// abs returns the absolute value of a float64.
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// sanitizeID sanitizes a string to be used as an XML ID.
func sanitizeID(s string) string {
	// Remove or replace characters that aren't valid in XML IDs
	result := ""
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result += string(r)
		} else if r == ' ' {
			result += "_"
		}
	}
	if result == "" {
		result = "file"
	}
	return result
}

// isFileURL checks if a string is a file:// URL.
func isFileURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme == "file"
}

// convertToGenerator checks if a clip is a generator and converts it.
func (e *Encoder) convertToGenerator(clip *opentimelineio.Clip, rate *Rate, startPosition int64) (bool, *GeneratorItem) {
	metadata := clip.Metadata()
	if metadata == nil {
		return false, nil
	}

	// Check if this is marked as a generator
	isGen, ok := metadata["fcp7xml_generator"].(bool)
	if !ok || !isGen {
		return false, nil
	}

	// Get duration
	dur, err := clip.Duration()
	if err != nil {
		return false, nil
	}
	duration := int64(dur.Value())

	// Get source range for in/out points
	inPoint := int64(0)
	outPoint := duration
	if clip.SourceRange() != nil {
		inPoint = int64(clip.SourceRange().StartTime().Value())
		outPoint = inPoint + duration
	}

	genItem := &GeneratorItem{
		Name:     clip.Name(),
		Duration: duration,
		Rate:     *rate,
		Start:    startPosition,
		End:      startPosition + duration,
		In:       inPoint,
		Out:      outPoint,
	}

	// Set enabled state
	enabled := clip.Enabled()
	genItem.Enabled = &enabled

	// Restore effect from metadata
	if effectMeta, ok := metadata["fcp7xml_effect"].(opentimelineio.AnyDictionary); ok {
		genItem.Effect = e.metadataToEffect(effectMeta)
	}

	// Restore filters from metadata
	if filters, ok := metadata["fcp7xml_filters"].([]opentimelineio.AnyDictionary); ok {
		genItem.Filter = e.metadataToFilters(filters)
	}

	// Convert markers
	for _, marker := range clip.Markers() {
		fcpMarker := e.convertMarkerToFCP(marker)
		genItem.Marker = append(genItem.Marker, fcpMarker)
	}

	return true, genItem
}

// convertTransitionToItem converts an OTIO Transition to FCP7 TransitionItem.
func (e *Encoder) convertTransitionToItem(trans *opentimelineio.Transition, rate *Rate, startPosition int64) (*TransitionItem, error) {
	duration := trans.InOffset().Add(trans.OutOffset())
	durationFrames := int64(duration.Value())

	transItem := &TransitionItem{
		Name:      trans.Name(),
		Rate:      *rate,
		Start:     startPosition,
		End:       startPosition + durationFrames,
		Alignment: "center", // default
	}

	// Get alignment from metadata
	if metadata := trans.Metadata(); metadata != nil {
		if alignment, ok := metadata["fcp7xml_alignment"].(string); ok {
			transItem.Alignment = alignment
		}

		// Restore effect from metadata
		if effectMeta, ok := metadata["fcp7xml_effect"].(opentimelineio.AnyDictionary); ok {
			transItem.Effect = e.metadataToEffect(effectMeta)
		}
	}

	return transItem, nil
}

// convertMarkerToFCP converts an OTIO Marker to FCP7 Marker.
func (e *Encoder) convertMarkerToFCP(marker *opentimelineio.Marker) Marker {
	markedRange := marker.MarkedRange()
	inPoint := int64(markedRange.StartTime().Value())
	outPoint := inPoint + int64(markedRange.Duration().Value())

	fcpMarker := Marker{
		Name:    marker.Name(),
		Comment: marker.Comment(),
		In:      inPoint,
		Out:     outPoint,
	}

	// Restore FCP7 color from metadata if available
	if metadata := marker.Metadata(); metadata != nil {
		if colorMap, ok := metadata["fcp7xml_color"].(map[string]int); ok {
			fcpMarker.Color = &Color{
				Red:   colorMap["red"],
				Green: colorMap["green"],
				Blue:  colorMap["blue"],
				Alpha: colorMap["alpha"],
			}
		}
	}

	return fcpMarker
}

// metadataToEffect converts metadata dictionary to Effect.
func (e *Encoder) metadataToEffect(metadata opentimelineio.AnyDictionary) *Effect {
	effect := &Effect{}

	if name, ok := metadata["name"].(string); ok {
		effect.Name = name
	}
	if effectID, ok := metadata["effectid"].(string); ok {
		effect.EffectID = effectID
	}
	if effectType, ok := metadata["effecttype"].(string); ok {
		effect.EffectType = effectType
	}
	if mediaType, ok := metadata["mediatype"].(string); ok {
		effect.MediaType = mediaType
	}
	if effectCat, ok := metadata["effectcategory"].(string); ok {
		effect.EffectCategory = effectCat
	}
	if duration, ok := metadata["duration"].(int64); ok {
		effect.Duration = duration
	}
	if startRatio, ok := metadata["startratio"].(float64); ok {
		effect.StartRatio = &startRatio
	}
	if endRatio, ok := metadata["endratio"].(float64); ok {
		effect.EndRatio = &endRatio
	}
	if reverse, ok := metadata["reverse"].(bool); ok {
		effect.Reverse = &reverse
	}

	// Convert parameters
	if params, ok := metadata["parameters"].([]opentimelineio.AnyDictionary); ok {
		for _, paramMeta := range params {
			param := e.metadataToParameter(paramMeta)
			effect.Parameter = append(effect.Parameter, param)
		}
	}

	return effect
}

// metadataToEffects converts metadata array to Effects array.
func (e *Encoder) metadataToEffects(metadataArray []opentimelineio.AnyDictionary) []Effect {
	effects := make([]Effect, len(metadataArray))
	for i, meta := range metadataArray {
		effects[i] = *e.metadataToEffect(meta)
	}
	return effects
}

// metadataToFilters converts metadata array to Filters array.
func (e *Encoder) metadataToFilters(metadataArray []opentimelineio.AnyDictionary) []Filter {
	filters := make([]Filter, len(metadataArray))
	for i, meta := range metadataArray {
		filter := Filter{}

		if enabled, ok := meta["enabled"].(bool); ok {
			filter.Enabled = &enabled
		}
		if start, ok := meta["start"].(int64); ok {
			filter.Start = start
		}
		if end, ok := meta["end"].(int64); ok {
			filter.End = end
		}
		if effectMeta, ok := meta["effect"].(opentimelineio.AnyDictionary); ok {
			filter.Effect = e.metadataToEffect(effectMeta)
		}

		filters[i] = filter
	}
	return filters
}

// metadataToParameter converts metadata dictionary to Parameter.
func (e *Encoder) metadataToParameter(metadata opentimelineio.AnyDictionary) Parameter {
	param := Parameter{}

	if paramID, ok := metadata["parameterid"].(string); ok {
		param.ParameterID = paramID
	}
	if name, ok := metadata["name"].(string); ok {
		param.Name = name
	}
	if value, ok := metadata["value"].(string); ok {
		param.Value = value
	}
	if valueID, ok := metadata["valueid"].(string); ok {
		param.ValueID = valueID
	}
	if valueMin, ok := metadata["valuemin"].(float64); ok {
		param.ValueMin = &valueMin
	}
	if valueMax, ok := metadata["valuemax"].(float64); ok {
		param.ValueMax = &valueMax
	}
	if valueList, ok := metadata["valuelist"].(string); ok {
		param.ValueList = valueList
	}

	return param
}

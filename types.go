// SPDX-License-Identifier: Apache-2.0
// Copyright Contributors to the OpenTimelineIO project

package fcp7xml

import "encoding/xml"

// XMEML represents the root element of a Final Cut Pro 7 XML document.
type XMEML struct {
	XMLName  xml.Name   `xml:"xmeml"`
	Version  string     `xml:"version,attr"`
	Sequence []Sequence `xml:"sequence"`
}

// Sequence represents a timeline sequence in FCP7.
type Sequence struct {
	XMLName  xml.Name `xml:"sequence"`
	Name     string   `xml:"name"`
	Duration int64    `xml:"duration,omitempty"`
	Rate     Rate     `xml:"rate"`
	Timecode Timecode `xml:"timecode,omitempty"`
	Media    Media    `xml:"media"`
	Marker   []Marker `xml:"marker,omitempty"`
}

// Rate represents frame rate information.
type Rate struct {
	XMLName  xml.Name `xml:"rate"`
	Timebase int      `xml:"timebase"`
	NTSC     bool     `xml:"ntsc"`
}

// Timecode represents timecode information.
type Timecode struct {
	XMLName      xml.Name `xml:"timecode"`
	Rate         Rate     `xml:"rate"`
	String       string   `xml:"string,omitempty"`
	Frame        int64    `xml:"frame,omitempty"`
	DisplayFormat string   `xml:"displayformat,omitempty"`
}

// Media contains video and audio tracks.
type Media struct {
	XMLName xml.Name `xml:"media"`
	Video   *Video   `xml:"video,omitempty"`
	Audio   *Audio   `xml:"audio,omitempty"`
}

// Video contains video tracks.
type Video struct {
	XMLName xml.Name `xml:"video"`
	Track   []Track  `xml:"track"`
}

// Audio contains audio tracks.
type Audio struct {
	XMLName xml.Name `xml:"audio"`
	Track   []Track  `xml:"track"`
}

// Track represents a single video or audio track.
type Track struct {
	XMLName        xml.Name         `xml:"track"`
	Enabled        *bool            `xml:"enabled,omitempty"`
	Locked         *bool            `xml:"locked,omitempty"`
	ClipItem       []ClipItem       `xml:"clipitem"`
	TransitionItem []TransitionItem `xml:"transitionitem"`
	GeneratorItem  []GeneratorItem  `xml:"generatoritem"`
}

// ClipItem represents a clip in a track.
type ClipItem struct {
	XMLName      xml.Name   `xml:"clipitem"`
	ID           string     `xml:"id,attr,omitempty"`
	Name         string     `xml:"name"`
	Enabled      *bool      `xml:"enabled,omitempty"`
	Duration     int64      `xml:"duration"`
	Rate         Rate       `xml:"rate"`
	Start        int64      `xml:"start"`
	End          int64      `xml:"end"`
	In           int64      `xml:"in"`
	Out          int64      `xml:"out"`
	File         *File      `xml:"file,omitempty"`
	Sequence     *Sequence  `xml:"sequence,omitempty"` // For nested sequences
	SourceTrack  *SourceTrack `xml:"sourcetrack,omitempty"`
	Labels       *Labels    `xml:"labels,omitempty"`
	Comments     *Comments  `xml:"comments,omitempty"`
	Link         []Link     `xml:"link,omitempty"`
	Filter       []Filter   `xml:"filter,omitempty"`
	Effect       []Effect   `xml:"effect,omitempty"`
	Marker       []Marker   `xml:"marker,omitempty"`
}

// File represents a media file reference.
type File struct {
	XMLName     xml.Name    `xml:"file"`
	ID          string      `xml:"id,attr"`
	Name        string      `xml:"name"`
	PathURL     string      `xml:"pathurl,omitempty"`
	Rate        Rate        `xml:"rate,omitempty"`
	Duration    int64       `xml:"duration,omitempty"`
	Timecode    *Timecode   `xml:"timecode,omitempty"`
	Media       *FileMedia  `xml:"media,omitempty"`
}

// FileMedia contains video and audio track information for a file.
type FileMedia struct {
	XMLName xml.Name   `xml:"media"`
	Video   *FileVideo `xml:"video,omitempty"`
	Audio   *FileAudio `xml:"audio,omitempty"`
}

// FileVideo contains video track information.
type FileVideo struct {
	XMLName        xml.Name        `xml:"video"`
	SampleCharacteristics *SampleCharacteristics `xml:"samplecharacteristics,omitempty"`
}

// FileAudio contains audio track information.
type FileAudio struct {
	XMLName        xml.Name        `xml:"audio"`
	SampleCharacteristics *SampleCharacteristics `xml:"samplecharacteristics,omitempty"`
}

// SampleCharacteristics defines media characteristics.
type SampleCharacteristics struct {
	XMLName       xml.Name `xml:"samplecharacteristics"`
	Rate          *Rate    `xml:"rate,omitempty"`
	Width         int      `xml:"width,omitempty"`
	Height        int      `xml:"height,omitempty"`
	AnamorphicMode string  `xml:"anamorphic,omitempty"`
	PixelAspectRatio string `xml:"pixelaspectratio,omitempty"`
	FieldDominance string  `xml:"fielddominance,omitempty"`
	Depth         int      `xml:"depth,omitempty"`
	SampleRate    int      `xml:"samplerate,omitempty"`
	Channels      int      `xml:"channelcount,omitempty"`
}

// SourceTrack identifies which track in the source file.
type SourceTrack struct {
	XMLName   xml.Name `xml:"sourcetrack"`
	MediaType string   `xml:"mediatype"`
	TrackIndex int     `xml:"trackindex,omitempty"`
}

// Labels contains color labels for clips.
type Labels struct {
	XMLName xml.Name `xml:"labels"`
	Label2  string   `xml:"label2,omitempty"`
}

// Comments contains clip comments.
type Comments struct {
	XMLName xml.Name `xml:"comments"`
	Comment []Comment `xml:"comment"`
}

// Comment represents a single comment.
type Comment struct {
	XMLName xml.Name `xml:"comment"`
	Text    string   `xml:",chardata"`
}

// Link represents a link between clips.
type Link struct {
	XMLName    xml.Name `xml:"link"`
	LinkClipRef string  `xml:"linkclipref"`
	MediaType   string  `xml:"mediatype,omitempty"`
	TrackIndex  int     `xml:"trackindex,omitempty"`
}

// Filter represents an effect or filter applied to a clip.
type Filter struct {
	XMLName xml.Name `xml:"filter"`
	Enabled *bool    `xml:"enabled,omitempty"`
	Start   int64    `xml:"start,omitempty"`
	End     int64    `xml:"end,omitempty"`
	Effect  *Effect  `xml:"effect,omitempty"`
}

// Effect represents an effect or processing operation.
type Effect struct {
	XMLName        xml.Name     `xml:"effect"`
	Name           string       `xml:"name"`
	EffectID       string       `xml:"effectid"`
	EffectType     string       `xml:"effecttype"`
	MediaType      string       `xml:"mediatype"`
	EffectCategory string       `xml:"effectcategory,omitempty"`
	Duration       int64        `xml:"duration,omitempty"`
	StartRatio     *float64     `xml:"startratio,omitempty"`
	EndRatio       *float64     `xml:"endratio,omitempty"`
	Reverse        *bool        `xml:"reverse,omitempty"`
	Parameter      []Parameter  `xml:"parameter,omitempty"`
}

// Parameter represents an effect parameter.
type Parameter struct {
	XMLName      xml.Name `xml:"parameter"`
	ParameterID  string   `xml:"parameterid,omitempty"`
	Name         string   `xml:"name,omitempty"`
	Value        string   `xml:"value,omitempty"`
	ValueID      string   `xml:"valueid,omitempty"`
	ValueMin     *float64 `xml:"valuemin,omitempty"`
	ValueMax     *float64 `xml:"valuemax,omitempty"`
	ValueList    string   `xml:"valuelist,omitempty"`
}

// TransitionItem represents a transition in a track.
type TransitionItem struct {
	XMLName   xml.Name `xml:"transitionitem"`
	Name      string   `xml:"name"`
	Rate      Rate     `xml:"rate"`
	Start     int64    `xml:"start"`
	End       int64    `xml:"end"`
	Alignment string   `xml:"alignment"`
	Effect    *Effect  `xml:"effect,omitempty"`
}

// GeneratorItem represents a generator clip (slug, color bars, etc).
type GeneratorItem struct {
	XMLName     xml.Name `xml:"generatoritem"`
	Name        string   `xml:"name"`
	Duration    int64    `xml:"duration"`
	Rate        Rate     `xml:"rate"`
	Start       int64    `xml:"start"`
	End         int64    `xml:"end"`
	In          int64    `xml:"in,omitempty"`
	Out         int64    `xml:"out,omitempty"`
	Enabled     *bool    `xml:"enabled,omitempty"`
	Anamorphic  *bool    `xml:"anamorphic,omitempty"`
	AlphaType   string   `xml:"alphatype,omitempty"`
	Effect      *Effect  `xml:"effect,omitempty"`
	Filter      []Filter `xml:"filter,omitempty"`
	Marker      []Marker `xml:"marker,omitempty"`
}

// Marker represents a marker in a clip or sequence.
type Marker struct {
	XMLName xml.Name `xml:"marker"`
	Name    string   `xml:"name"`
	Comment string   `xml:"comment,omitempty"`
	In      int64    `xml:"in"`
	Out     int64    `xml:"out"`
	Color   *Color   `xml:"color,omitempty"`
}

// Color represents an RGB color value.
type Color struct {
	XMLName xml.Name `xml:"color"`
	Red     int      `xml:"red"`
	Green   int      `xml:"green"`
	Blue    int      `xml:"blue"`
	Alpha   int      `xml:"alpha,omitempty"`
}

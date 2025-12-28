// SPDX-License-Identifier: Apache-2.0
// Copyright Contributors to the OpenTimelineIO project

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Avalanche-io/otio-fcp7xml"
)

func main() {
	var (
		input  = flag.String("i", "", "Input FCP7 XML file")
		output = flag.String("o", "", "Output file (optional, prints to stdout if not specified)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A tool for working with Final Cut Pro 7 XML files.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Read and validate FCP7 XML\n")
		fmt.Fprintf(os.Stderr, "  %s -i sequence.xml\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Convert FCP7 XML to normalized format\n")
		fmt.Fprintf(os.Stderr, "  %s -i input.xml -o output.xml\n\n", os.Args[0])
	}

	flag.Parse()

	if *input == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Open input file
	inFile, err := os.Open(*input)
	if err != nil {
		log.Fatalf("Failed to open input file: %v", err)
	}
	defer inFile.Close()

	// Decode FCP7 XML
	decoder := fcp7xml.NewDecoder(inFile)
	timeline, err := decoder.Decode()
	if err != nil {
		log.Fatalf("Failed to decode FCP7 XML: %v", err)
	}

	// Print timeline info
	fmt.Fprintf(os.Stderr, "Timeline: %s\n", timeline.Name())
	fmt.Fprintf(os.Stderr, "Video Tracks: %d\n", len(timeline.VideoTracks()))
	fmt.Fprintf(os.Stderr, "Audio Tracks: %d\n", len(timeline.AudioTracks()))

	duration, err := timeline.Duration()
	if err == nil {
		fmt.Fprintf(os.Stderr, "Duration: %s\n", duration.String())
	}

	// If output is specified, encode back to FCP7 XML
	if *output != "" {
		outFile, err := os.Create(*output)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer outFile.Close()

		encoder := fcp7xml.NewEncoder(outFile)
		if err := encoder.Encode(timeline); err != nil {
			log.Fatalf("Failed to encode FCP7 XML: %v", err)
		}

		fmt.Fprintf(os.Stderr, "Successfully wrote: %s\n", *output)
	}
}

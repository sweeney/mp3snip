package main

import (
	"fmt"
	"github.com/sweeney/mp3lib"
	"io"
	"os"
	"time"
)

func main() {

	inpath := "in.mp3"
	outpath := "out.mp3"
	after := "25s"

	var framesEncountered, framesDropped, framesIncluded, outputBytes int
	var frameDuration, cumulativeDuration, outputDuration time.Duration

	// Setup files
	out, err := os.Create(outpath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	in, err := os.Open(inpath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Parse for duration
	startAfter, err := time.ParseDuration(after)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	line()
	fmt.Printf("Trim %0.1fs from %s\n", startAfter.Seconds(), inpath)
	fmt.Println("Starting...")

	for {

		// Read the next frame
		// If we're EOF, break out
		frame, frameErr := mp3lib.NextFrame(in)
		if frameErr == io.EOF {
			break
		}
		framesEncountered = framesEncountered + 1

		// If we not a proper frame, continue
		if frame == nil {
			continue
		}

		framesEncountered = framesEncountered + 1

		// Skip VBR headers
		if framesEncountered == 1 {
			if mp3lib.IsXingHeader(frame) || mp3lib.IsVbriHeader(frame) {
				continue
			}
		}

		// Turn into nanoseconds, which is what time.duration expects. Bit crufty
		frameDuration = time.Duration((float64(frame.SampleCount) / float64(frame.SamplingRate)) * 1e9)
		cumulativeDuration = cumulativeDuration + frameDuration

		if cumulativeDuration < startAfter {
			framesDropped = framesDropped + 1
			continue
		}

		framesIncluded = framesIncluded + 1
		outputBytes = outputBytes + frame.FrameLength
		outputDuration = outputDuration + frameDuration

		// Write the frame out
		_, err := out.Write(frame.RawBytes)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

	}

	fmt.Println("Finished.")
	line()
	fmt.Printf("Skipped %d frames\nNew file %f seconds long\n", framesDropped, outputDuration.Seconds())
	line()
}

func line() {
	fmt.Println("------------------------------------------------------------------------------------")
}

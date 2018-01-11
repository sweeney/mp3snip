package main

import (
	"encoding/binary"
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
	before := "9s"

	var framesEncountered, framesDropped, framesIncluded, predictedFrames int64
	var inputBytes, effectiveBytes, outputBytes int64
	var frameDuration, cumulativeDuration, outputDuration time.Duration

	// Parse for duration
	startAfter, err := time.ParseDuration(after)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	finishBefore, err := time.ParseDuration(before)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	line()
	fmt.Printf("Snip %0.1fs Leading from %s\n", startAfter.Seconds(), inpath)
	if finishBefore > 0 {
		fmt.Printf("Snip %0.1fs Trailing from %s\n", finishBefore.Seconds(), inpath)
	}
	fmt.Println("Starting...")

	start := time.Now()

	// Setup files
	in, err := os.Open(inpath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	out, err := os.Create(outpath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer out.Close()

	inputBytes = info.Size()
	effectiveBytes = inputBytes

	for {

		// Read the next frame
		frame, ID3, frameErr := mp3lib.NextFrameOrID3v2Tag(in)
		if frame == nil {

			// ID3 tag encountered
			if ID3 != nil {
				// Deduct the ID3 bytes from the total input file size to get the number of bytes available to host frames
				effectiveBytes = effectiveBytes - int64(binary.Size(ID3.RawBytes))

				// Write the ID3 tag to the output file
				_, err := out.Write(ID3.RawBytes)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}

				continue
			}

			// EOF kills the loop
			if frameErr == io.EOF {
				break
			}

			// Just skil any other sort of nil frame
			continue
		}

		// Skip VBR headers
		if framesEncountered == 1 {
			if mp3lib.IsXingHeader(frame) || mp3lib.IsVbriHeader(frame) {
				continue
			}
		}

		// We've got a frame, count it
		framesEncountered = framesEncountered + 1

		// Take this first frame proper, use to predict how many frames there are in the whole file
		if predictedFrames == 0 {
			predictedFrames = effectiveBytes / int64(binary.Size(frame.RawBytes))
		}

		// Caculate how long the frame lasts using the sampling rate and the number of samples in the frame
		// Turn into nanoseconds, which is what time.Duration needs. A bit crufty
		frameDuration = time.Duration((float64(frame.SampleCount) / float64(frame.SamplingRate)) * 1e9)
		cumulativeDuration = cumulativeDuration + frameDuration

		// Drop frames if they come before the start point
		if cumulativeDuration < startAfter {
			framesDropped = framesDropped + 1
			continue
		}

		// If we have a finish point
		if finishBefore > 0 && predictedFrames > 0 {
			// Calculate the last frame number we should consider
			stopPoint := predictedFrames - int64(finishBefore/frameDuration)

			// Drop any frames thereafter
			if framesEncountered > stopPoint {
				framesDropped = framesDropped + 1
				continue
			}
		}

		// If we've got this far, we're in between start and end points so
		// Keep track of the frame stats
		framesIncluded = framesIncluded + 1
		outputBytes = outputBytes + int64(frame.FrameLength)
		outputDuration = outputDuration + frameDuration

		// And write the frame to the out file
		_, err := out.Write(frame.RawBytes)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

	}

	t := time.Now()
	elapsed := t.Sub(start)

	fmt.Printf("Finished - took %s, saw %v frames vs %v predicted\n", elapsed.String(), framesEncountered, predictedFrames)
	fmt.Printf("Skipped %d frames\n", framesDropped)
	fmt.Printf("New file %0.2fs long vs %0.2fs original\n", outputDuration.Seconds(), cumulativeDuration.Seconds())
	line()
}

func line() {
	fmt.Println("------------------------------------------------------------------------------------")
}

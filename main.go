package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"github.com/sweeney/mp3lib"
	"io"
	"os"
	"time"
)

func main() {

	startAfter, endAt, inPath, outPath, err := grabAndValidateArgs()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	line()
	fmt.Printf("Snip %0.1fs Leading from %s\n", startAfter.Seconds(), inPath)
	if endAt > 0 {
		fmt.Printf("Snip %0.1fs Trailing from %s\n", endAt.Seconds(), inPath)
	}
	fmt.Println("Starting...")

	start := time.Now()

	meta, result := snip(startAfter, endAt, inPath, outPath)
	if result != nil {
		fmt.Fprintln(os.Stderr, result)
		os.Exit(1)
	}

	t := time.Now()
	elapsed := t.Sub(start)

	fmt.Printf("Finished - took %s, saw %v frames vs %v predicted\n", elapsed.String(), meta["framesEncountered"], meta["predictedFrames"])
	fmt.Printf("Skipped %d frames\n", meta["framesDropped"])
	fmt.Printf("New file %ds long vs %ds original\n", meta["outputDuration"], meta["cumulativeDuration"])
	line()

}

func grabAndValidateArgs() (time.Duration, time.Duration, string, string, error) {

	start := flag.String("start", "", "Start after; Duration, parsable by go https://golang.org/pkg/time/#ParseDuration - eg 25s")
	end := flag.String("end", "", "End before; Duration, parsable by go https://golang.org/pkg/time/#ParseDuration - eg 10s. Optional")
	inputFile := flag.String("in", "", "Path to input mp3 file")
	outputFile := flag.String("out", "", "Path to output mp3 file")

	flag.Parse()

	if *start == "" {
		return 0, 0, "", "", errors.New("Missing start time flag")
	}

	startAfter, err := time.ParseDuration(*start)
	if err != nil {
		return 0, 0, "", "", err
	}

	var endAt time.Duration
	if *end != "" {
		endAt, err = time.ParseDuration(*end)
		if err != nil {
			return 0, 0, "", "", err
		}
	}

	if *inputFile == "" {
		return 0, 0, "", "", errors.New("Missing input file path")
	}

	if *outputFile == "" {
		return 0, 0, "", "", errors.New("Missing output file path")
	}

	return startAfter, endAt, *inputFile, *outputFile, nil

}

func snip(startAfter time.Duration, endAt time.Duration, inPath string, outPath string) (map[string]int64, error) {

	meta := make(map[string]int64)
	var frameDuration, cumulativeDuration, outputDuration time.Duration

	// Setup files
	in, err := os.Open(inPath)
	if err != nil {
		return meta, err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return meta, err
	}

	out, err := os.Create(outPath)
	if err != nil {
		return meta, err
	}
	defer out.Close()

	meta["inputBytes"] = info.Size()
	meta["effectiveBytes"] = meta["inputBytes"]

	for {

		// Read the next frame
		frame, ID3, frameErr := mp3lib.NextFrameOrID3v2Tag(in)
		if frame == nil {

			// ID3 tag encountered
			if ID3 != nil {
				// Deduct the ID3 bytes from the total input file size to get the number of bytes available to host frames
				meta["effectiveBytes"] = meta["effectiveBytes"] - int64(binary.Size(ID3.RawBytes))

				// Write the ID3 tag to the output file
				_, err := out.Write(ID3.RawBytes)
				if err != nil {
					return meta, err
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
		if meta["framesEncountered"] == 1 {
			if mp3lib.IsXingHeader(frame) || mp3lib.IsVbriHeader(frame) {
				continue
			}
		}

		// We've got a frame, count it
		meta["framesEncountered"] = meta["framesEncountered"] + 1

		// Take this first frame proper, use to predict how many frames there are in the whole file
		if meta["predictedFrames"] == 0 {
			meta["predictedFrames"] = meta["effectiveBytes"] / int64(binary.Size(frame.RawBytes))
		}

		// Caculate how long the frame lasts using the sampling rate and the number of samples in the frame
		// Turn into nanoseconds, which is what time.Duration needs. A bit crufty
		frameDuration = time.Duration((float64(frame.SampleCount) / float64(frame.SamplingRate)) * 1e9)
		cumulativeDuration = cumulativeDuration + frameDuration

		// Drop frames if they come before the start point
		if cumulativeDuration < startAfter {
			meta["framesDropped"] = meta["framesDropped"] + 1
			continue
		}

		// If we have a finish point
		if endAt > 0 && meta["predictedFrames"] > 0 {
			// Calculate the last frame number we should consider
			stopPoint := meta["predictedFrames"] - int64(endAt/frameDuration)

			// Drop any frames thereafter
			if meta["framesEncountered"] > stopPoint {
				meta["framesDropped"] = meta["framesDropped"] + 1
				continue
			}
		}

		// If we've got this far, we're in between start and end points so
		// Keep track of the frame stats
		meta["framesIncluded"] = meta["framesIncluded"] + 1
		meta["outputBytes"] = meta["outputBytes"] + int64(frame.FrameLength)
		outputDuration = outputDuration + frameDuration

		// And write the frame to the out file
		_, err := out.Write(frame.RawBytes)
		if err != nil {
			return meta, err
		}

	}

	meta["outputDuration"] = int64(outputDuration.Seconds())
	meta["cumulativeDuration"] = int64(cumulativeDuration.Seconds())

	return meta, nil

}

func line() {
	fmt.Println("------------------------------------------------------------------------------------")
}

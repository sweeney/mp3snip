# mp3snip

MP3Snip is a crufty little tool to lop the beginning and end off of mp3 files without re-encoding. It preserves ID3v2 tags in the resulting, snipped, file.

	$ mp3snip --help

	Usage of mp3snip:
		-end string
    		End before; Duration, parsable by go https://golang.org/pkg/time/#ParseDuration - eg 10s (default "9s")
  		-in string
    		Path to input mp3 file (default "in.mp3")
  		-out string
    		Path to output mp3 file (default "out.mp3")
 		-start string
    		Start after; Duration, parsable by go https://golang.org/pkg/time/#ParseDuration - eg 25s (default "25s")

## Installation

	go install github.com/sweeney/mp3snip
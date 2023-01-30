package mp3joiner

import (
	"bytes"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

var FFMPEG_STATS_REGEX = regexp.MustCompile(`.+time=(?:.*)([0-9]{2,99}):([0-9]{2}):([0-9]{2}).([0-9]{2})`)
var random = rand.New(rand.NewSource(time.Now().UnixNano()))

type MP3Container struct {
	streams []*ffmpeg.Stream
}

func NewMP3() *MP3Container {
	return &MP3Container{
		streams: make([]*ffmpeg.Stream, 0),
	}
}

func GetMP3Metadata() {

}

func SetMP3Metadata() error {
	return nil
}

func (c *MP3Container) Persist(path string) (err error) {
	if len(c.streams) < 1 {
		return fmt.Errorf("no streams to persist")
	}
	// set 0 video stream and 1 audio stream
	err = ffmpeg.Concat(c.streams, ffmpeg.KwArgs{"a": 1, "v": 0}).Output(path).Run()
	return err
}

func (c *MP3Container) AddSection(mp3Filepath string, startInSeconds int, endInSeconds int) (err error) {
	// input validation test
	if endInSeconds != -1 && startInSeconds > endInSeconds {
		return fmt.Errorf("start %v set after end %v", startInSeconds, endInSeconds)
	}

	// set end to last position
	length, err := getLengthInSeconds(mp3Filepath)
	if err != nil {
		return err
	}
	endPos := length
	// set defined pos is not set to -1 end and end is in length of mp3
	if endInSeconds != -1 && endInSeconds < int(length) {
		endPos = float64(endInSeconds)
	}

	// ffmpeg -ss 3 -t 5 -i input.mp3
	input := ffmpeg.Input(mp3Filepath, ffmpeg.KwArgs{"ss": startInSeconds, "t": endPos - float64(startInSeconds)})

	c.streams = append(c.streams, input)
	return err
}

func getLengthInSeconds(mp3Filepath string) (float64, error) {
	outputBuffer := new(bytes.Buffer)

	// ffmpeg -f null - -stats -v quiet -i input.mp3
	ffmpeg.Input(mp3Filepath, ffmpeg.KwArgs{"v": "quiet", "format": "null", "stats": "", "": ""}).
		WithErrorOutput(outputBuffer).Run()

	// expected is a multi line output lik this:
	// size=N/A time=00:00:00.00 bitrate=N/A speed=   0x
	// size=N/A time=00:17:05.36 bitrate=N/A speed=2.05e+03x
	// size=N/A time=00:17:39.89 bitrate=N/A speed=2.05e+03x
	outputString := outputBuffer.String()
	matches := FFMPEG_STATS_REGEX.FindStringSubmatch(outputString)
	if len(matches) != 5 {
		return -1, fmt.Errorf("did not find time in '%s'", outputString)
	}

	hours, err := strconv.Atoi(matches[1])
	if err != nil {
		return -1, err
	}
	minutes, err := strconv.Atoi(matches[2])
	if err != nil {
		return -1, err
	}
	second, err := strconv.Atoi(matches[3])
	if err != nil {
		return -1, err
	}
	milliseconds, err := strconv.Atoi(matches[4])
	if err != nil {
		return -1, err
	}
	result := (hours * 60 * 60) + (minutes * 60) + (second)

	return float64(result) + (float64(milliseconds) * 0.01), nil
}

func getChapterMetadata(path string, start float32, end float32) ([]string, error) {
	return nil, nil
}

func mergeSections([]string) []string {
	return nil
}

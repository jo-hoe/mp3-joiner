package internal

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

var FFMPEG_STATS_REGEX = regexp.MustCompile(`.+time=(?:.*)([0-9]{2,99}):([0-9]{2}):([0-9]{2}).([0-9]{2})`)

func GetMP3Metadata() {

}

func SetMP3Metadata() error {
	return nil
}

func GetLengthInSeconds(mp3Filepath string) (float64, error) {
	return parseMP3Length(getFFmpegStats(mp3Filepath))
}

func getFFmpegStats(mp3Filepath string) string {
	outputBuffer := new(bytes.Buffer)

	// ffmpeg -f null - -stats -v quiet -i input.mp3
	ffmpeg.Input(mp3Filepath, ffmpeg.KwArgs{"v": "quiet", "format": "null", "stats": "", "": ""}).
		WithErrorOutput(outputBuffer).Run()

	return outputBuffer.String()
}

func parseMP3Length(ffmpegStats string) (float64, error) {
	// expected is a multi line output lik this:
	// size=N/A time=00:00:00.00 bitrate=N/A speed=   0x
	// size=N/A time=00:17:05.36 bitrate=N/A speed=2.05e+03x
	// size=N/A time=00:17:39.89 bitrate=N/A speed=2.05e+03x
	matches := FFMPEG_STATS_REGEX.FindStringSubmatch(ffmpegStats)
	if len(matches) != 5 {
		return -1, fmt.Errorf("did not find time in '%s'", ffmpegStats)
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

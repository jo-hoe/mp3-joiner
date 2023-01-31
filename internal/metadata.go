package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

var FFMPEG_STATS_REGEX = regexp.MustCompile(`.+time=(?:.*)([0-9]{2,99}):([0-9]{2}):([0-9]{2}).([0-9]{2})`)

type metadata struct {
	Format struct {
		Tags map[string]string `json:"tags,omitempty"`
	} `json:"format,omitempty"`
}

type chapters struct {
	Chapters []Chapter `json:"chapters,omitempty"`
}

type Chapter struct {
	Id        int    `json:"id,omitempty"`
	TimeBase  string `json:"time_base,omitempty"`
	Start     int    `json:"start,omitempty"`
	StartTime string `json:"start_time,omitempty"`
	End       int    `json:"end,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
	Tags      Tags   `json:"tags,omitempty"`
}

type Tags struct {
	Title string `json:"title,omitempty"`
}

func GetMP3Metadata(mp3Filepath string) (result map[string]string, err error) {
	var data metadata
	// ffprobe -hide_banner -v 0 -i input.mp3 -print_format json -show_chapters
	err = ffprobe(mp3Filepath, ffmpeg.KwArgs{"hide_banner": "", "v": 0, "show_entries": "format", "of": "json"}, &data)
	result = data.Format.Tags
	return result, err
}

func GetChapterMetadata(mp3Filepath string) (result []Chapter, err error) {
	var data chapters
	// ffprobe -hide_banner -v 0 -i input.mp3 -print_format json -show_chapters
	err = ffprobe(mp3Filepath, ffmpeg.KwArgs{"hide_banner": "", "v": 0, "print_format": "json", "show_chapters": ""}, &data)
	result = data.Chapters
	// sort by start
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Start < result[j].Start
	})
	return result, err
}

func ffprobe(mp3Filepath string, args ffmpeg.KwArgs, v any) (err error) {
	output, err := ffmpeg.Probe(mp3Filepath, args)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(output), v)
}

func SetMP3Metadata() error {
	// https://ffmpeg.org/ffmpeg-formats.html#Metadata-1
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

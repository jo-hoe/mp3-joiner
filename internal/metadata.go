package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

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

func SetMP3Metadata(mp3Filepath string, metadata map[string]string, chapters []Chapter) error {
	// ffmpeg -i INPUT -i FFMETADATAFILE -map_metadata 1 -codec copy OUTPUT
	return nil
}

// Creates an meta data file in the temp folder.
// This file format is described here:
// https://ffmpeg.org/ffmpeg-formats.html#Metadata-1
func createTempMetadataFile(metadata map[string]string, chapters []Chapter) (metadataFilepath string, err error) {
	// TODO: The header is a ‘;FFMETADATA’ string, followed by a version number (now 1).
	// TODO: Metadata keys or values containing special characters (‘=’, ‘;’, ‘#’, ‘\’ and a newline) must be escaped with a backslash ‘\’.
	tempFile, err := os.CreateTemp("", "ffmpegMetaData")
	if err != nil {
		return "", err
	}
        // TODO: close temp file here
	metadataFilepath = tempFile.Name()

	var stringBuilder strings.Builder

	for key, value := range metadata {
		stringBuilder.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	if len(chapters) > 0 {
		for _, chapter := range chapters {
			stringBuilder.WriteString("[CHAPTER]\n")
			stringBuilder.WriteString(fmt.Sprintf("TIMEBASE=%s\n", chapter.TimeBase))
			stringBuilder.WriteString(fmt.Sprintf("START=%d\n", chapter.Start))
			stringBuilder.WriteString(fmt.Sprintf("END=%d\n", chapter.End))
			stringBuilder.WriteString(fmt.Sprintf("title=%s\n", chapter.Tags.Title))
		}
	}

	// TODO: A section starts with the section name in uppercase (i.e. STREAM or CHAPTER) in brackets (‘[’, ‘]’) and ends with next section or end of file.
	_, err = tempFile.WriteString(stringBuilder.String())
	return metadataFilepath, err
}

func GetLengthInSeconds(mp3Filepath string) (result float64, err error) {
	output, err := getFFmpegStats(mp3Filepath)
	if err != nil {
		return -1, err
	}

	return parseMP3Length(output)
}

func getFFmpegStats(mp3Filepath string) (output string, err error) {
	outputBuffer := new(bytes.Buffer)

	// ffmpeg -f null - -stats -v quiet -i input.mp3
	err = ffmpeg.Input(mp3Filepath, ffmpeg.KwArgs{"v": "quiet", "format": "null", "stats": "", "": ""}).
		WithErrorOutput(outputBuffer).Run()

	return outputBuffer.String(), err
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

	return float64(result) + (float64(milliseconds) * 0.01), err
}

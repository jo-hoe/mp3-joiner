package mp3joiner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

var (
	ILLEGAL_METADATA_CHARATERS = regexp.MustCompile(`(#|;|=|\\)`)
	FFMPEG_STATS_REGEX         = regexp.MustCompile(`.+time=(?:.*)([0-9]{2,99}):([0-9]{2}):([0-9]{2}).([0-9]{2})`)
	random                     = rand.New(rand.NewSource(time.Now().UnixNano()))
)

type chapters struct {
	Chapters []Chapter `json:"chapters,omitempty"`
}

type filemetadata struct {
	Streams []stream `json:"streams,omitempty"`
}

type stream struct {
	Bitrate string `json:"bit_rate,omitempty"`
}

// Gets a map of ffmpeg MP3 metadata tags. Note that the ID3 tags
// and ffmpeg tags are not equivalent. See this documentation for
// the mapping:
// https://wiki.multimedia.cx/index.php/FFmpeg_Metadata#MP3
func GetFFmpegMetadataTag(mp3Filepath string) (result map[string]string, err error) {
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

// This function does not read the length from the metadata of the file,
// as metadata and actual length can be inconsistent. Instead this implementation
// decodes the file and returns the actual length of the audio stream.
// This is slower but more accurate then reading the length from the metadata.
func GetLengthInSeconds(mp3Filepath string) (result float64, err error) {
	output, err := getFFmpegStats(mp3Filepath)
	if err != nil {
		return -1, err
	}

	return parseMP3Length(output)
}

func GetBitrate(mp3Filepath string) (result int, err error) {
	var bitrate filemetadata
	result = -1

	// ffprobe -i .\input.mp3 -v 0 -show_entries stream=bit_rate -print_format json
	err = ffprobe(mp3Filepath, ffmpeg.KwArgs{"v": 0, "show_entries": "stream", "print_format": "json"}, &bitrate)
	if err != nil {
		return result, err
	}

	return strconv.Atoi(bitrate.Streams[0].Bitrate)
}

// Sets FFmpeg MP3 metadata tag. Note that the ID3 tags and
// ffmpeg tags are not equivalent. See this documentation
// for the mapping:
// https://wiki.multimedia.cx/index.php/FFmpeg_Metadata#MP3
//
// This function creates a new temp file and replaces the initial file.
func SetFFmpegMetadataTag(mp3Filepath string, metadata map[string]string, chapters []Chapter) (err error) {
	bitrate, err := GetBitrate(mp3Filepath)
	if err != nil {
		return err
	}
	return setMetadataWithBitrate(mp3Filepath, metadata, chapters, bitrate)
}

func ffprobe(mp3Filepath string, args ffmpeg.KwArgs, v any) (err error) {
	output, err := ffmpeg.Probe(mp3Filepath, args)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(output), v)
}

func setMetadataWithBitrate(mp3Filepath string, metadata map[string]string, chapters []Chapter, bitrate int) (err error) {
	tempMetadataFile, err := createTempMetadataFile(metadata, chapters)
	if err != nil {
		return err
	}
	defer deleteFile(tempMetadataFile)

	tempFile := filepath.Join(os.TempDir(), strconv.Itoa(random.Intn(9999999999999))+".mp3")

	// ffmpeg -i INPUT.mp3 -i MATADATA -map_chapters 1 -map_metadata 1 -b:a 32k -codec copy OUTPUT.mp3
	mp3Input := ffmpeg.Input(mp3Filepath)
	metadataInput := ffmpeg.Input(tempMetadataFile)
	command := ffmpeg.Output([]*ffmpeg.Stream{mp3Input, metadataInput}, tempFile, ffmpeg.KwArgs{"map_metadata": "1", "map_chapters": "1", "b:a": fmt.Sprintf("%dk", int(bitrate/1000)), "codec": "copy"}).
		Compile()
	// removed unused map parameters
	command.Args = removeParameters(command.Args, "-map", `^[0-9]{0,10}$`)
	err = command.Run()
	if err != nil {
		return err
	}
	defer deleteFile(tempFile)

	return overwriteFile(tempFile, mp3Filepath)
}

func overwriteFile(inputFilePath, outputFilePath string) (err error) {
	targetFile, err := os.OpenFile(outputFilePath, os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer closeFile(targetFile)
	sourceFile, err := os.Open(inputFilePath)
	if err != nil {
		return err
	}
	defer closeFile(sourceFile)
	_, err = io.Copy(targetFile, sourceFile)

	return err
}

// Creates an meta data file in the temp folder.
// This file format is described here:
// https://ffmpeg.org/ffmpeg-formats.html#Metadata-1
func createTempMetadataFile(metadata map[string]string, chapters []Chapter) (metadataFilepath string, err error) {
	tempFile, err := os.CreateTemp("", "ffmpegMetaData")
	if err != nil {
		return "", err
	}
	defer func() {
		if _, checkerr := os.Stat(tempFile.Name()); checkerr == nil {
			err = tempFile.Close()
		}
	}()
	metadataFilepath = tempFile.Name()

	var stringBuilder strings.Builder
	stringBuilder.WriteString(";FFMETADATA1")

	for key, value := range metadata {
		stringBuilder.WriteString(fmt.Sprintf("\n%s=%s", sanitizeMetadata(key), sanitizeMetadata(value)))
	}

	if len(chapters) > 0 {
		for _, chapter := range chapters {
			stringBuilder.WriteString("\n[CHAPTER]")
			stringBuilder.WriteString(fmt.Sprintf("\nTIMEBASE=%s", sanitizeMetadata(chapter.TimeBase)))
			stringBuilder.WriteString(fmt.Sprintf("\nSTART=%d", chapter.Start))
			stringBuilder.WriteString(fmt.Sprintf("\nEND=%d", chapter.End))
			stringBuilder.WriteString(fmt.Sprintf("\ntitle=%s", sanitizeMetadata(chapter.Tags.Title)))
		}
	}

	_, err = tempFile.WriteString(stringBuilder.String())
	return metadataFilepath, err
}

// Metadata keys or values containing special characters
// (‘=’, ‘;’, ‘#’, ‘\’ and a newline) will escaped with a
// backslash ‘\’.
func sanitizeMetadata(input string) (output string) {
	// make string "unescaped" not efficent but quick to implement
	// better would be to look ahead and look behind chars to escape
	// and only handle these characters
	output = strings.Replace(input, "\\\\", "\\", -1)
	output = strings.Replace(output, "\\=", "=", -1)
	output = strings.Replace(output, "\\;", ";", -1)
	output = strings.Replace(output, "\\#", "#", -1)

	// escape complete string
	matches := ILLEGAL_METADATA_CHARATERS.FindAllStringIndex(output, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		output = output[:matches[i][0]] + "\\" + output[matches[i][0]:]
	}

	return output
}

func getFFmpegStats(mp3Filepath string) (output string, err error) {
	outputBuffer := new(bytes.Buffer)

	// this command will resolve to:
	// ffmpeg -map 0:a -f null - -stats -v quiet -i input.mp3
	// -f null - : specifies that there should not be an output file and the output should be redirected to stdout
	command := ffmpeg.Input(mp3Filepath, ffmpeg.KwArgs{"v": "quiet", "format": "null", "stats": "", "": ""}).
		WithErrorOutput(outputBuffer).Compile()
	// injects -map 0:a
	// with this inject the returned length directly depends on length of audio stream
	modifiedArgs := append([]string{"ffmpeg", "-map", "0:a"}, command.Args[1:]...)
	command.Args = modifiedArgs
	err = command.Run()

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

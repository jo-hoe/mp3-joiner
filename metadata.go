package mp3joiner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	illegalMetadataChars = regexp.MustCompile(`(#|;|=|\\)`)
	ffmpegStatsRegex     = regexp.MustCompile(`.+time=(?:.*)([0-9]{2,99}):([0-9]{2}):([0-9]{2}).([0-9]{2})`)
)

type ffprobeChapters struct {
	Chapters []ffprobeChapter `json:"chapters,omitempty"`
}

type ffprobeStreams struct {
	Streams []ffprobeStream `json:"streams,omitempty"`
}

type ffprobeStream struct {
	Bitrate string `json:"bit_rate,omitempty"`
}

// GetFFmpegMetadataTag returns a map of ffmpeg MP3 metadata tags.
// Note that ID3 tags and ffmpeg tags are not equivalent; see:
// https://wiki.multimedia.cx/index.php/FFmpeg_Metadata#MP3
func GetFFmpegMetadataTag(mp3Filepath string) (map[string]string, error) {
	var data ffprobeFormat
	args := []string{"-hide_banner", "-v", "0", "-show_entries", "format", "-of", "json"}
	if err := ffprobe(mp3Filepath, args, &data); err != nil {
		return nil, err
	}
	return data.Format.Tags, nil
}

// GetChapterMetadata returns the chapters embedded in an MP3 file, sorted by start time.
func GetChapterMetadata(mp3Filepath string) ([]Chapter, error) {
	var data ffprobeChapters
	args := []string{"-hide_banner", "-v", "0", "-print_format", "json", "-show_chapters"}
	if err := ffprobe(mp3Filepath, args, &data); err != nil {
		return nil, err
	}
	result := make([]Chapter, len(data.Chapters))
	for i, c := range data.Chapters {
		result[i] = c.toChapter()
	}
	sortChaptersByStart(result)
	return result, nil
}

// GetLengthInSeconds returns the actual decoded length of the audio stream.
// It decodes the file rather than reading metadata, which is slower but accurate.
func GetLengthInSeconds(mp3Filepath string) (float64, error) {
	output, err := getFFmpegStats(mp3Filepath)
	if err != nil {
		return -1, err
	}
	return parseMP3Length(output)
}

// GetBitrate returns the bitrate of the first audio stream in bits per second.
func GetBitrate(mp3Filepath string) (int, error) {
	var data ffprobeStreams
	args := []string{"-v", "0", "-show_entries", "stream=bit_rate", "-print_format", "json"}
	if err := ffprobe(mp3Filepath, args, &data); err != nil {
		return 0, err
	}
	if len(data.Streams) == 0 {
		return 0, fmt.Errorf("no audio streams found in %s", mp3Filepath)
	}
	return strconv.Atoi(data.Streams[0].Bitrate)
}

// SetFFmpegMetadataTag writes metadata tags and chapters to an MP3 file in-place.
// Note that ID3 tags and ffmpeg tags are not equivalent; see:
// https://wiki.multimedia.cx/index.php/FFmpeg_Metadata#MP3
func SetFFmpegMetadataTag(mp3Filepath string, tags map[string]string, chapters []Chapter) error {
	bitrate, err := GetBitrate(mp3Filepath)
	if err != nil {
		return err
	}
	return setMetadataWithBitrate(mp3Filepath, tags, chapters, bitrate)
}

func ffprobe(mp3Filepath string, extraArgs []string, v any) error {
	args := make([]string, 0, len(extraArgs)+2)
	args = append(args, extraArgs...)
	args = append(args, "-i", mp3Filepath)
	output, err := runCmd("ffprobe", args...)
	if err != nil {
		return fmt.Errorf("ffprobe failed: %w - output: %s", err, output)
	}
	return json.Unmarshal([]byte(output), v)
}

func setMetadataWithBitrate(mp3Filepath string, tags map[string]string, chapters []Chapter, bitrate int) error {
	tempMetadataFile, err := createTempMetadataFile(tags, chapters)
	if err != nil {
		return err
	}
	defer deleteFile(tempMetadataFile)

	tempFile, err := os.CreateTemp("", "*.mp3")
	if err != nil {
		return err
	}
	tempFilePath := tempFile.Name()
	closeFile(tempFile)
	defer deleteFile(tempFilePath)

	args := []string{
		"-y",
		"-i", mp3Filepath,
		"-i", tempMetadataFile,
		"-map_metadata", "1",
		"-map_chapters", "1",
		"-b:a", fmt.Sprintf("%dk", bitrate/1000),
		"-codec", "copy",
		tempFilePath,
	}
	if output, errRun := runCmd("ffmpeg", args...); errRun != nil {
		return fmt.Errorf("ffmpeg metadata set failed: %w - output: %s", errRun, output)
	}

	return overwriteFile(tempFilePath, mp3Filepath)
}

func overwriteFile(inputFilePath, outputFilePath string) error {
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

// createTempMetadataFile writes an ffmpeg metadata file to a temp path and returns it.
// Format: https://ffmpeg.org/ffmpeg-formats.html#Metadata-1
func createTempMetadataFile(tags map[string]string, chapters []Chapter) (metadataFilepath string, err error) {
	tempFile, err := os.CreateTemp("", "ffmpegMetaData")
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := tempFile.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	metadataFilepath = tempFile.Name()

	var sb strings.Builder
	sb.WriteString(";FFMETADATA1")
	for key, value := range tags {
		fmt.Fprintf(&sb, "\n%s=%s", sanitizeMetadata(key), sanitizeMetadata(value))
	}
	for _, chapter := range chapters {
		sb.WriteString("\n[CHAPTER]")
		fmt.Fprintf(&sb, "\nTIMEBASE=%s", sanitizeMetadata(chapter.TimeBase))
		fmt.Fprintf(&sb, "\nSTART=%d", chapter.Start)
		fmt.Fprintf(&sb, "\nEND=%d", chapter.End)
		fmt.Fprintf(&sb, "\ntitle=%s", sanitizeMetadata(chapter.Tags.Title))
	}

	_, err = tempFile.WriteString(sb.String())
	return metadataFilepath, err
}

// sanitizeMetadata escapes characters that are special in ffmpeg metadata files
// ('=', ';', '#', '\') with a backslash.
func sanitizeMetadata(input string) string {
	// Unescape any already-escaped sequences to avoid double-escaping.
	output := strings.ReplaceAll(input, "\\\\", "\\")
	output = strings.ReplaceAll(output, "\\=", "=")
	output = strings.ReplaceAll(output, "\\;", ";")
	output = strings.ReplaceAll(output, "\\#", "#")

	matches := illegalMetadataChars.FindAllStringIndex(output, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		output = output[:matches[i][0]] + "\\" + output[matches[i][0]:]
	}
	return output
}

func getFFmpegStats(mp3Filepath string) (string, error) {
	args := []string{
		"-i", mp3Filepath,
		"-map", "0:a",
		"-f", "null", "-",
		"-stats",
		"-v", "quiet",
	}
	return runCmd("ffmpeg", args...)
}

func parseMP3Length(ffmpegStats string) (float64, error) {
	matches := ffmpegStatsRegex.FindStringSubmatch(ffmpegStats)
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
	seconds, err := strconv.Atoi(matches[3])
	if err != nil {
		return -1, err
	}
	milliseconds, err := strconv.Atoi(matches[4])
	if err != nil {
		return -1, err
	}
	total := (hours * 3600) + (minutes * 60) + seconds
	return float64(total) + float64(milliseconds)*0.01, nil
}

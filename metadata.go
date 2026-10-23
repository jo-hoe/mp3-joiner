package mp3joiner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const ffprobeOutputFormat = "json"

var ffmpegStatsRegex = regexp.MustCompile(`.+time=(?:.*)(\d{2,99}):(\d{2}):(\d{2}).(\d{2})`)

type ffprobeChapters struct {
	Chapters []ffprobeChapter `json:"chapters,omitempty"`
}

type ffprobeStreams struct {
	Streams []ffprobeStream `json:"streams,omitempty"`
}

type ffprobeStream struct {
	Bitrate string `json:"bit_rate,omitempty"`
}

type ffprobeAllData struct {
	Streams  []ffprobeStream  `json:"streams,omitempty"`
	Format   struct {
		Tags map[string]string `json:"tags,omitempty"`
	} `json:"format,omitempty"`
	Chapters []ffprobeChapter `json:"chapters,omitempty"`
}

// GetFFmpegMetadataTag returns a map of ffmpeg MP3 metadata tags.
// Note that ID3 tags and ffmpeg tags are not equivalent; see:
// https://wiki.multimedia.cx/index.php/FFmpeg_Metadata#MP3
func GetFFmpegMetadataTag(mp3Filepath string) (map[string]string, error) {
	var data ffprobeFormat
	args := []string{"-hide_banner", "-v", "0", "-show_entries", "format", "-of", ffprobeOutputFormat}
	if err := ffprobe(mp3Filepath, args, &data); err != nil {
		return nil, err
	}
	return data.Format.Tags, nil
}

// GetChapterMetadata returns the chapters embedded in an MP3 file, sorted by start time.
func GetChapterMetadata(mp3Filepath string) ([]Chapter, error) {
	var data ffprobeChapters
	args := []string{"-hide_banner", "-v", "0", "-print_format", ffprobeOutputFormat, "-show_chapters"}
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
	bitrate, err := strconv.Atoi(data.Streams[0].Bitrate)
	if err != nil {
		return 0, fmt.Errorf("failed to parse bitrate: %w", err)
	}
	return bitrate, nil
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
	if err := json.Unmarshal([]byte(output), v); err != nil {
		return fmt.Errorf("failed to parse ffprobe output: %w", err)
	}
	return nil
}

func getFileMetadata(mp3Filepath string) ([]Chapter, map[string]string, int, error) {
	var data ffprobeAllData
	args := []string{
		"-hide_banner", "-v", "0",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-show_chapters",
	}
	if err := ffprobe(mp3Filepath, args, &data); err != nil {
		return nil, nil, 0, err
	}
	chapters := make([]Chapter, len(data.Chapters))
	for i, c := range data.Chapters {
		chapters[i] = c.toChapter()
	}
	sortChaptersByStart(chapters)
	if len(data.Streams) == 0 {
		return nil, nil, 0, fmt.Errorf("no audio streams found in %s", mp3Filepath)
	}
	bitrate, err := strconv.Atoi(data.Streams[0].Bitrate)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to parse bitrate: %w", err)
	}
	return chapters, data.Format.Tags, bitrate, nil
}

func setMetadataWithBitrate(mp3Filepath string, tags map[string]string, chapters []Chapter, _ int) error {
	tempMetadataFile, err := createTempMetadataFile(tags, chapters)
	if err != nil {
		return err
	}
	defer deleteFile(tempMetadataFile)

	tempFile, err := os.CreateTemp("", "*.mp3")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
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
		"-codec", "copy",
		tempFilePath,
	}
	if output, errRun := runCmd("ffmpeg", args...); errRun != nil {
		return fmt.Errorf("ffmpeg metadata set failed: %w - output: %s", errRun, output)
	}

	return overwriteFile(tempFilePath, mp3Filepath)
}

func overwriteFile(inputFilePath, outputFilePath string) error {
	if err := os.Rename(inputFilePath, outputFilePath); err == nil {
		return nil
	}
	// Rename across filesystems fails; fall back to copy-then-replace.
	// Write to a sibling temp file first so the original is never truncated
	// before the new content is fully written.
	dir := filepath.Dir(outputFilePath)
	tmpFile, err := os.CreateTemp(dir, "overwrite-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file for overwrite: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		if _, serr := os.Stat(tmpPath); serr == nil {
			_ = os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup
		}
	}()

	sourceFile, err := os.Open(inputFilePath) //nolint:gosec // path is controlled by internal callers only
	if err != nil {
		closeFile(tmpFile)
		return fmt.Errorf("failed to open input file: %w", err)
	}
	if _, err = io.Copy(tmpFile, sourceFile); err != nil {
		closeFile(tmpFile)
		closeFile(sourceFile)
		return fmt.Errorf("failed to copy file: %w", err)
	}
	closeFile(sourceFile)
	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to flush temp file: %w", err)
	}
	if err = os.Rename(tmpPath, outputFilePath); err != nil {
		return fmt.Errorf("failed to replace output file: %w", err)
	}
	return nil
}

// createTempMetadataFile writes an ffmpeg metadata file to a temp path and returns it.
// Format: https://ffmpeg.org/ffmpeg-formats.html#Metadata-1
func createTempMetadataFile(tags map[string]string, chapters []Chapter) (metadataFilepath string, err error) {
	tempFile, err := os.CreateTemp("", "ffmpegMetaData")
	if err != nil {
		return "", fmt.Errorf("failed to create temp metadata file: %w", err)
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

	if _, err = tempFile.WriteString(sb.String()); err != nil {
		return metadataFilepath, fmt.Errorf("failed to write metadata file: %w", err)
	}
	return metadataFilepath, nil
}

// sanitizeMetadata escapes characters that are special in ffmpeg metadata files
// ('=', ';', '#', '\') with a backslash.
func sanitizeMetadata(input string) string {
	var sb strings.Builder
	sb.Grow(len(input))
	for i := 0; i < len(input); i++ {
		c := input[i]
		if c == '\\' && i+1 < len(input) {
			next := input[i+1]
			if next == '\\' || next == '=' || next == ';' || next == '#' {
				// already a valid escape sequence — copy it verbatim
				sb.WriteByte(c)
				sb.WriteByte(next)
				i++
				continue
			}
		}
		if c == '#' || c == ';' || c == '=' || c == '\\' {
			sb.WriteByte('\\')
		}
		sb.WriteByte(c)
	}
	return sb.String()
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
		return -1, fmt.Errorf("failed to parse hours: %w", err)
	}
	minutes, err := strconv.Atoi(matches[2])
	if err != nil {
		return -1, fmt.Errorf("failed to parse minutes: %w", err)
	}
	seconds, err := strconv.Atoi(matches[3])
	if err != nil {
		return -1, fmt.Errorf("failed to parse seconds: %w", err)
	}
	milliseconds, err := strconv.Atoi(matches[4])
	if err != nil {
		return -1, fmt.Errorf("failed to parse milliseconds: %w", err)
	}
	total := (hours * 3600) + (minutes * 60) + seconds
	return float64(total) + float64(milliseconds)*0.01, nil
}

package mp3joiner

import (
	"fmt"
	"strconv"
	"strings"
)

// EndOfFile is passed as endInSeconds to Append to read to the end of the file.
const EndOfFile float64 = -1

type segment struct {
	file     string
	start    float64
	duration float64
}

// MP3Builder accumulates MP3 segments and builds a merged output file.
type MP3Builder struct {
	streams             []segment
	chapters            []Chapter
	metaData            map[string]string
	bitrate             int
	accumulatedDuration float64
}

// NewMP3Builder returns a new MP3Builder.
func NewMP3Builder() *MP3Builder {
	return &MP3Builder{
		streams: make([]segment, 0),
	}
}

// Build writes the merged MP3 to filePath.
func (b *MP3Builder) Build(filePath string) error {
	if len(b.streams) < 1 {
		return fmt.Errorf("no streams to persist")
	}

	b.chapters = mergeChapters(b.chapters)
	tempMetadataFile, err := createTempMetadataFile(b.metaData, b.chapters)
	if err != nil {
		return err
	}
	defer deleteFile(tempMetadataFile)

	args := make([]string, 0, 32+(len(b.streams)*6))
	args = append(args, "-y") // overwrite output if it exists
	for _, s := range b.streams {
		args = append(args,
			"-ss", formatSeconds(s.start),
			"-t", formatSeconds(s.duration),
			"-i", s.file,
		)
	}
	args = append(args, "-i", tempMetadataFile)

	var sb strings.Builder
	for i := range b.streams {
		sb.WriteString("[" + strconv.Itoa(i) + ":a]")
	}
	fmt.Fprintf(&sb, "concat=n=%d:v=0:a=1[aout]", len(b.streams))
	args = append(args, "-filter_complex", sb.String())
	args = append(args, "-map", "[aout]")

	metadataIndex := strconv.Itoa(len(b.streams))
	args = append(args,
		"-map_metadata", metadataIndex,
		"-map_chapters", metadataIndex,
		"-c:a", "libmp3lame",
		"-b:a", fmt.Sprintf("%dk", b.bitrate/1000),
		filePath,
	)

	if output, runErr := runCmd("ffmpeg", args...); runErr != nil {
		return fmt.Errorf("ffmpeg build failed: %w - output: %s", runErr, output)
	}
	return nil
}

// Append adds a segment of an MP3 file to the builder.
// Pass EndOfFile as endInSeconds to read to the end of the file.
func (b *MP3Builder) Append(mp3Filepath string, startInSeconds float64, endInSeconds float64) error {
	if endInSeconds != EndOfFile && startInSeconds > endInSeconds {
		return fmt.Errorf("start %v set after end %v", startInSeconds, endInSeconds)
	}

	length, err := GetLengthInSeconds(mp3Filepath)
	if err != nil {
		return err
	}

	endPos := length
	if endInSeconds != EndOfFile && endInSeconds < length {
		endPos = endInSeconds
	}

	allChapters, err := GetChapterMetadata(mp3Filepath)
	if err != nil {
		return err
	}

	chaptersInTimeFrame := getChapterInTimeFrame(allChapters, startInSeconds, endPos)
	for i := range chaptersInTimeFrame {
		chaptersInTimeFrame[i].shiftBy(b.accumulatedDuration - startInSeconds)
	}
	b.chapters = append(b.chapters, chaptersInTimeFrame...)

	duration := endPos - startInSeconds
	if duration < 0 {
		return fmt.Errorf("calculated negative duration")
	}
	b.streams = append(b.streams, segment{
		file:     mp3Filepath,
		start:    startInSeconds,
		duration: duration,
	})
	b.accumulatedDuration += duration

	if b.metaData == nil {
		tags, err := GetFFmpegMetadataTag(mp3Filepath)
		if err != nil {
			return err
		}
		b.metaData = tags
	}

	bitrate, err := GetBitrate(mp3Filepath)
	if err != nil {
		return err
	}
	if bitrate > b.bitrate {
		b.bitrate = bitrate
	}

	return nil
}

func formatSeconds(v float64) string {
	return strconv.FormatFloat(v, 'f', 3, 64)
}

package mp3joiner

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type segment struct {
	File     string
	Start    float64
	Duration float64
}

type MP3Builder struct {
	streams  []segment
	chapters []Chapter
	metaData map[string]string
	bitrate  int
}

// Builder that holds the added MP3 sections
func NewMP3Builder() *MP3Builder {
	return &MP3Builder{
		streams: make([]segment, 0),
	}
}

// Creates the MP3 file a the chosen path
func (b *MP3Builder) Build(filePath string) (err error) {
	if len(b.streams) < 1 {
		return fmt.Errorf("no streams to persist")
	}

	b.chapters = mergeChapters(b.chapters)
	tempMetadataFile, err := createTempMetadataFile(b.metaData, b.chapters)
	if err != nil {
		return err
	}
	defer func() {
		deleteFile(tempMetadataFile)
		// check if copying was successful
		if err != nil {
			return
		}
	}()

	// Build ffmpeg args to trim inputs and concat
	args := make([]string, 0, 32+(len(b.streams)*6))
	for _, s := range b.streams {
		args = append(args,
			"-ss", formatSeconds(s.Start),
			"-t", formatSeconds(s.Duration),
			"-i", s.File,
		)
	}

	// Add metadata ffmetadata input; index is after the N audio inputs
	args = append(args, "-i", tempMetadataFile)

	// Build filter_complex: [0:a][1:a]...concat=n=N:v=0:a=1[aout]
	var sb strings.Builder
	for i := range b.streams {
		sb.WriteString("[" + strconv.Itoa(i) + ":a]")
	}
	sb.WriteString(fmt.Sprintf("concat=n=%d:v=0:a=1[aout]", len(b.streams)))
	filter := sb.String()
	args = append(args, "-filter_complex", filter)
	args = append(args, "-map", "[aout]")
	metadataIndex := len(b.streams) // metadata file comes after N stream inputs
	args = append(args,
		"-map_metadata", strconv.Itoa(metadataIndex),
		"-map_chapters", strconv.Itoa(metadataIndex),
	)

	// Set audio codec/bitrate and output path
	args = append(args,
		"-c:a", "libmp3lame",
		"-b:a", fmt.Sprintf("%dk", int(b.bitrate/1000)),
		filePath,
	)

	cmd := exec.Command("ffmpeg", args...)
	if output, runErr := cmd.CombinedOutput(); runErr != nil {
		return fmt.Errorf("ffmpeg build failed: %w - output: %s", runErr, string(output))
	}
	return nil
}

// Adds a MP3 file to the builder.
// If endInSeconds is set to "-1" the stream will be read until the end of the file.
func (b *MP3Builder) Append(mp3Filepath string, startInSeconds float64, endInSeconds float64) (err error) {
	// input validation test
	if endInSeconds != -1 && startInSeconds > endInSeconds {
		return fmt.Errorf("start %v set after end %v", startInSeconds, endInSeconds)
	}

	// set end to last position
	length, err := GetLengthInSeconds(mp3Filepath)
	if err != nil {
		return err
	}
	endPos := length
	// set defined pos is not set to -1 end and end is in length of mp3
	if endInSeconds != -1 && endInSeconds < length {
		endPos = float64(endInSeconds)
	}

	// retrieve chapters
	allChapters, err := GetChapterMetadata(mp3Filepath)
	if err != nil {
		return err
	}
	chaptersInTimeFrame := getChapterInTimeFrame(allChapters, startInSeconds, endPos)
	b.chapters = chaptersInTimeFrame

	// cache segment definition (use -ss/-t before -i for each segment)
	duration := endPos - startInSeconds
	if duration < 0 {
		return fmt.Errorf("calculated negative duration")
	}
	b.streams = append(b.streams, segment{
		File:     mp3Filepath,
		Start:    startInSeconds,
		Duration: duration,
	})

	if b.metaData == nil {
		metadata, err := GetFFmpegMetadataTag(mp3Filepath)
		if err != nil {
			return err
		}
		b.metaData = metadata
	}
	bitrate, err := GetBitrate(mp3Filepath)
	if err != nil {
		return err
	}
	if bitrate > b.bitrate {
		b.bitrate = bitrate
	}

	return err
}

func formatSeconds(v float64) string {
	// ffmpeg accepts simple decimal seconds
	return strconv.FormatFloat(v, 'f', 3, 64)
}

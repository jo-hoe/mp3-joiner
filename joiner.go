package mp3joiner

import (
	"fmt"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type MP3Builder struct {
	streams  []*ffmpeg.Stream
	chapters []Chapter
	metaData map[string]string
	bitrate  int
}

// Builder that holds the added MP3 sections
func NewMP3Builder() *MP3Builder {
	// stop to log the ffmpeg compiled commands of the lib as
	// some commands get changed before execution
	ffmpeg.LogCompiledCommand = false
	return &MP3Builder{
		streams: make([]*ffmpeg.Stream, 0),
	}
}

func (b *MP3Builder) uniqueStreamSize() int {
	result := 0

	streamHashes := make(map[int]bool)
	for _, stream := range b.streams {
		streamHashes[stream.Hash()] = true
	}
	result = len(streamHashes)

	return result
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

	// -v 0 = set 0 video stream
	// -a 1 = set 1 audio stream
	streams := ffmpeg.Concat(b.streams, ffmpeg.KwArgs{"a": 1, "v": 0})
	metadataInput := ffmpeg.Input(tempMetadataFile)

	numberOfStreams := b.uniqueStreamSize()
	parameters := ffmpeg.KwArgs{
		// set metadata file index to file after streams
		"map_metadata": numberOfStreams,
		"map_chapters": numberOfStreams,
		// set bitrate in 100k format
		"b:a": fmt.Sprintf("%dk", int(b.bitrate/1000)),
	}
	command := ffmpeg.Output([]*ffmpeg.Stream{streams, metadataInput}, filePath, parameters).
		Compile()

	// remove unneeded mapping parameters
	// the ffmpeg lib does not support that out of the box yet
	command.Args = removeParameters(command.Args, "-map", `^[0-9]{0,10}$`)
	return command.Run()
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

	// creates an input stream
	// example command below:
	// ffmpeg -ss 3 -t 5 -i input.mp3
	input := ffmpeg.Input(mp3Filepath, ffmpeg.KwArgs{"ss": startInSeconds, "t": endPos - startInSeconds})
	b.streams = append(b.streams, input)

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

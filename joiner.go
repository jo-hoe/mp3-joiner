package mp3joiner

import (
	"fmt"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type MP3Container struct {
	streams  []*ffmpeg.Stream
	chapters []Chapter
	metaData map[string]string
	bitrate  int
}

func NewMP3() *MP3Container {
	return &MP3Container{
		streams: make([]*ffmpeg.Stream, 0),
	}
}

func (c *MP3Container) Persist(path string) (err error) {
	if len(c.streams) < 1 {
		return fmt.Errorf("no streams to persist")
	}

	c.chapters = mergeChapters(c.chapters)
	tempMetadataFile, err := createTempMetadataFile(c.metaData, c.chapters)
	if err != nil {
		return err
	}
	defer deleteFile(tempMetadataFile)

	// -v 0 = set 0 video stream
	// -a 1 = set 1 audio stream
	streams := ffmpeg.Concat(c.streams, ffmpeg.KwArgs{"a": 1, "v": 0})
	metadataInput := ffmpeg.Input(tempMetadataFile)

	parameters := ffmpeg.KwArgs{
		// set metadata file index to file after streams
		"map_metadata": len(c.streams),
		"map_chapters": len(c.streams),
		// set bitrate in 100k format
		"b:a": fmt.Sprintf("%dk", int(c.bitrate/1000)),
	}
	command := ffmpeg.Output([]*ffmpeg.Stream{streams, metadataInput}, path, parameters).
		Compile()

	// remove unneeded mapping parameters
	// the ffmpeg lib does not support that out of the box yet
	command.Args = removeParameters(command.Args, "-map", `^[0-9]{0,10}$`)
	return command.Run()
}

func (c *MP3Container) AddSection(mp3Filepath string, startInSeconds float64, endInSeconds float64) (err error) {
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
	c.chapters = chaptersInTimeFrame

	// ffmpeg -ss 3 -t 5 -i input.mp3
	input := ffmpeg.Input(mp3Filepath, ffmpeg.KwArgs{"ss": startInSeconds, "t": endPos - startInSeconds})
	c.streams = append(c.streams, input)

	if c.metaData == nil {
		metadata, err := GetMetadata(mp3Filepath)
		if err != nil {
			return err
		}
		c.metaData = metadata
	}
	bitrate, err := GetBitrate(mp3Filepath)
	if err != nil {
		return err
	}
	if bitrate > c.bitrate {
		c.bitrate = bitrate
	}

	return err
}

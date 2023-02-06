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

	// -v 0 = set 0 video stream
	// -a 1 = set 1 audio stream
	err = ffmpeg.Concat(c.streams, ffmpeg.KwArgs{"a": 1, "v": 0}).Output(path, ffmpeg.KwArgs{"b:a": fmt.Sprintf("%dk", int(c.bitrate/1000))}).Run()
	if err != nil {
		return err
	}

	// this creates a new file and replaces the initial file
	// it might be possible to apply the metadata in the step
	// above and remove the need for this time consuming step
	c.chapters = mergeChapters(c.chapters)
	err = setMetadataWithBitrate(path, c.metaData, c.chapters, c.bitrate)

	return err
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
	chaptersInTimeFrame := getChapterInTimeFrame(allChapters, startInSeconds, endInSeconds)
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

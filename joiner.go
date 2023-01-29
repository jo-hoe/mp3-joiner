package mp3joiner

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/hajimehoshi/go-mp3"
)

type MP3Container struct {
	buffer []byte
}

func NewMP3() *MP3Container {
	return &MP3Container{
		buffer: make([]byte, 0),
	}
}

func GetMP3Metadata() {

}

func SetMP3Metadata() error {
	return nil
}

func (c *MP3Container) Persit(path string) (err error) {
	err = ioutil.WriteFile(path, c.buffer, os.ModePerm)
	return err
}

func (c *MP3Container) AddSection(mp3Filepath string, startInSeconds int, endInSeconds int) (err error) {
	// input validation test
	if endInSeconds != -1 && startInSeconds > endInSeconds {
		return fmt.Errorf("start %v set after end %v", startInSeconds, endInSeconds)
	}

	// open and parse mp3 file
	file, err := os.Open(mp3Filepath)
	if err != nil {
		return err
	}
	defer func() {
		if file != nil {
			err = file.Close()
		}
	}()
	mp3File, err := mp3.NewDecoder(file)
	if err != nil {
		return err
	}

	// calculate size of buffer
	// calculated starting point in bytes
	startPos := byteOfSecond(startInSeconds, mp3File.SampleRate())

	// set end to last position
	lastByteIndex := int(mp3File.Length())
	endPosInByte := byteOfSecond(endInSeconds, mp3File.SampleRate())
	endPos := lastByteIndex
	// set defined pos is not set to -1 end and end is in length of mp3
	if endInSeconds != -1 && endPosInByte < lastByteIndex {
		endPos = endPosInByte
	}

	// seek to the start position
	_, err = mp3File.Seek(int64(startPos), io.SeekStart)
	if err != nil {
		return err
	}

	// read all until EOF
	temp := make([]byte, endPos-startPos)
	for i := 0; i < endPos-startPos; {
		readBytes, err := mp3File.Read(temp)
		if err == io.EOF {
			break
		}
		i += readBytes
	}

	// copy item onto the end of the current buffer
	c.buffer = append(c.buffer, temp...)

	return err
}

func byteOfSecond(sec int, freq int) int {
	return sec * freq
}

func getChapterMetadata(path string, start float32, end float32) ([]string, error) {
	return nil, nil
}

func mergeSections([]string) []string {
	return nil
}

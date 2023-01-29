package mp3joiner

import (
	"os"
	"path/filepath"
	"testing"
)

var (
	testFileName = "edgar-allen-poe-the-telltale-heart-original.mp3"
)

func TestMP3Container_AddSection(t *testing.T) {
	type args struct {
		mp3Filepath    string
		startInSeconds int
		endInSeconds   int
	}
	tests := []struct {
		name             string
		c                *MP3Container
		args             args
		wantErr          bool
		wantResultLength int
	}{
		{
			name: "cut first",
			c:    NewMP3(),
			args: args{
				mp3Filepath:    filepath.Join(getMP3TestFolder(t), testFileName),
				startInSeconds: 1,
				endInSeconds:   2,
			},
			wantErr:          false,
			wantResultLength: 22050,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.c.AddSection(tt.args.mp3Filepath, tt.args.startInSeconds, tt.args.endInSeconds); (err != nil) != tt.wantErr {
				t.Errorf("MP3Container.AddSection() error = %v, wantErr %v", err, tt.wantErr)
			}
			actualBufferLength := len(tt.c.buffer)
			if actualBufferLength != tt.wantResultLength {
				t.Errorf("MP3Container.AddSection() expected a result length of %d but actual size was %d", tt.wantResultLength, actualBufferLength)
			}
		})
	}
}

func getMP3TestFolder(t *testing.T) string {
	// get test folder
	testFileFolder, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	return filepath.Join(testFileFolder, "test", "mp3")
}

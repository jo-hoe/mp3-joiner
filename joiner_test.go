package mp3joiner

import (
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

var (
	testFileName = "edgar-allen-poe-the-telltale-heart-original.mp3"
)

var generatedMP3FilePaths = make([]string, 0)
var random = rand.New(rand.NewSource(time.Now().UnixNano()))

type SecondsWindow struct {
	start, end int
}

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
		{
			name: "end before start",
			c:    NewMP3(),
			args: args{
				mp3Filepath:    filepath.Join(getMP3TestFolder(t), testFileName),
				startInSeconds: 1,
				endInSeconds:   0,
			},
			wantErr:          true,
			wantResultLength: 0,
		},
		{
			name: "read until end",
			c:    NewMP3(),
			args: args{
				mp3Filepath:    filepath.Join(getMP3TestFolder(t), testFileName),
				startInSeconds: 4239, //file has ~ 4,239.99 seconds
				endInSeconds:   -1,
			},
			wantErr:          false,
			wantResultLength: 21762,
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

func TestMP3Container_Persist(t *testing.T) {
	t.Cleanup(func() {
		for _, filePath := range generatedMP3FilePaths {
			err := os.Remove(filePath)
			if err != nil {
				t.Errorf("Could not deleted file %s", filePath)
			}
		}
		generatedMP3FilePaths = make([]string, 0)
	})
	container := NewMP3()
	container.AddSection(filepath.Join(getMP3TestFolder(t), testFileName), 0, 5)

	type args struct {
		path string
	}
	tests := []struct {
		name    string
		c       *MP3Container
		args    args
		wantErr bool
	}{
		{
			name: "empty MP3 test",
			c:    createContainer(t, make([]SecondsWindow, 0)),
			args: args{
				path: generateMP3FileName(),
			},
			wantErr: false,
		}, {
			name: "positiv test",
			c: createContainer(t, []SecondsWindow{{
				start: 1,
				end:   10,
			}}),
			args: args{
				path: generateMP3FileName(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.c.Persist(tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("MP3Container.Persist() error = %v, wantErr %v", err, tt.wantErr)
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

func generateMP3FileName() string {
	filePath := filepath.Join(os.TempDir(), strconv.Itoa(random.Intn(9999999999999))+".mp3")
	generatedMP3FilePaths = append(generatedMP3FilePaths, filePath)
	return filePath
}

func createContainer(t *testing.T, windows []SecondsWindow) *MP3Container {
	container := NewMP3()

	for _, window := range windows {
		container.AddSection(filepath.Join(getMP3TestFolder(t), testFileName), window.start, window.end)
	}

	return container
}

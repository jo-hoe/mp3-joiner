package mp3joiner

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

var (
	testFileName = "edgar-allen-poe-the-telltale-heart-original.mp3"
)

type SecondsWindow struct {
	start, end float64
}

func TestMP3Container_AddSection(t *testing.T) {
	type args struct {
		mp3Filepath    string
		startInSeconds float64
		endInSeconds   float64
	}
	tests := []struct {
		name                string
		c                   *MP3Container
		args                args
		wantErr             bool
		streamsCount        int
		approximateFileSize int
	}{
		{
			name: "positive test",
			c:    NewMP3(),
			args: args{
				mp3Filepath:    filepath.Join(getMP3TestFolder(t), testFileName),
				startInSeconds: 1,
				endInSeconds:   2,
			},
			wantErr:      false,
			streamsCount: 1,
		},
		{
			name: "end before start",
			c:    NewMP3(),
			args: args{
				mp3Filepath:    filepath.Join(getMP3TestFolder(t), testFileName),
				startInSeconds: 1,
				endInSeconds:   0,
			},
			wantErr:      true,
			streamsCount: 0,
		},
		{
			name: "non-existing file",
			c:    NewMP3(),
			args: args{
				mp3Filepath:    "dummy",
				startInSeconds: 0,
				endInSeconds:   1,
			},
			wantErr:      true,
			streamsCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.c.AddSection(tt.args.mp3Filepath, tt.args.startInSeconds, tt.args.endInSeconds); (err != nil) != tt.wantErr {
				t.Errorf("MP3Container.AddSection() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(tt.c.streams) != tt.streamsCount {
				t.Errorf("MP3Container.AddSection() expected %v cached streams, found %v", tt.streamsCount, len(tt.c.streams))
			}
		})
	}
}

func TestMP3Container_Persist(t *testing.T) {
	testfilepath := filepath.Join(getMP3TestFolder(t), testFileName)
	totalFileSize := getFileSizeInBytes(t, testfilepath)
	totalFileLength, err := GetLengthInSeconds(testfilepath)
	if err != nil {
		t.Errorf("cloud not get file length %v", err)
	}

	type args struct {
		path string
	}
	tests := []struct {
		name                     string
		c                        *MP3Container
		args                     args
		expectedLengthInSeconds  float64
		expectedNumberOfChapters int
		wantErr                  bool
	}{
		{
			name: "sections test",
			c: createContainer(t, []SecondsWindow{{
				start: 1,
				end:   2,
			}, {
				start: 1,
				end:   5,
			}}),
			args: args{
				path: generateMP3FileName(t),
			},
			expectedLengthInSeconds:  5,
			expectedNumberOfChapters: 1,
			wantErr:                  false,
		}, {
			name: "sub second test",
			c: createContainer(t, []SecondsWindow{{
				start: 0,
				end:   1.5,
			}}),
			args: args{
				path: generateMP3FileName(t),
			},
			expectedLengthInSeconds:  1.5,
			expectedNumberOfChapters: 1,
			wantErr:                  false,
		}, {
			name: "complete file test",
			c: createContainer(t, []SecondsWindow{{
				start: 0,
				end:   -1,
			}}),
			args: args{
				path: generateMP3FileName(t),
			},
			expectedLengthInSeconds:  1059.89,
			expectedNumberOfChapters: 4,
			wantErr:                  false,
		}, {
			name: "file not available",
			c:    createContainer(t, make([]SecondsWindow, 0)),
			args: args{
				path: "dummy 1",
			},
			expectedLengthInSeconds:  -1,
			expectedNumberOfChapters: -1,
			wantErr:                  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.c.Persist(tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("MP3Container.Persist() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				actualLength, err := GetLengthInSeconds(tt.args.path)
				if err != nil {
					t.Errorf("MP3Container.Persist() found error while calculating length = %v", err)
				}
				// fuzzy test if expected length is out by 0.1 or more
				if math.Abs(actualLength-tt.expectedLengthInSeconds) > 0.1 {
					t.Errorf("MP3Container.Persist() expected length = %v, actual length = %v", tt.expectedLengthInSeconds, actualLength)
				}

				fileSize := getFileSizeInBytes(t, tt.args.path)
				expectedSize := (float64(totalFileSize) / totalFileLength) * tt.expectedLengthInSeconds
				// test that resulting file is not less the 95% from expected file size
				if (float64(fileSize) / expectedSize) < 0.95 {
					t.Errorf("MP3Container.Persist() file did not have approximated size, expected %v, actual %v", expectedSize, fileSize)
				}
				chapters, err := GetChapterMetadata(tt.args.path)
				if err != nil {
					t.Errorf("MP3Container.Persist() could not read chapters = %v", err)
				}
				if len(chapters) != tt.expectedNumberOfChapters {
					t.Errorf("MP3Container.Persist() expected number of chapters = %v, actual %v", tt.expectedNumberOfChapters, len(chapters))
				}
			}
		})
	}
}

func getFileSizeInBytes(t *testing.T, filePath string) int64 {
	file, err := os.Open(filePath)
	if err != nil {
		t.Errorf("could not open file path %s, %v", filePath, err)
		return -1
	}

	fileInfo, err := file.Stat()
	if err != nil {
		t.Errorf("could not get file details %s, %v", filePath, err)
		return -1
	}

	defer file.Close()

	return fileInfo.Size()
}

func getMP3TestFolder(t *testing.T) string {
	// get test folder
	testFileFolder, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	return filepath.Join(testFileFolder, "test", "mp3")
}

func generateMP3FileName(t *testing.T) string {
	filePath := filepath.Join(os.TempDir(), strconv.Itoa(random.Intn(9999999999999))+".mp3")
	t.Cleanup(func() {
		if _, err := os.Stat(filePath); err == nil {
			err := os.Remove(filePath)
			if err != nil {
				t.Errorf("could not delete file %s, %v", filePath, err)
			}
		}
	})
	return filePath
}

func createContainer(t *testing.T, windows []SecondsWindow) *MP3Container {
	container := NewMP3()

	for _, window := range windows {
		err := container.AddSection(filepath.Join(getMP3TestFolder(t), testFileName), window.start, window.end)
		if err != nil {
			t.Errorf("could not add section %v", err)
		}
	}

	return container
}

package mp3joiner

import (
	"fmt"
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

func TestMP3Builder_Append(t *testing.T) {
	type args struct {
		mp3Filepath    string
		startInSeconds float64
		endInSeconds   float64
	}
	tests := []struct {
		name                string
		c                   *MP3Builder
		args                args
		wantErr             bool
		streamsCount        int
		approximateFileSize int
	}{
		{
			name: "positive test",
			c:    NewMP3Builder(),
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
			c:    NewMP3Builder(),
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
			c:    NewMP3Builder(),
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
			if err := tt.c.Append(tt.args.mp3Filepath, tt.args.startInSeconds, tt.args.endInSeconds); (err != nil) != tt.wantErr {
				t.Errorf("MP3Builder.Append() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(tt.c.streams) != tt.streamsCount {
				t.Errorf("MP3Builder.Append() expected %v cached streams, found %v", tt.streamsCount, len(tt.c.streams))
			}
		})
	}
}

func TestMP3Builder_Build(t *testing.T) {
	testFilePath := filepath.Join(getMP3TestFolder(t), testFileName)
	totalFileSize := getFileSizeInBytes(t, testFilePath)
	totalFileLength, err := GetLengthInSeconds(testFilePath)
	if err != nil {
		t.Errorf("cloud not get file length %v", err)
	}

	type args struct {
		path string
	}
	tests := []struct {
		name                     string
		c                        *MP3Builder
		args                     args
		expectedLengthInSeconds  float64
		expectedNumberOfChapters int
		wantErr                  bool
	}{
		{
			name: "multiple sections with different file test",
			c: createContainerWithDifferentFiles(t, []SecondsWindow{{
				start: 1,
				end:   2,
			}, {
				start: 1,
				end:   5,
			}, {
				start: 7,
				end:   8,
			}}),
			args: args{
				path: generateMP3FileName(t),
			},
			expectedLengthInSeconds:  6,
			expectedNumberOfChapters: 1,
			wantErr:                  false,
		}, {
			name: "multiple sections with same file test",
			c: createContainerWithSameFile(t, []SecondsWindow{{
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
			name: "repeat same file and same section",
			c: createContainerWithSameFile(t, []SecondsWindow{{
				start: 0,
				end:   2,
			}, {
				start: 0,
				end:   2,
			}}),
			args: args{
				path: generateMP3FileName(t),
			},
			expectedLengthInSeconds:  4,
			expectedNumberOfChapters: 1,
			wantErr:                  false,
		}, {
			name: "sub second test",
			c: createContainerWithSameFile(t, []SecondsWindow{{
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
			c: createContainerWithSameFile(t, []SecondsWindow{{
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
			c:    createContainerWithSameFile(t, make([]SecondsWindow, 0)),
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
			if err := tt.c.Build(tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("MP3Builder.Build() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				actualLength, err := GetLengthInSeconds(tt.args.path)
				if err != nil {
					t.Errorf("MP3Builder.Build() found error while calculating length = %v", err)
				}
				// fuzzy test if expected length is out by 0.1 or more
				if math.Abs(actualLength-tt.expectedLengthInSeconds) > 0.1 {
					t.Errorf("MP3Builder.Build() expected length = %v, actual length = %v", tt.expectedLengthInSeconds, actualLength)
				}

				fileSize := getFileSizeInBytes(t, tt.args.path)
				expectedSize := (float64(totalFileSize) / totalFileLength) * tt.expectedLengthInSeconds
				// test that resulting file is not less the 95% from expected file size
				if (float64(fileSize) / expectedSize) < 0.95 {
					t.Errorf("MP3Builder.Build() file did not have approximated size, expected %v, actual %v", expectedSize, fileSize)
				}
				chapters, err := GetChapterMetadata(tt.args.path)
				if err != nil {
					t.Errorf("MP3Builder.Build() could not read chapters = %v", err)
				}
				if len(chapters) != tt.expectedNumberOfChapters {
					t.Errorf("MP3Builder.Build() expected number of chapters = %v, actual %v", tt.expectedNumberOfChapters, len(chapters))
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
	storeFileForCleanUp(t, filePath)
	return filePath
}

func createContainerWithSameFile(t *testing.T, windows []SecondsWindow) *MP3Builder {
	builder := NewMP3Builder()

	for _, window := range windows {
		err := builder.Append(filepath.Join(getMP3TestFolder(t), testFileName), window.start, window.end)
		if err != nil {
			t.Errorf("could not add section %v", err)
		}
	}

	return builder
}

func createContainerWithDifferentFiles(t *testing.T, windows []SecondsWindow) *MP3Builder {
	builder := NewMP3Builder()
	filenames := make([]string, 0)
	for range windows {
		filename := fmt.Sprintf("%s.%s", filepath.Join(os.TempDir(), strconv.Itoa(random.Intn(9999999999999))), "mp3")
		err := copy(filepath.Join(getMP3TestFolder(t), testFileName), filename)
		if err != nil {
			t.Errorf("could copy file %v", err)
		}
		storeFileForCleanUp(t, filename)
		filenames = append(filenames, filename)
	}

	for i, window := range windows {
		err := builder.Append(filenames[i], window.start, window.end)
		if err != nil {
			t.Errorf("could not add section %v", err)
		}
	}

	return builder
}

func storeFileForCleanUp(t *testing.T, filename string) {
	t.Cleanup(func() {
		if _, err := os.Stat(filename); err == nil {
			err := os.Remove(filename)
			if err != nil {
				t.Errorf("could not delete file %s, %v", filename, err)
			}
		}
	})
}

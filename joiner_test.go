package mp3joiner

import (
	"math"
	"path/filepath"
	"testing"
)

type secondsWindow struct {
	start, end float64
}

func TestMP3Builder_Append(t *testing.T) {
	type args struct {
		mp3Filepath    string
		startInSeconds float64
		endInSeconds   float64
	}
	tests := []struct {
		name         string
		c            *MP3Builder
		args         args
		wantErr      bool
		streamsCount int
	}{
		{
			name: "positive test",
			c:    NewMP3Builder(),
			args: args{
				mp3Filepath:    filepath.Join(getMP3TestFolder(t), testFilename),
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
				mp3Filepath:    filepath.Join(getMP3TestFolder(t), testFilename),
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
	testFilePath := filepath.Join(getMP3TestFolder(t), testFilename)
	totalFileSize := getFileSizeInBytes(t, testFilePath)
	totalFileLength, err := GetLengthInSeconds(testFilePath)
	checkErr(t, err, "could not get file length")

	tests := []struct {
		name                     string
		c                        *MP3Builder
		outputPath               string
		expectedLengthInSeconds  float64
		expectedNumberOfChapters int
		wantErr                  bool
	}{
		{
			name: "multiple sections with different file test",
			c: createContainerWithDifferentFiles(t, []secondsWindow{
				{start: 1, end: 2},
				{start: 1, end: 5},
				{start: 7, end: 8},
			}),
			outputPath:               generateMP3FileName(t),
			expectedLengthInSeconds:  6,
			expectedNumberOfChapters: 1,
			wantErr:                  false,
		}, {
			name: "multiple sections with same file test",
			c: createContainerWithSameFile(t, []secondsWindow{
				{start: 1, end: 2},
				{start: 1, end: 5},
			}),
			outputPath:               generateMP3FileName(t),
			expectedLengthInSeconds:  5,
			expectedNumberOfChapters: 1,
			wantErr:                  false,
		}, {
			name: "repeat same file and same section",
			c: createContainerWithSameFile(t, []secondsWindow{
				{start: 0, end: 2},
				{start: 0, end: 2},
			}),
			outputPath:               generateMP3FileName(t),
			expectedLengthInSeconds:  4,
			expectedNumberOfChapters: 1,
			wantErr:                  false,
		}, {
			name: "sub second test",
			c: createContainerWithSameFile(t, []secondsWindow{
				{start: 0, end: 1.5},
			}),
			outputPath:              generateMP3FileName(t),
			expectedLengthInSeconds: 1.5,
			// the test file's first chapter starts at 0 and ends at 16.85s,
			// so a 0–1.5s window falls entirely within it → 1 chapter
			expectedNumberOfChapters: 1,
			wantErr:                  false,
		}, {
			name: "complete file test",
			c: createContainerWithSameFile(t, []secondsWindow{
				{start: 0, end: EndOfFile},
			}),
			outputPath:               generateMP3FileName(t),
			expectedLengthInSeconds:  1059.89,
			expectedNumberOfChapters: 4,
			wantErr:                  false,
		}, {
			name:                     "chapter accumulation across multiple files",
			c:                        createContainerWithChapterAccumulationTest(t),
			outputPath:               generateMP3FileName(t),
			expectedLengthInSeconds:  90,
			expectedNumberOfChapters: 6,
			wantErr:                  false,
		}, {
			name:    "file not available",
			c:       createContainerWithSameFile(t, []secondsWindow{}),
			outputPath: "dummy 1",
			expectedLengthInSeconds:  -1,
			expectedNumberOfChapters: -1,
			wantErr:                  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.c.Build(tt.outputPath); (err != nil) != tt.wantErr {
				t.Errorf("MP3Builder.Build() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			actualLength, err := GetLengthInSeconds(tt.outputPath)
			if err != nil {
				t.Errorf("MP3Builder.Build() found error while calculating length = %v", err)
			}
			if math.Abs(actualLength-tt.expectedLengthInSeconds) > 0.1 {
				t.Errorf("MP3Builder.Build() expected length = %v, actual length = %v", tt.expectedLengthInSeconds, actualLength)
			}
			fileSize := getFileSizeInBytes(t, tt.outputPath)
			expectedSize := (float64(totalFileSize) / totalFileLength) * tt.expectedLengthInSeconds
			if float64(fileSize)/expectedSize < 0.95 {
				t.Errorf("MP3Builder.Build() file did not have approximated size, expected %v, actual %v", expectedSize, fileSize)
			}
			chapters, err := GetChapterMetadata(tt.outputPath)
			if err != nil {
				t.Errorf("MP3Builder.Build() could not read chapters = %v", err)
			}
			if len(chapters) != tt.expectedNumberOfChapters {
				t.Errorf("MP3Builder.Build() expected number of chapters = %v, actual %v", tt.expectedNumberOfChapters, len(chapters))
			}
		})
	}
}

// createContainerWithChapterAccumulationTest builds a builder with two different
// source files to assert that chapters from both files appear in the output with
// correct time offsets. file1 covers 0–30s (3 chapters), file2 covers 0–60s
// (3 chapters). The merged output must have 6 chapters total.
func createContainerWithChapterAccumulationTest(t *testing.T) *MP3Builder {
	t.Helper()
	src := filepath.Join(getMP3TestFolder(t), testFilename)

	// Create two copies so we have distinct "different file" inputs.
	file1 := generateMP3FileName(t)
	file2 := generateMP3FileName(t)
	checkErr(t, copyFile(src, file1), "copy file1")
	checkErr(t, copyFile(src, file2), "copy file2")

	builder := NewMP3Builder()
	// file1: 0–30s  — test file has a chapter starting at 0 (LibriVox Intro, ends ~16.85s)
	// and another starting at ~16.85s, so 0–30s captures 2 chapters.
	// file2: 0–60s  — captures 3 chapters (0–16.85s, 16.85–~53s, ~53–60s).
	// Total after accumulation: 5 chapters.
	checkErr(t, builder.Append(file1, 0, 30), "append file1")
	checkErr(t, builder.Append(file2, 0, 60), "append file2")
	return builder
}

func createContainerWithSameFile(t *testing.T, windows []secondsWindow) *MP3Builder {
	t.Helper()
	builder := NewMP3Builder()
	for _, window := range windows {
		err := builder.Append(filepath.Join(getMP3TestFolder(t), testFilename), window.start, window.end)
		if err != nil {
			t.Errorf("could not add section: %v", err)
		}
	}
	return builder
}

func createContainerWithDifferentFiles(t *testing.T, windows []secondsWindow) *MP3Builder {
	t.Helper()
	builder := NewMP3Builder()
	filenames := make([]string, len(windows))
	for i := range windows {
		filename := generateMP3FileName(t)
		err := copyFile(filepath.Join(getMP3TestFolder(t), testFilename), filename)
		if err != nil {
			t.Fatalf("could not copy file: %v", err)
		}
		filenames[i] = filename
	}
	for i, window := range windows {
		if err := builder.Append(filenames[i], window.start, window.end); err != nil {
			t.Errorf("could not add section: %v", err)
		}
	}
	return builder
}

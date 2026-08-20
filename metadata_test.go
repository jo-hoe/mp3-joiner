package mp3joiner

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGetChapterMetadata(t *testing.T) {
	tests := []struct {
		name            string
		mp3Filepath     string
		firstItem       Chapter
		numberOfResults int
		wantErr         bool
	}{
		{
			name:            "positive test",
			mp3Filepath:     filepath.Join(getMP3TestFolder(t), testFilename),
			numberOfResults: 4,
			firstItem: Chapter{
				TimeBase: "1/1000",
				Start:    0,
				End:      16850,
				Tags:     Tags{Title: "LibriVox Introduction"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := GetChapterMetadata(tt.mp3Filepath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetChapterMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(gotResult) != tt.numberOfResults {
				t.Errorf("GetChapterMetadata() found %v elements, want %v elements", len(gotResult), tt.numberOfResults)
			}
			if tt.numberOfResults > 0 {
				if !reflect.DeepEqual(gotResult[0], tt.firstItem) {
					t.Errorf("GetChapterMetadata() = %v, want %v", gotResult[0], tt.firstItem)
				}
			}
		})
	}
}

func TestGetFFmpegMetadataTag(t *testing.T) {
	tests := []struct {
		name        string
		mp3Filepath string
		wantResult  map[string]string
		wantErr     bool
	}{
		{
			name:        "positive test",
			mp3Filepath: filepath.Join(getMP3TestFolder(t), testFilename),
			wantResult: map[string]string{
				"ID3v1 Comment": "Read by John Doyle",
				"album":         "Librivox Short Ghost and Horror Story Collection Vol. 009",
				"genre":         "Speech",
				"title":         "The Tell-Tale Heart",
				"artist":        "Edgar Allen Poe",
				"track":         "13/16",
				"TLEN":          "1060",
				"encoder":       "Lavf61.1.100",
			},
			wantErr: false,
		}, {
			name:        "non existing file",
			mp3Filepath: "",
			wantResult:  nil,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := GetFFmpegMetadataTag(tt.mp3Filepath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFFmpegMetadataTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("GetFFmpegMetadataTag() = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}

func TestSetFFmpegMetadataTag(t *testing.T) {
	testFilePath := generateMP3FileName(t)
	checkErr(t, copyFile(filepath.Join(getMP3TestFolder(t), testFilename), testFilePath), "could not create temp file")

	chapterMetaData, err := GetChapterMetadata(testFilePath)
	checkErr(t, err, "could not read chapter metadata")
	metaData, err := GetFFmpegMetadataTag(testFilePath)
	checkErr(t, err, "could not read metadata tags")

	tests := []struct {
		name        string
		mp3Filepath string
		tags        map[string]string
		chapters    []Chapter
		wantErr     bool
	}{
		{
			name:        "set same metadata again",
			mp3Filepath: testFilePath,
			tags:        metaData,
			chapters:    chapterMetaData,
			wantErr:     false,
		}, {
			name:        "set new different on non-existing file",
			mp3Filepath: generateMP3FileName(t),
			tags:        map[string]string{"title": "test"},
			chapters: []Chapter{{
				TimeBase: "1/1",
				Start:    1,
				End:      2,
				Tags:     Tags{Title: "testtitle"},
			}},
			wantErr: true,
		}, {
			name:        "non existing file",
			mp3Filepath: "non existing",
			tags:        metaData,
			chapters:    chapterMetaData,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetFFmpegMetadataTag(tt.mp3Filepath, tt.tags, tt.chapters); (err != nil) != tt.wantErr {
				t.Errorf("SetFFmpegMetadataTag() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				newMetaData, err := GetFFmpegMetadataTag(testFilePath)
				if err != nil {
					t.Errorf("could not read metadata: %v", err)
				}
				if !isMetaDataSimilar(t, newMetaData, tt.tags) {
					t.Errorf("not equal metadata = %v, want %v", newMetaData, metaData)
				}
				newChapterData, err := GetChapterMetadata(testFilePath)
				if err != nil {
					t.Errorf("could not read chapter data: %v", err)
				}
				if !isChapterDataSimilar(t, newChapterData, tt.chapters) {
					t.Errorf("not equal chapters = %v, want %v", newChapterData, chapterMetaData)
				}
			}
		})
	}
}

func TestGetLengthInSeconds(t *testing.T) {
	tests := []struct {
		name        string
		mp3Filepath string
		want        float64
		wantErr     bool
	}{
		{
			name:        "positive test",
			mp3Filepath: filepath.Join(getMP3TestFolder(t), testFilename),
			want:        1059.89,
			wantErr:     false,
		}, {
			name:        "non existing file",
			mp3Filepath: "nofile",
			want:        -1,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetLengthInSeconds(tt.mp3Filepath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLengthInSeconds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if math.Abs(tt.want-got) > 0.01 {
				t.Errorf("GetLengthInSeconds() more than 0.01 apart - found %v, expected %v", got, tt.want)
			}
		})
	}
}

func TestGetBitrate(t *testing.T) {
	tests := []struct {
		name        string
		mp3Filepath string
		wantResult  int
		wantErr     bool
	}{
		{
			name:        "positive test",
			mp3Filepath: filepath.Join(getMP3TestFolder(t), testFilename),
			wantResult:  32000,
			wantErr:     false,
		}, {
			name:        "non existing file",
			mp3Filepath: "",
			wantResult:  0,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := GetBitrate(tt.mp3Filepath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBitrate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotResult != tt.wantResult {
				t.Errorf("GetBitrate() = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}

func Test_createTempMetadataFile(t *testing.T) {
	tests := []struct {
		name            string
		tags            map[string]string
		chapters        []Chapter
		wantFileContent string
		wantErr         bool
	}{
		{
			name:            "empty test",
			tags:            map[string]string{},
			chapters:        []Chapter{},
			wantFileContent: ";FFMETADATA1",
			wantErr:         false,
		}, {
			name:     "positive test",
			tags:     map[string]string{"title": "my title"},
			chapters: []Chapter{{TimeBase: "1/1", Start: 12, End: 13, Tags: Tags{Title: "my chapter"}}},
			wantFileContent: ";FFMETADATA1\n" +
				"title=my title\n" +
				"[CHAPTER]\n" +
				"TIMEBASE=1/1\n" +
				"START=12\n" +
				"END=13\n" +
				"title=my chapter",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, err := createTempMetadataFile(tt.tags, tt.chapters)
			if (err != nil) != tt.wantErr {
				t.Errorf("createTempMetadataFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			buffer, err := os.ReadFile(gotPath)
			if err != nil {
				t.Errorf("could not read file: %v", err)
			}
			if string(buffer) != tt.wantFileContent {
				t.Errorf("createTempMetadataFile() expected:\n%q\nactual:\n%q", tt.wantFileContent, string(buffer))
			}
			if err := os.Remove(gotPath); err != nil {
				t.Errorf("could not remove file %v: %v", gotPath, err)
			}
		})
	}
}

func Test_parseMP3Length(t *testing.T) {
	tests := []struct {
		name        string
		ffmpegStats string
		want        float64
		wantErr     bool
	}{
		{
			name:        "positive test",
			ffmpegStats: "size=N/A time=02:02:02.02 bitrate=N/A speed=2.05e+03x",
			want:        7322.02,
			wantErr:     false,
		}, {
			name:        "not parsable size",
			ffmpegStats: "size=N/A time=xx:02:02.02 bitrate=N/A speed=2.05e+03x",
			want:        -1,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMP3Length(tt.ffmpegStats)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMP3Length() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseMP3Length() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sanitizeMetadata(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantOutput string
	}{
		{
			name:       "Escape",
			input:      "= ; # \\",
			wantOutput: "\\= \\; \\# \\\\",
		}, {
			name:       "Leave alone already escaped characters",
			input:      "\\= \\; \\# \\\\",
			wantOutput: "\\= \\; \\# \\\\",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotOutput := sanitizeMetadata(tt.input); gotOutput != tt.wantOutput {
				t.Errorf("sanitizeMetadata() = %v, want %v", gotOutput, tt.wantOutput)
			}
		})
	}
}

func isMetaDataSimilar(t *testing.T, leftMetadata, rightMetadata map[string]string) bool {
	t.Helper()
	if len(leftMetadata) != len(rightMetadata) {
		t.Errorf("metadata length mismatch: got %d, want %d", len(leftMetadata), len(rightMetadata))
		return false
	}
	for key := range leftMetadata {
		// skip encoder: different ffmpeg versions produce different values
		if key == "encoder" {
			continue
		}
		if rightMetadata[key] != leftMetadata[key] {
			return false
		}
	}
	return true
}

func Test_overwriteFile(t *testing.T) {
	tests := []struct {
		name           string
		inputFilePath  string
		outputFilePath string
		wantErr        bool
	}{
		{
			name:           "positive test",
			inputFilePath:  setupTestFile(t),
			outputFilePath: setupTestFile(t),
			wantErr:        false,
		}, {
			name:           "input file does not exist",
			inputFilePath:  "",
			outputFilePath: setupTestFile(t),
			wantErr:        true,
		}, {
			name:           "output file does not exist",
			inputFilePath:  setupTestFile(t),
			outputFilePath: "",
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := overwriteFile(tt.inputFilePath, tt.outputFilePath); (err != nil) != tt.wantErr {
				t.Errorf("overwriteFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// isChapterDataSimilar compares chapters allowing for minor timestamp drift from re-encoding.
func isChapterDataSimilar(t *testing.T, leftChapters, rightChapters []Chapter) bool {
	t.Helper()
	if len(leftChapters) != len(rightChapters) {
		t.Errorf("chapter count mismatch: got %d, want %d", len(leftChapters), len(rightChapters))
		return false
	}
	for i := range rightChapters {
		l, r := leftChapters[i], rightChapters[i]
		if l.Tags.Title != r.Tags.Title || l.TimeBase != r.TimeBase {
			return false
		}
		if math.Abs(float64(l.End-r.End)) > 50 || math.Abs(float64(l.Start-r.Start)) > 50 {
			return false
		}
	}
	return true
}

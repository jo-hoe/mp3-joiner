package mp3joiner

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var (
	TEST_FILENAME = "edgar-allen-poe-the-telltale-heart-original.mp3"
)

func Test_createTempMetadataFile(t *testing.T) {
	type args struct {
		metadata map[string]string
		chapters []Chapter
	}
	tests := []struct {
		name            string
		args            args
		wantFileContent string
		wantErr         bool
	}{
		{
			name:            "empty test",
			args:            args{metadata: map[string]string{}, chapters: []Chapter{}},
			wantFileContent: ";FFMETADATA1",
			wantErr:         false,
		}, {
			name: "positive test",
			args: args{metadata: map[string]string{"title": "my title"}, chapters: []Chapter{
				{TimeBase: "1/1", Start: 12, End: 13, Tags: Tags{Title: "my chapter"}},
			}},
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
			gotMetadataFilepath, err := createTempMetadataFile(tt.args.metadata, tt.args.chapters)
			if (err != nil) != tt.wantErr {
				t.Errorf("createTempMetadataFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			buffer, err := os.ReadFile(gotMetadataFilepath)
			if err != nil {
				t.Errorf("could not read file %v", err)
			}

			result := string(buffer)
			if result != tt.wantFileContent {
				t.Errorf("createTempMetadataFile() expected content:\n'%v', actual content:\n'%v'", tt.wantFileContent, result)
			}
			os.Remove(gotMetadataFilepath)
		})
	}
}

func TestGetChapterMetadata(t *testing.T) {
	type args struct {
		mp3Filepath string
	}
	tests := []struct {
		name            string
		args            args
		firstItem       Chapter
		numberOfResults int
		wantErr         bool
	}{
		{
			name:            "positive test",
			args:            args{mp3Filepath: filepath.Join(getMP3TestFolder(t), TEST_FILENAME)},
			numberOfResults: 4,
			firstItem: Chapter{
				TimeBase: "1/1000",
				Start:    0,
				End:      16900,
				Tags: Tags{
					Title: "LibriVox Introduction",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := GetChapterMetadata(tt.args.mp3Filepath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetChapterMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.numberOfResults != len(gotResult) {
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

func TestGetMetadata(t *testing.T) {
	type args struct {
		mp3Filepath string
	}
	tests := []struct {
		name       string
		args       args
		wantResult map[string]string
		wantErr    bool
	}{
		{
			name: "positive test",
			args: args{
				mp3Filepath: filepath.Join(getMP3TestFolder(t), TEST_FILENAME),
			},
			wantResult: map[string]string{
				"ID3v1 Comment": "Read by John Doyle",
				"album":         "Librivox Short Ghost and Horror Story Collection Vol. 009",
				"genre":         "Speech",
				"title":         "The Tell-Tale Heart",
				"artist":        "Edgar Allen Poe",
				"track":         "13/16",
				"TLEN":          "1060",
				"encoder":       "Lavf58.76.100",
			},
			wantErr: false,
		}, {
			name: "non existing file",
			args: args{
				mp3Filepath: "",
			},
			wantResult: nil,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := GetMetadata(tt.args.mp3Filepath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("GetMetadata() = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}

func Test_GetLengthInSeconds(t *testing.T) {
	type args struct {
		mp3Filepath string
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{
			name:    "positive test",
			args:    args{mp3Filepath: filepath.Join(getMP3TestFolder(t), TEST_FILENAME)},
			want:    1059.89,
			wantErr: false,
		}, {
			name:    "non existing file",
			args:    args{mp3Filepath: "nofile"},
			want:    -1,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetLengthInSeconds(tt.args.mp3Filepath)
			if (err != nil) != tt.wantErr {
				t.Errorf("getLengthInSeconds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if math.Abs(tt.want-got) > 0.01 {
				t.Errorf("getLengthInSeconds() more than 0.01 apart = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseMP3Length(t *testing.T) {
	type args struct {
		ffmpegStats string
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{
			name: "positive test",
			args: args{
				ffmpegStats: "size=N/A time=02:02:02.02 bitrate=N/A speed=2.05e+03x",
			},
			want:    7322.02,
			wantErr: false,
		}, {
			name: "positive test",
			args: args{
				ffmpegStats: "size=N/A time=xx:02:02.02 bitrate=N/A speed=2.05e+03x",
			},
			want:    -1,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMP3Length(tt.args.ffmpegStats)
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
	type args struct {
		input string
	}
	tests := []struct {
		name       string
		args       args
		wantOutput string
	}{
		{
			name: "Escape",
			args: args{
				input: "= ; # \\",
			},
			wantOutput: "\\= \\; \\# \\\\",
		}, {
			name: "Leave alone already escaped characters",
			args: args{
				input: "\\= \\; \\# \\\\",
			},
			wantOutput: "\\= \\; \\# \\\\",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotOutput := sanitizeMetadata(tt.args.input); gotOutput != tt.wantOutput {
				t.Errorf("sanitizeMetadata() = %v, want %v", gotOutput, tt.wantOutput)
			}
		})
	}
}

func TestSetMetadata(t *testing.T) {
	testFilePath := generateMP3FileName(t)
	err := copy(filepath.Join(getMP3TestFolder(t), TEST_FILENAME), testFilePath)
	checkErr(err, "could not create temp file", t)

	chapterMetaData, err := GetChapterMetadata(testFilePath)
	if err != nil {
		t.Errorf("could not create temp file %v", err)
	}
	metaData, err := GetMetadata(testFilePath)
	if err != nil {
		t.Errorf("could not create temp file %v", err)
	}

	type args struct {
		mp3Filepath string
		metadata    map[string]string
		chapters    []Chapter
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "set same metadata again",
			args: args{
				mp3Filepath: testFilePath,
				metadata:    metaData,
				chapters:    chapterMetaData,
			},
			wantErr: false,
		}, {
			name: "set new different",
			args: args{
				mp3Filepath: generateMP3FileName(t),
				metadata:    map[string]string{"title": "test"},
				chapters: []Chapter{{
					cachedMultipicator: 0,
					TimeBase:           "1/1",
					Start:              1,
					End:                2,
					Tags: Tags{
						Title: "testtitle",
					},
				}},
			},
			wantErr: true,
		}, {
			name: "non existing file",
			args: args{
				mp3Filepath: "non existing",
				metadata:    metaData,
				chapters:    chapterMetaData,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetMetadata(tt.args.mp3Filepath, tt.args.metadata, tt.args.chapters); (err != nil) != tt.wantErr {
				t.Errorf("SetMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr == false {
				newMetaData, err := GetMetadata(testFilePath)
				if err != nil {
					t.Errorf("could not read metadata %v", err)
				}
				if !isMetaDataSimilar(t, newMetaData, tt.args.metadata) {
					t.Errorf("not equal metadata = %v, want %v", newMetaData, metaData)
				}

				newChapterData, err := GetChapterMetadata(testFilePath)
				if err != nil {
					t.Errorf("could not read chapter data %v", err)
				}
				if !isChapterDataSimilar(t, newChapterData, tt.args.chapters) {
					t.Errorf("not equal chapters = %v, want %v", newChapterData, chapterMetaData)
				}
			}
		})
	}
}

func isMetaDataSimilar(t *testing.T, leftMetadata, rightMetadata map[string]string) bool {
	leftLength := len(leftMetadata)
	righLength := len(rightMetadata)
	if leftLength != righLength {
		t.Errorf("not equal length of meta data map new value  = %v, want %v", leftMetadata, rightMetadata)
		return false
	}

	for key := range leftMetadata {
		// test file might be encoded with version A of ffmpeg
		// and test may reencode with version B of ffmpeg
		//
		// skip encoder test to make test resiliant against
		// different version of ffmeg
		if key == "encoder" {
			continue
		}

		if rightMetadata[key] != leftMetadata[key] {
			return false
		}
	}

	return true
}

// reencoding results in slightly different lengths
func isChapterDataSimilar(t *testing.T, leftChapters, rightChapters []Chapter) bool {
	leftLength := len(leftChapters)
	righLength := len(rightChapters)
	if leftLength != righLength {
		t.Errorf("not equal length chapters new value  = %v, want %v", leftChapters, rightChapters)
		return false
	}

	for i := 0; i < righLength; i++ {
		left := leftChapters[i]
		right := rightChapters[i]

		if left.Tags.Title != right.Tags.Title || left.TimeBase != right.TimeBase {
			return false
		}

		if math.Abs(float64(left.End-right.End)) > 50 || math.Abs(float64(left.Start-right.Start)) > 50 {
			return false
		}
	}

	return true
}

func Test_overwriteFile(t *testing.T) {
	type args struct {
		inputFilePath  string
		outputFilePath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "positive test",
			args: args{
				inputFilePath:  setupTestFile(t),
				outputFilePath: setupTestFile(t),
			},
			wantErr: false,
		}, {
			name: "input file does not exist",
			args: args{
				inputFilePath:  "",
				outputFilePath: setupTestFile(t),
			},
			wantErr: true,
		}, {
			name: "input file does not exist",
			args: args{
				inputFilePath:  setupTestFile(t),
				outputFilePath: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := overwriteFile(tt.args.inputFilePath, tt.args.outputFilePath); (err != nil) != tt.wantErr {
				t.Errorf("overwriteFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetBitrate(t *testing.T) {
	type args struct {
		mp3Filepath string
	}
	tests := []struct {
		name       string
		args       args
		wantResult int
		wantErr    bool
	}{
		{
			name: "positive test",
			args: args{
				mp3Filepath: filepath.Join(getMP3TestFolder(t), TEST_FILENAME),
			},
			wantResult: 32000,
			wantErr:    false,
		}, {
			name: "non existing file",
			args: args{
				mp3Filepath: "",
			},
			wantResult: -1,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := GetBitrate(tt.args.mp3Filepath)
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

func checkErr(err error, error_prefix string, t *testing.T) {
	if err != nil {
		t.Errorf(fmt.Sprintf("%s '%v'", error_prefix, err))
	}
}

func copy(src string, dst string) error {
	// Read all content of src to data, may cause OOM for a large file.
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	// Write data to dst
	return os.WriteFile(dst, data, 0644)
}

// create file to be moved around
func setupTestFile(t *testing.T) string {
	directory := os.TempDir()
	file, err := os.CreateTemp(directory, "testFile")
	filePath := file.Name()
	if err != nil {
		t.Error("could not create file")
		return ""
	}

	t.Cleanup(func() {
		err := file.Close()
		if err != nil {
			return
		}
		err = os.Remove(filePath)
		if err != nil {
			return
		}
	})

	return filePath
}

package internal

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
			wantFileContent: ";FFMETADATA",
			wantErr:         false,
		}, {
			name: "positive test",
			args: args{metadata: map[string]string{"title": "my title"}, chapters: []Chapter{
				{TimeBase: "1/1", Start: 12, End: 13, Tags: Tags{Title: "my chapter"}},
			}},
			wantFileContent: ";FFMETADATA\n" +
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
				TimeBase:  "1/1000",
				Start:     0,
				StartTime: "0.000000",
				End:       17000,
				EndTime:   "17.000000",
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

func TestGetMP3Metadata(t *testing.T) {
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
			gotResult, err := GetMP3Metadata(tt.args.mp3Filepath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMP3Metadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("GetMP3Metadata() = %v, want %v", gotResult, tt.wantResult)
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
	testFile := filepath.Join(os.TempDir(), TEST_FILENAME)
	err := copy(filepath.Join(getMP3TestFolder(t), TEST_FILENAME), testFile)
	checkErr(err, "could not create temp file", t)

	chapterMetaData, err := GetChapterMetadata(testFile)
	if err != nil {
		t.Errorf("could not create temp file %v", err)
	}
	metaData, err := GetMP3Metadata(testFile)
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
			name: "positive test",
			args: args{
				mp3Filepath: testFile,
				metadata:    metaData,
				chapters:    chapterMetaData,
			},
			wantErr: false,
		},{
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
				newMetaData, err := GetMP3Metadata(testFile)
				if err != nil {
					t.Errorf("could not read metadata %v", err)
				}
				if !reflect.DeepEqual(newMetaData, metaData) {
					t.Errorf("not equal metadata = %v, want %v", newMetaData, metaData)
				}

				newChapterData, err := GetChapterMetadata(testFile)
				if err != nil {
					t.Errorf("could not read chapter data %v", err)
				}
				if !reflect.DeepEqual(newChapterData, chapterMetaData) {
					t.Errorf("not equal chapters = %v, want %v", newChapterData, chapterMetaData)
				}
			}
		})
	}
	err = os.Remove(testFile)
	if err != nil {
		t.Errorf("could not delete file %v", err)
	}
}

func getMP3TestFolder(t *testing.T) string {
	// get test folder
	testFileFolder, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	return filepath.Join(filepath.Dir(testFileFolder), "test", "mp3")
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

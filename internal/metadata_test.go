package internal

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var (
	testFileName = "edgar-allen-poe-the-telltale-heart-original.mp3"
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
			wantFileContent: ";FFMETADATA\n",
			wantErr:         false,
		}, {
			name: "positive test",
			args: args{metadata: map[string]string{"title": "my title"}, chapters: []Chapter{
				{TimeBase: "1/1", Start: 12, End: 13, Tags: Tags{Title: "my chapter"}},
			}},
			wantFileContent: `;FFMETADATA
title=my title
[CHAPTER]
TIMEBASE=1/1
START=12
END=13
title=my chapter
`,
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
			args:            args{mp3Filepath: filepath.Join(getMP3TestFolder(t), testFileName)},
			numberOfResults: 4,
			firstItem: Chapter{
				Id:        2,
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
				mp3Filepath: filepath.Join(getMP3TestFolder(t), testFileName),
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
			args:    args{mp3Filepath: filepath.Join(getMP3TestFolder(t), testFileName)},
			want:    1059.89,
			wantErr: false,
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

func getMP3TestFolder(t *testing.T) string {
	// get test folder
	testFileFolder, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	return filepath.Join(filepath.Dir(testFileFolder), "test", "mp3")
}

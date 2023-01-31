package internal

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

var (
	testFileName = "edgar-allen-poe-the-telltale-heart-original.mp3"
)

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

func getMP3TestFolder(t *testing.T) string {
	// get test folder
	testFileFolder, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	return filepath.Join(filepath.Dir(testFileFolder), "test", "mp3")
}

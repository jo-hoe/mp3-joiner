package mp3joiner

import (
	"reflect"
	"testing"
)

func Test_getChapterInTimeFrame(t *testing.T) {
	type args struct {
		chapters       []Chapter
		startInSeconds float64
		endInSeconds   float64
	}
	tests := []struct {
		name       string
		args       args
		wantResult []Chapter
	}{
		{
			name: "strict subset test",
			args: args{
				chapters: []Chapter{
					{TimeBase: "1/1", Start: 0, End: 10, Tags: Tags{Title: "demo"}},
				},
				startInSeconds: 2,
				endInSeconds:   8,
			},
			wantResult: []Chapter{
				{TimeBase: "1/1", Start: 2, End: 8, Tags: Tags{Title: "demo"}},
			},
		}, {
			name: "outside before",
			args: args{
				chapters: []Chapter{
					{TimeBase: "1/1", Start: 3, End: 4, Tags: Tags{Title: "demo"}},
				},
				startInSeconds: 1,
				endInSeconds:   2,
			},
			wantResult: []Chapter{},
		}, {
			name: "outside after",
			args: args{
				chapters: []Chapter{
					{TimeBase: "1/1", Start: 1, End: 2, Tags: Tags{Title: "demo"}},
				},
				startInSeconds: 3,
				endInSeconds:   4,
			},
			wantResult: []Chapter{},
		}, {
			name: "cross section test",
			args: args{
				chapters: []Chapter{
					{TimeBase: "1/1", Start: 1, End: 5, Tags: Tags{Title: "demo"}},
					{TimeBase: "1/1", Start: 5, End: 10, Tags: Tags{Title: "demo2"}},
				},
				startInSeconds: 3,
				endInSeconds:   7,
			},
			wantResult: []Chapter{
				{TimeBase: "1/1", Start: 3, End: 5, Tags: Tags{Title: "demo"}},
				{TimeBase: "1/1", Start: 5, End: 7, Tags: Tags{Title: "demo2"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotResult := getChapterInTimeFrame(tt.args.chapters, tt.args.startInSeconds, tt.args.endInSeconds); !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("getChapterInTimeFrame() = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}

func Test_mergeChapters(t *testing.T) {
	tests := []struct {
		name       string
		chapters   []Chapter
		wantResult []Chapter
	}{
		{
			name: "merge chapters with same name test",
			chapters: []Chapter{
				{TimeBase: "1/1000", Start: 0, End: 15000, Tags: Tags{Title: "First"}},
				{TimeBase: "1/1000", Start: 15000, End: 20000, Tags: Tags{Title: "First"}},
			},
			wantResult: []Chapter{
				{TimeBase: "1/1000", Start: 0, End: 20000, Tags: Tags{Title: "First"}},
			},
		}, {
			name: "merge chapters with same name test while leaving other alone",
			chapters: []Chapter{
				{TimeBase: "1/1000", Start: 0, End: 15000, Tags: Tags{Title: "First"}},
				{TimeBase: "1/1000", Start: 15000, End: 20000, Tags: Tags{Title: "First"}},
				{TimeBase: "1/1000", Start: 20000, End: 30000, Tags: Tags{Title: "Second"}},
			},
			wantResult: []Chapter{
				{TimeBase: "1/1000", Start: 0, End: 20000, Tags: Tags{Title: "First"}},
				{TimeBase: "1/1000", Start: 20000, End: 30000, Tags: Tags{Title: "Second"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotResult := mergeChapters(tt.chapters); !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("mergeChapters() = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}

func Test_timeBaseMultiplier(t *testing.T) {
	tests := []struct {
		name     string
		timeBase string
		want     int
	}{
		{
			name:     "positive test",
			timeBase: "1/1000",
			want:     1000,
		}, {
			name:     "malformed timebase",
			timeBase: "1-1000",
			want:     defaultTimeBaseInt,
		}, {
			name:     "no timebase set",
			timeBase: "",
			want:     defaultTimeBaseInt,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := timeBaseMultiplier(tt.timeBase); got != tt.want {
				t.Errorf("timeBaseMultiplier(%q) = %v, want %v", tt.timeBase, got, tt.want)
			}
		})
	}
}

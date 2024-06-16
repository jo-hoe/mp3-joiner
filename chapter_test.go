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
					{
						TimeBase: "1/1",
						Start:    0,
						End:      10,
						Tags: Tags{
							Title: "demo",
						},
					},
				},
				startInSeconds: 2,
				endInSeconds:   8,
			},
			wantResult: []Chapter{
				{
					TimeBase: "1/1",
					Start:    2,
					End:      8,
					Tags: Tags{
						Title: "demo",
					},
					cachedMultiplicator: 1},
			},
		}, {
			name: "outside before",
			args: args{
				chapters: []Chapter{
					{
						TimeBase: "1/1",
						Start:    3,
						End:      4,
						Tags: Tags{
							Title: "demo",
						},
					},
				},
				startInSeconds: 1,
				endInSeconds:   2,
			},
			wantResult: []Chapter{},
		}, {
			name: "outside after",
			args: args{
				chapters: []Chapter{
					{
						TimeBase: "1/1",
						Start:    1,
						End:      2,
						Tags: Tags{
							Title: "demo",
						},
					},
				},
				startInSeconds: 3,
				endInSeconds:   4,
			},
			wantResult: []Chapter{},
		}, {
			name: "cross section test",
			args: args{
				chapters: []Chapter{
					{
						TimeBase: "1/1",
						Start:    1,
						End:      5,
						Tags: Tags{
							Title: "demo",
						},
					},
					{
						TimeBase: "1/1",
						Start:    5,
						End:      10,
						Tags: Tags{
							Title: "demo2",
						},
					},
				},
				startInSeconds: 3,
				endInSeconds:   7,
			},
			wantResult: []Chapter{
				{
					TimeBase: "1/1",
					Start:    3,
					End:      5,
					Tags: Tags{
						Title: "demo",
					},
					cachedMultiplicator: 1,
				},
				{
					TimeBase: "1/1",
					Start:    5,
					End:      7,
					Tags: Tags{
						Title: "demo2",
					},
					cachedMultiplicator: 1,
				}},
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
	type args struct {
		chapters []Chapter
	}
	tests := []struct {
		name       string
		args       args
		wantResult []Chapter
	}{
		{
			name: "merge chapters with same name test",
			args: args{
				chapters: []Chapter{
					{
						TimeBase: "1/1000",
						Start:    0,
						End:      15000,
						Tags: Tags{
							Title: "First",
						},
					}, {
						TimeBase: "1/1000",
						Start:    15000,
						End:      20000,
						Tags: Tags{
							Title: "First",
						},
					},
				},
			},
			wantResult: []Chapter{{
				TimeBase: "1/1000",
				Start:    0,
				End:      20000,
				Tags: Tags{
					Title: "First",
				},
				cachedMultiplicator: 1000,
			}},
		}, {
			name: "merge chapters with same name test while leaving other alone",
			args: args{
				chapters: []Chapter{
					{
						TimeBase: "1/1000",
						Start:    0,
						End:      15000,
						Tags: Tags{
							Title: "First",
						},
					}, {
						TimeBase: "1/1000",
						Start:    15000,
						End:      20000,
						Tags: Tags{
							Title: "First",
						},
					}, {
						TimeBase: "1/1000",
						Start:    20000,
						End:      30000,
						Tags: Tags{
							Title: "Second",
						},
					},
				},
			},
			wantResult: []Chapter{{
				TimeBase: "1/1000",
				Start:    0,
				End:      20000,
				Tags: Tags{
					Title: "First",
				},
				cachedMultiplicator: 1000,
			}, {
				TimeBase: "1/1000",
				Start:    20000,
				End:      30000,
				Tags: Tags{
					Title: "Second",
				},
				cachedMultiplicator: 0,
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotResult := mergeChapters(tt.args.chapters); !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("mergeChapters() = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}

func TestChapter_getCachedMultiplicator(t *testing.T) {
	tests := []struct {
		name string
		c    *Chapter
		want int
	}{
		{
			name: "positive test",
			c: &Chapter{
				TimeBase: "1/1000",
			},
			want: 1000,
		}, {
			name: "malformed timebase",
			c: &Chapter{
				TimeBase: "1-1000",
			},
			want: 1000000000,
		}, {
			name: "no timebase set",
			c: &Chapter{
				TimeBase: "",
			},
			want: 1000000000,
		}, {
			name: "cache test",
			c: &Chapter{
				cachedMultiplicator: 5,
			},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.getCachedMultiplicator(); got != tt.want {
				t.Errorf("Chapter.getCachedMultiplicator() = %v, want %v", got, tt.want)
			}
		})
	}
}

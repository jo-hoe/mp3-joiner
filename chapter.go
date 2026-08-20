package mp3joiner

import (
	"regexp"
	"sort"
	"strconv"
)

const defaultTimeBaseInt = 1_000_000_000

var timeBaseRegex = regexp.MustCompile(`1/([0-9]*)`)

// Tags holds the title metadata for a chapter.
type Tags struct {
	Title string `json:"title,omitempty"`
}

// Chapter represents a named time range within an MP3 file.
type Chapter struct {
	TimeBase string
	Start    int
	End      int
	Tags     Tags
}

// ffprobeChapter is the private DTO for deserialising ffprobe chapter JSON.
type ffprobeChapter struct {
	TimeBase string `json:"time_base,omitempty"`
	Start    int    `json:"start,omitempty"`
	End      int    `json:"end,omitempty"`
	Tags     Tags   `json:"tags,omitempty"`
}

func (f ffprobeChapter) toChapter() Chapter {
	return Chapter(f)
}

// ffprobeFormat is the private DTO for deserialising ffprobe format output.
type ffprobeFormat struct {
	Format struct {
		Tags map[string]string `json:"tags,omitempty"`
	} `json:"format,omitempty"`
}

func timeBaseMultiplier(timeBase string) int {
	if len(timeBase) == 0 {
		return defaultTimeBaseInt
	}
	matches := timeBaseRegex.FindStringSubmatch(timeBase)
	if len(matches) != 2 {
		return defaultTimeBaseInt
	}
	multiplier, err := strconv.Atoi(matches[1])
	if err != nil {
		return defaultTimeBaseInt
	}
	return multiplier
}

// GetStartTimeInSeconds returns the chapter start time in seconds.
func (c Chapter) GetStartTimeInSeconds() float64 {
	return float64(c.Start) / float64(timeBaseMultiplier(c.TimeBase))
}

// GetEndTimeInSeconds returns the chapter end time in seconds.
func (c Chapter) GetEndTimeInSeconds() float64 {
	return float64(c.End) / float64(timeBaseMultiplier(c.TimeBase))
}

// SetStartTime sets the chapter start from a value in seconds.
func (c *Chapter) SetStartTime(seconds float64) {
	c.Start = int(seconds * float64(timeBaseMultiplier(c.TimeBase)))
}

// SetEndTime sets the chapter end from a value in seconds.
func (c *Chapter) SetEndTime(seconds float64) {
	c.End = int(seconds * float64(timeBaseMultiplier(c.TimeBase)))
}

func (c *Chapter) shiftBy(seconds float64) {
	m := float64(timeBaseMultiplier(c.TimeBase))
	c.Start += int(seconds * m)
	c.End += int(seconds * m)
}

func sortChaptersByStart(chapters []Chapter) {
	sort.SliceStable(chapters, func(i, j int) bool {
		return chapters[i].Start < chapters[j].Start
	})
}

func getChapterInTimeFrame(chapters []Chapter, startInSeconds float64, endInSeconds float64) []Chapter {
	result := make([]Chapter, 0)
	for _, chapter := range chapters {
		if !isChapterInTimeFrame(chapter, startInSeconds, endInSeconds) {
			continue
		}
		if chapter.GetStartTimeInSeconds() < startInSeconds {
			chapter.SetStartTime(startInSeconds)
		}
		if chapter.GetEndTimeInSeconds() > endInSeconds {
			chapter.SetEndTime(endInSeconds)
		}
		result = append(result, chapter)
	}
	sortChaptersByStart(result)
	return result
}

func isChapterInTimeFrame(chapter Chapter, startInSeconds float64, endInSeconds float64) bool {
	return chapter.GetStartTimeInSeconds() < endInSeconds && chapter.GetEndTimeInSeconds() > startInSeconds
}

func mergeChapters(chapters []Chapter) []Chapter {
	if len(chapters) < 2 {
		return chapters
	}
	sortChaptersByStart(chapters)
	for i := len(chapters) - 1; i >= 1; i-- {
		if chapters[i].Tags.Title == chapters[i-1].Tags.Title {
			chapters[i-1].SetEndTime(chapters[i].GetEndTimeInSeconds())
			chapters = append(chapters[:i], chapters[i+1:]...)
		}
	}
	return chapters
}

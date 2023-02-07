package mp3joiner

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
)

var (
	DEFAULT_TIME_BASE_INT = 1000000000
	DEFAULT_TIME_BASE     = fmt.Sprintf("1/%d", DEFAULT_TIME_BASE_INT)
	TIME_BASE_REGEX       = regexp.MustCompile("1/([0-9]*)")
)

type Chapter struct {
	TimeBase string `json:"time_base,omitempty"`
	Start    int    `json:"start,omitempty"`
	End      int    `json:"end,omitempty"`
	Tags     Tags   `json:"tags,omitempty"`

	cachedMultipicator int
}

type Tags struct {
	Title string `json:"title,omitempty"`
}

type metadata struct {
	Format struct {
		Tags map[string]string `json:"tags,omitempty"`
	} `json:"format,omitempty"`
}

func (c *Chapter) getCachedMultipicator() int {
	if c.cachedMultipicator != 0 {
		return c.cachedMultipicator
	}

	if len(c.TimeBase) == 0 {
		return DEFAULT_TIME_BASE_INT
	}

	matches := TIME_BASE_REGEX.FindStringSubmatch(c.TimeBase)
	if len(matches) != 2 {
		return DEFAULT_TIME_BASE_INT
	}

	multiplicator, err := strconv.Atoi(matches[1])
	if err != nil {
		return DEFAULT_TIME_BASE_INT
	}
	c.cachedMultipicator = multiplicator
	return c.cachedMultipicator
}

func (c *Chapter) GetStartTimeInSeconds() float64 {
	return float64(c.Start) / float64(c.getCachedMultipicator())
}

func (c *Chapter) GetEndTimeInSeconds() float64 {
	return float64(c.End) / float64(c.getCachedMultipicator())
}

func (c *Chapter) SetStartTime(seconds float64) {
	intermediate := int(seconds * float64(c.getCachedMultipicator()))
	c.Start = intermediate
}

func (c *Chapter) SetEndTime(seconds float64) {
	intermediate := int(seconds * float64(c.getCachedMultipicator()))
	c.End = intermediate
}

func getChapterInTimeFrame(chapters []Chapter, startInSeconds float64, endInSeconds float64) (result []Chapter) {
	result = make([]Chapter, 0)

	// add all chapters which are in frame
	for _, chapter := range chapters {
		if isChapterInTimeFrame(chapter, startInSeconds, endInSeconds) {
			if chapter.GetStartTimeInSeconds() < startInSeconds {
				chapter.SetStartTime(startInSeconds)
			}
			if chapter.GetEndTimeInSeconds() > endInSeconds {
				chapter.SetEndTime(endInSeconds)
			}
			result = append(result, chapter)
		}
	}

	// sort by start
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Start < result[j].Start
	})

	return result
}

func isChapterInTimeFrame(chapter Chapter, startInSeconds float64, endInSeconds float64) bool {
	isOutside := chapter.GetEndTimeInSeconds() <= startInSeconds || chapter.GetStartTimeInSeconds() >= endInSeconds
	if isOutside {
		return false
	}
	isInside := startInSeconds <= chapter.GetEndTimeInSeconds() && endInSeconds >= chapter.GetEndTimeInSeconds()
	if isInside {
		return true
	}

	isStartInChapter := startInSeconds >= chapter.GetStartTimeInSeconds()
	isEndInChapter := endInSeconds <= chapter.GetEndTimeInSeconds()

	return isStartInChapter || isEndInChapter
}

func mergeChapters(chapters []Chapter) (result []Chapter) {
	result = chapters
	if len(result) < 2 {
		return result
	}

	// sort by start
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Start < result[j].Start
	})

	for i := len(result) - 1; i >= 0; i-- {
		if i-1 < 0 {
			return
		}

		if result[i].Tags.Title == result[i-1].Tags.Title {
			// reset end of next item
			result[i-1].SetEndTime(float64(result[i].GetEndTimeInSeconds()))

			// remove current item from slice
			result = append(result[:i], result[i+1:]...)
		}
	}

	return result
}

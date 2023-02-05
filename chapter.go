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

type metadata struct {
	Format struct {
		Tags map[string]string `json:"tags,omitempty"`
	} `json:"format,omitempty"`
}

type Tags struct {
	Title string `json:"title,omitempty"`
}

func (c *Chapter) getcachedMultipicator() int {
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

func (c *Chapter) getStartTimeInSeconds() float64 {
	return float64(c.Start) / float64(c.getcachedMultipicator())
}

func (c *Chapter) getEndTimeInSeconds() float64 {
	return float64(c.End) / float64(c.getcachedMultipicator())
}

func (c *Chapter) setStartTime(seconds float64) {
	intermediate := int(seconds * float64(c.getcachedMultipicator()))
	c.Start = intermediate
}

func (c *Chapter) setEndTime(seconds float64) {
	intermediate := int(seconds * float64(c.getcachedMultipicator()))
	c.End = intermediate
}

func getChapterInTimeFrame(chapters []Chapter, startInSeconds float64, endInSeconds float64) (result []Chapter) {
	result = make([]Chapter, 0)

	// add all chapters which are in frame
	for _, chapter := range chapters {
		if isChapterInTimeFrame(chapter, startInSeconds, endInSeconds) {
			if chapter.getStartTimeInSeconds() < startInSeconds {
				chapter.setStartTime(startInSeconds)
			}
			if chapter.getEndTimeInSeconds() > endInSeconds {
				chapter.setEndTime(endInSeconds)
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
	isOutside := chapter.getEndTimeInSeconds() <= startInSeconds || chapter.getStartTimeInSeconds() >= endInSeconds
	if isOutside {
		return false
	}

	isStartInChapter := startInSeconds >= chapter.getStartTimeInSeconds()
	isEndInChapter := endInSeconds <= chapter.getEndTimeInSeconds()

	return isStartInChapter || isEndInChapter
}

func mergeChapters(chapters []Chapter) (result []Chapter) {
	// sort by start
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Start < result[j].Start
	})

	result = make([]Chapter, 0)

	for i := len(chapters) - 1; i >= 0; i-- {
		if i-1 < 0 {
			return
		}

		if chapters[i].Tags.Title == chapters[i-1].Tags.Title {
			// reset end of next item
			chapters[i-1].setEndTime(float64(chapters[i].getEndTimeInSeconds()))

			// remove current item from slice
			result = append(chapters[:i], chapters[i+1:]...)
		}
	}

	return result
}

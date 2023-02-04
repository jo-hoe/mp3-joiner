package mp3joiner

import "sort"

type Chapter struct {
	TimeBase  string `json:"time_base,omitempty"`
	Start     int    `json:"start,omitempty"`
	StartTime string `json:"start_time,omitempty"`
	End       int    `json:"end,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
	Tags      Tags   `json:"tags,omitempty"`
}

type metadata struct {
	Format struct {
		Tags map[string]string `json:"tags,omitempty"`
	} `json:"format,omitempty"`
}

type Tags struct {
	Title string `json:"title,omitempty"`
}

func (c *Chapter) getStartTimeInSeconds() float64 {
	return -1
}

func (c *Chapter) getEndTimeInSeconds() float64 {
	return -1
}

//TODO: implement properly
func GetChapterInTimeFrame(chapters []Chapter, startInSeconds float64, endInSeconds float64) (result []Chapter) {
	result = make([]Chapter, 0)

	for _, chapter := range chapters {
		if chapter.getStartTimeInSeconds() <= startInSeconds {
			continue
		}
		if chapter.getEndTimeInSeconds() <= endInSeconds {
			result = append(result, chapter)
		}
	}

	// sort by start
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Start < result[j].Start
	})

	return result
}

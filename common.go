package mp3joiner

import "strings"

func removeParameters(parameterList []string, parameterKey string) []string {
	var result = make([]string, 0)

	for i := 0; i < len(parameterList); i++ {
		// skip appending key if key = parameter we search for
		if parameterList[i] == parameterKey {
			// skip appanding parameter value in case it follows
			// the key and is not itself a new parameter
			if i+1 < len(parameterList) && !strings.HasPrefix(parameterList[i+1], "-") {
				i = i + 1
			}
		} else {
			result = append(result, parameterList[i])
		}
	}

	return result
}

package mp3joiner

import (
	"log"
	"os"
	"regexp"
	"strings"
)

func removeParameters(parameterList []string, parameterKey string, valueRegex string) []string {
	var result = make([]string, 0)

	for i := 0; i < len(parameterList); i++ {
		// check if this is the key we search for
		if parameterList[i] == parameterKey {
			// check if following value is a parameter value and not a
			// new parameter
			if i+1 < len(parameterList) && !strings.HasPrefix(parameterList[i+1], "-") {
				// test if parameter value complies with regex
				match, err := regexp.MatchString(valueRegex, parameterList[i+1])
				if err == nil && match {
					i = i + 1
					continue
				}
			}
		}
		result = append(result, parameterList[i])
	}

	return result
}

func deleteFile(filePath string) {
	err := os.Remove(filePath)
	if err != nil {
		log.Printf("could not delete temp file %s", err)
	}
}

func closeFile(file *os.File) {
	err := file.Close()
	if err != nil {
		log.Printf("could not close file %s", err)
	}
}

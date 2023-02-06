package mp3joiner

func removeParameters(parameterList []string, parameterKey string, parameterValue string) []string {
	var result = make([]string, 0)
	for i := 0; i < len(parameterList); i++ {
		if i+1 > len(parameterList) {
			result = append(result, parameterList[i])
		}

		if parameterList[i] == parameterKey && parameterList[i+1] == parameterValue {
			i = i + 1
		} else {
			result = append(result, parameterList[i])
		}
	}
	return result
}

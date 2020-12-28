package media

import "regexp"

func FindStringSubmatch(exp *regexp.Regexp, str string) map[string]string {
	match := exp.FindStringSubmatch(str)
	result := make(map[string]string)
	for i, name := range exp.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	return result
}

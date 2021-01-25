package mediaserver

import (
	"fmt"
	"strings"
)

type ParamBuilder map[string][]string

func (pb ParamBuilder) Clear(action string, params []string) (map[string]string, error) {
	var result = make(map[string]string)

	ps, ok := pb[action]
	if !ok {
		return nil, fmt.Errorf("action %s not allowed", action)
	}
	for _, param := range params {
		param = strings.ToLower(param)
		for _, key := range ps {
			if strings.HasPrefix(param, key) {
				result[key] = param[len(key):]
			}
		}
	}
	return result, nil
}

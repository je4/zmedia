package media

import "regexp"

func FindStringSubmatch(exp *regexp.Regexp, str string) map[string]string {
	match := exp.FindStringSubmatch(str)
	result := make(map[string]string)
	if match == nil {
		return result
	}
	for i, name := range exp.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	return result
}

func CalcSize(origWidth, origHeight, Width, Height int64) (width int64, height int64) {
	//    oW    W
	//    -- = --
	//    oH    H
	origAspect := float64(origWidth) / float64(origHeight)
	newAspect := float64(Width) / float64(Height)

	if origAspect < newAspect {
		height = Height
		width = (Height * origWidth) / origHeight
	} else {
		width = Width
		height = (Width * origHeight) / origWidth
	}
	return
}

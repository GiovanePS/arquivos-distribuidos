package utils

import "strings"

func GetFilenameFromPath(filepath string) string {
	idx := strings.LastIndex(filepath, "/")
	if idx == -1 {
		return filepath
	}

	filename := filepath[idx+1:]
	return filename
}

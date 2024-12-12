package utils

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// Returns a boolean to define if the file is from remote connection or not,
// in addition to return the IP Address, the source directory and the destiantion directory.
func GetArgs() (bool, string, string, string) {
	arg1 := os.Args[1] // Source
	arg2 := os.Args[2] // Destination

	// If IP is on source
	idx := strings.LastIndex(arg1, ":")
	if idx != -1 {
		ip := net.ParseIP(arg1[:idx])
		if ip == nil {
			fmt.Println("IP not found in args.")
			os.Exit(1)
		}

		src := arg1[idx+1:]
		dst := arg2
		isRemoteFile := true
		return isRemoteFile, ip.String(), src, dst
	}

	// If IP is on destination
	idx = strings.LastIndex(arg2, ":")
	if idx != -1 {
		ip := net.ParseIP(arg2[:idx])
		if ip == nil {
			fmt.Println("IP not found in args.")
			os.Exit(1)
		}

		src := arg1
		dst := arg2[idx+1:]
		isRemoteFile := false
		return isRemoteFile, ip.String(), src, dst
	}

	fmt.Println("No IP found in args.")
	os.Exit(1)
	return false, "", "", ""
}

func GetFilenameFromPath(filepath string) string {
	idx := strings.LastIndex(filepath, "/")
	if idx == -1 {
		return filepath
	}

	filename := filepath[idx+1:]
	return filename
}

func GenerateBarProgress(percentage float64) string {
	barLength := 50
	if percentage > 100 {
		percentage = 100
	}
	filledLength := int((percentage / 100) * float64(barLength))
	bar := "[" + strings.Repeat("=", filledLength) + strings.Repeat(" ", barLength-filledLength) + "]"
	progressBar := fmt.Sprintf("\r%s %.2f%%", bar, percentage)
	return progressBar
}

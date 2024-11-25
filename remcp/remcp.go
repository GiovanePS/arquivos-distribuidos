package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"remcp/utils"
	"strings"
)

const Port = ":3000"

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: ./remcp arg1 arg2")
		os.Exit(1)
	}

	isRemoteFile, ip, sourcePath, destinationPath := getArgs()

	conn, err := net.Dial("tcp", ip+Port)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if isRemoteFile {
		file, err := os.Open(destinationPath)
		if os.IsNotExist(err) { // Verifying if the directory of destination exists.
			fmt.Println("Directory %s doesn't exist.", destinationPath)
			os.Exit(1)
		}

		receiveFile(conn, sourcePath, destinationPath)
		file.Close()
		return
	}

	file, err := os.Open(sourcePath)
	if err != nil {
		fmt.Println("File doesn't exist.")
		os.Exit(1)
	}

	filename := sourcePath
	idx := strings.LastIndex(filename, "/") // Getting only the name of file
	if idx != -1 {
		filename = filename[idx+1:]
	}

	sendFile(conn, filename, file, destinationPath)
	file.Close()
	return
}

// Returns a boolean to define if the file is from remote connection or not,
// in addition it returns the IP Address, the source directory and the destiantion directory.
func getArgs() (bool, string, string, string) {
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

func sendFile(conn net.Conn, filename string, file *os.File, destinationPath string) error {
	flagSendFile := 1
	binary.Write(conn, binary.LittleEndian, &flagSendFile)
	fullPath := destinationPath+"/"+filename
	io.WriteString(conn, fullPath)

	buf := make([]byte, 128)
	for {
		n, err := file.Read(buf) // Send file data
		if err != nil && err != io.EOF {
			return err
		}

		if n == 0 {
			break
		}

		if _, err := conn.Write(buf[:n]); err != nil {
			return err
		}
	}

	fmt.Printf("File sent to %s\n", conn.RemoteAddr())
	return nil
}

func receiveFile(conn net.Conn, filepath string, destinationPath string) error {
	flagReceiveFile := 0
	binary.Write(conn, binary.LittleEndian, &flagReceiveFile)
	io.WriteString(conn, filepath)

	file, err := os.Create(destinationPath + "/" + utils.GetFilenameFromPath(filepath))
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	for {
		_, err := io.CopyN(buf, conn, 128)
		fmt.Println(buf)
		// TODO: Refatorar
		if err != nil && err == io.EOF {
			if err == io.EOF {
				_, err = file.Write(buf.Bytes())
				break
			} else {
				return err
			}
		}

		_, err = file.Write(buf.Bytes())
		if err != nil {
			return err
		}

		buf.Reset()
	}

	fmt.Println("File successfully received!")
	return nil
}

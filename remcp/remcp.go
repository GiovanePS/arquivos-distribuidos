package main

import (
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
		return
	}

	if isRemoteFile {
		_, err := os.Open(destinationPath)
		if os.IsNotExist(err) { // Verifying if the directory of destination exists.
			fmt.Printf("Directory %s doesn't exist.\n", destinationPath)
			return
		}

		if err = receiveFile(conn, sourcePath, destinationPath); err != nil {
			fmt.Println(err)
		}

		return
	}

	file, err := os.Open(sourcePath)
	if os.IsNotExist(err) { // Verifying if the file of source exists.
		fmt.Printf("Directory %s doesn't exist.\n", destinationPath)
		return
	}

	if err = sendFile(conn, file, destinationPath); err != nil {
		fmt.Println(err)
	}

	return
}

// Returns a boolean to define if the file is from remote connection or not,
// in addition to return the IP Address, the source directory and the destiantion directory.
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

func sendFile(conn net.Conn, file *os.File, destinationPath string) error {
	fmt.Println("Send file")
	var flagSendFile int32 = 1
	binary.Write(conn, binary.LittleEndian, &flagSendFile)
	io.WriteString(conn, destinationPath + utils.GetFilenameFromPath(file.Name()))

	// Acknowledgment to start send file
	ack := make([]byte, 1)
	if _, err := conn.Read(ack); err != nil || ack[0] != 1 {
		return fmt.Errorf("Failed to receive acknowledgment from server.")
	}

	buf := make([]byte, 128)
	for {
		n, err := file.Read(buf)
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
	fmt.Println("Receive file")
	var flagReceiveFile int32 = 0
	binary.Write(conn, binary.LittleEndian, &flagReceiveFile)
	io.WriteString(conn, filepath)

	file, err := os.Create(destinationPath + "/" + utils.GetFilenameFromPath(filepath))
	if err != nil {
		return err
	}

	// Acknowledgment to start receive file
	ack := make([]byte, 1)
	if _, err := conn.Read(ack); err != nil || ack[0] != 1 {
		return fmt.Errorf("Failed to receive acknowledgment from server.")
	}

	buf := make([]byte, 128)
	for {
		n, err := conn.Read(buf)
		if err != nil && err == io.EOF {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}

		_, err = file.Write(buf[:n])
		if err != nil {
			return err
		}
	}

	fmt.Println("File successfully received!")
	return nil
}

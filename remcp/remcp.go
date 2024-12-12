package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"remcp/utils"
)

const (
	Port         = ":3000"
	transferRate = 128
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: ./remcp arg1 arg2")
		os.Exit(1)
	}

	isRemoteFile, ip, sourcePath, destinationPath := utils.GetArgs()

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
		fmt.Printf("Directory %s doesn't exist.\n", sourcePath)
		return
	}

	if err = sendFile(conn, file, destinationPath); err != nil {
		fmt.Println(err)
	}

	return
}

func sendFile(conn net.Conn, file *os.File, destinationPath string) error {
	var flagSendFile int32 = 1
	binary.Write(conn, binary.LittleEndian, &flagSendFile)
	conn.Write([]byte(file.Name() + "/" + destinationPath))

	// Acknowledgment to start send file
	ack := make([]byte, 1)
	if _, err := conn.Read(ack); err != nil || ack[0] != 1 {
		return fmt.Errorf("Failed to receive acknowledgment from server.")
	}

	buf := make([]byte, transferRate)
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
	var flagReceiveFile int32 = 0
	binary.Write(conn, binary.LittleEndian, &flagReceiveFile)

	if _, err := conn.Write([]byte(filepath)); err != nil {
		return fmt.Errorf("Error on write filename to server: %s", err)
	}

	file, err := os.Create(destinationPath + "/" + utils.GetFilenameFromPath(filepath))
	if err != nil {
		return err
	}

	var totalSize int64
	binary.Read(conn, binary.LittleEndian, &totalSize)

	// Acknowledgment to start receive file
	ack := []byte{1}
	if _, err := conn.Write(ack); err != nil || ack[0] != 1 {
		return fmt.Errorf("Failed to receive acknowledgment from server.")
	}

	transfered := 0
	buf := make([]byte, transferRate)
	for {
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}

		percentage := (float64(transfered) / float64(totalSize)) * 100
		progressBar := utils.GenerateBarProgress(percentage)
		fmt.Print(progressBar)
		transfered += n

		if n == 0 {
			break
		}

		_, err = file.Write(buf[:n])
		if err != nil {
			return err
		}
	}

	fmt.Println("\nFile successfully received!")
	return nil
}

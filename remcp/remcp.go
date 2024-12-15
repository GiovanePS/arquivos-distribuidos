package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"remcp/utils"
	"time"
)

const (
	Port         = ":3000"
	transferRate = 100000
	maxAttemps   = 5
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: ./remcp arg1 arg2")
		os.Exit(1)
	}

	isRemoteFile, ip, sourcePath, destinationPath := utils.GetArgs()

	var conn net.Conn
	var err error
	attemp := 0
	for {
		attemp++
		if attemp > maxAttemps {
			fmt.Println("Timeout!")
			return
		}

		conn, err = net.Dial("tcp", ip+Port)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Acknowledgment connection from server
		ack := make([]byte, 1)
		_, err = conn.Read(ack)
		if err != nil {
			fmt.Errorf("Error on read acknowledgment from server. %s", err)
			return
		}

		if ack[0] != 1 {
			if attemp == 1 {
				fmt.Println("Server overloaded.")
			}

			fmt.Println("Retrying connection in 5 seconds...")
		} else {
			break
		}

		time.Sleep(5 * time.Second)
	}

	if isRemoteFile {
		dir, err := os.Open(destinationPath)
		if os.IsNotExist(err) { // Verifying if the directory of destination exists.
			fmt.Printf("Directory %s doesn't exist.\n", destinationPath)
			return
		}
		dir.Close()

		if err = receiveFile(conn, sourcePath, destinationPath); err != nil {
			fmt.Printf("\n%s", err)
		}

		return
	}

	file, err := os.Open(sourcePath)
	if os.IsNotExist(err) { // Verifying if the file of source exists.
		fmt.Printf("File %s doesn't exist.\n", sourcePath)
		return
	}

	if err = sendFile(conn, file, destinationPath); err != nil {
		fmt.Println(err)
	}

	return
}

func sendFile(conn net.Conn, file *os.File, destinationPath string) error {
	flagSendFile := []byte{1}
	conn.Write(flagSendFile)
	conn.Write([]byte(destinationPath + "/" + utils.GetFilenameFromPath(file.Name())))

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

type FileTransfer struct {
	Filepath   string
	Transfered int64
}

// Function to receive a file from daemon server.
//   - conn: TCP Connection;
//   - filepath: directory in server where the file is;
//   - destinationPath: local directory where the file will be saved;
func receiveFile(conn net.Conn, filepath string, destinationPath string) error {
	flagReceiveFile := []byte{0}
	conn.Write(flagReceiveFile)

	file, err := getOrCreateFilePart(filepath, destinationPath)
	if err != nil {
		return err
	}

	n, err := file.Seek(0, io.SeekEnd) // Moving cursor to end of file
	if err != nil {
		return err
	}

	transfered := int(n)

	ft := FileTransfer{Filepath: filepath, Transfered: n}
	var structBuffer bytes.Buffer
	encoder := gob.NewEncoder(&structBuffer)
	if err = encoder.Encode(ft); err != nil {
		return fmt.Errorf("Error on encode struct: %s", err)
	}

	if _, err = conn.Write(structBuffer.Bytes()); err != nil {
		return fmt.Errorf("Error on send FileTransfer struct to server: %s", err)
	}

	// Get total file size to display progress bar
	totalSizeAsBytes := make([]byte, 8)
	conn.Read(totalSizeAsBytes)
	totalSize := binary.LittleEndian.Uint64(totalSizeAsBytes)

	// Acknowledgment to start receive file
	ack := []byte{1}
	if _, err := conn.Write(ack); err != nil || ack[0] != 1 {
		return fmt.Errorf("Failed to receive acknowledgment from server.")
	}

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

	file.Close()
	filename := utils.GetFilenameFromPath(filepath)
	if err := moveFile("/tmp/"+filename+".part", destinationPath+filename); err != nil {
		return err
	}

	fmt.Println("\nFile successfully received!")
	return nil
}

func getOrCreateFilePart(filepath, destinationPath string) (*os.File, error) {
	filename := utils.GetFilenameFromPath(filepath)

	// Open file or create if it doesn't exist
	file, err := os.OpenFile("/tmp/"+filename+".part", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func moveFile(source, destination string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	err = os.Remove(source)
	if err != nil {
		return err
	}

	return nil
}

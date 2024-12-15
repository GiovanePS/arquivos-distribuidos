package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

const (
	transferRate = 100000
	maxClients   = 2
)

var (
	currentClients      = 0
	mutexCurrentClients sync.Mutex
)

func main() {
	StartDaemon()
}

func StartDaemon() {
	ln, err := net.Listen("tcp", ":3000")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}

		mutexCurrentClients.Lock()
		if currentClients >= maxClients {
			ack := []byte{0}
			conn.Write(ack)
			conn.Close()
		} else {
			ack := []byte{1}
			conn.Write(ack)
			currentClients++
			go handleConn(conn) // Handle of requests
		}
		mutexCurrentClients.Unlock()
	}
}

func handleConn(conn net.Conn) {
	defer decrementClients() // Garantir o decremento dos clientes.
	defer conn.Close()
	flag := make([]byte, 1)
	conn.Read(flag)

	// The flag is what the connection starter wants to.
	// If he wants to receive a file, the server will send the file.
	// If he wants to send a file, the server will receive the file.
	switch flag[0] {
	case 0: // flagReceiveFile
		err := sendFile(conn)
		if err != nil {
			fmt.Printf("Error on connection with %s: %s\n", conn.RemoteAddr(), err)
		}
	case 1: // flagSendFile
		err := receiveFile(conn)
		if err != nil {
			fmt.Printf("Error on connection with %s: %s\n", conn.RemoteAddr(), err)
		}
	}
}

type FileTransfer struct {
	Filepath   string
	Transfered int64
}

func sendFile(conn net.Conn) error {
	structBuffer := make([]byte, 1024)
	n, err := conn.Read(structBuffer)
	if err != nil {
		return err
	}

	var ft FileTransfer
	decoder := gob.NewDecoder(bytes.NewReader(structBuffer[:n]))
	if err := decoder.Decode(&ft); err != nil {
		return fmt.Errorf("Error on read FileTransfer struct from client: %s", err)
	}

	file, err := os.Open(ft.Filepath)
	if os.IsNotExist(err) {
		fmt.Printf("Directory requested %s doesn't exist.\n", ft.Filepath)
		return err
	}
	defer file.Close()

	if _, err = file.Seek(int64(ft.Transfered), io.SeekStart); err != nil {
		return err
	}

	fileinfo, err := file.Stat() // Getting total size of file
	if err != nil {
		return fmt.Errorf("Error on get info about file: %s", err)
	}

	size := fileinfo.Size()
	sizeAsBytes := make([]byte, 8) // 8 bits to 64 uint
	binary.LittleEndian.PutUint64(sizeAsBytes, uint64(size))
	conn.Write(sizeAsBytes)

	// Acknowledgment to start receive file
	ack := make([]byte, 1)
	if _, err := conn.Read(ack); err != nil {
		return err
	}

	for {
		n, err := transferFileWithRateLimit(file, conn)
		if err != nil {
			return err
		}

		if n == 0 {
			break
		}

		time.Sleep(1 * time.Second)
	}

	fmt.Printf("File sent to %s\n", conn.RemoteAddr())
	return nil
}

func decrementClients() {
	mutexCurrentClients.Lock()
	currentClients--
	mutexCurrentClients.Unlock()
}

// Function to properly send the files within the rate limit.
func transferFileWithRateLimit(file *os.File, conn net.Conn) (int, error) {
	mutexCurrentClients.Lock()
	currentTransferRate := transferRate / currentClients
	mutexCurrentClients.Unlock()
	buf := make([]byte, currentTransferRate)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return n, err
	}

	if _, err := conn.Write(buf[:n]); err != nil {
		return n, err
	}

	return n, nil
}

func receiveFile(conn net.Conn) error {
	filepathAsBytes := make([]byte, 1024)
	n, err := conn.Read(filepathAsBytes)
	if err != nil {
		return fmt.Errorf("Error receive filepath: %s")
	}

	filepath := string(filepathAsBytes[:n])
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("Error on creating file: %s", err)
	}
	defer file.Close()

	// Acknowledgment to start receive file
	ack := []byte{1}
	if _, err := conn.Write(ack); err != nil {
		return err
	}

	buf := make([]byte, transferRate)
	for {
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}

		if n == 0 {
			break
		}

		_, err = file.Write(buf[:n])
		if err != nil {
			return err
		}
	}

	fmt.Println("File successfully received!")
	return nil
}

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

const (
	transferRate = 128
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
	var flag int32
	binary.Read(conn, binary.LittleEndian, &flag)

	// The flag is what the connection starter wants to.
	// If he wants to receive a file, the server will send the file.
	// If he wants to send a file, the server will receive the file.
	switch flag {
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

func sendFile(conn net.Conn) error {
	filenameAsBytes := make([]byte, 1024)
	n, err := conn.Read(filenameAsBytes)
	if err != nil {
		return fmt.Errorf("Error on read filename from client: %s", err)
	}

	filename := string(filenameAsBytes[:n])
	file, err := os.Open(filename)
	if os.IsNotExist(err) {
		fmt.Printf("Directory requested %s doesn't exist.\n", filename)
		return err
	}
	defer file.Close()

	fileinfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("Error on get info about file: %s", err)
	}

	binary.Write(conn, binary.LittleEndian, fileinfo.Size())

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
		return fmt.Errorf("Error receive transfer metadata: %s")
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

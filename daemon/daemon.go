package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
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

		go handleConn(conn) // Handle of requests
	}
}

func handleConn(conn net.Conn) {
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
	default:
		out := fmt.Sprintln("Only supports send or receive a file.")
		fmt.Println(out)
		io.WriteString(conn, out)
	}
}

func sendFile(conn net.Conn) error {
	filenameAsBytes := make([]byte, 1024)
	n, err := conn.Read(filenameAsBytes)
	if err != nil {
		return err
	}

	// Acknowledgment to start receive file
	ack := []byte{1}
	if _, err := conn.Write(ack); err != nil {
		return err
	}

	filename := string(filenameAsBytes[:n])
	file, err := os.Open(filename)
	if os.IsNotExist(err) {
		fmt.Printf("Directory requested %s doesn't exist.\n", filename)
		return err
	}
	defer file.Close()

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

func receiveFile(conn net.Conn) error {
	filepath := make([]byte, 1024)
	n, err := conn.Read(filepath)
	if err != nil {
		return fmt.Errorf("Error reading filepath: %s")
	}

	file, err := os.Create(string(filepath[:n]))
	if err != nil {
		return fmt.Errorf("Error on creating file: %s", err)
	}

	defer file.Close()

	// Acknowledgment to start receive file
	ack := []byte{1}
	if _, err := conn.Write(ack); err != nil {
		return err
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

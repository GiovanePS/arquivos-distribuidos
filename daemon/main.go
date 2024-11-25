package main

import (
	"bytes"
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
	var flag int
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

	fmt.Println(string(filenameAsBytes))
	return nil

	filename := string(filenameAsBytes[:n])
	file, err := os.Open(filename)
	if os.IsNotExist(err) {
		fmt.Printf("File requested %s doesn't exist.\n", filename)
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
	destinationPathAsBytes := make([]byte, 1024)
	n, err := conn.Read(destinationPathAsBytes)
	if err != nil {
		return err
	}

	destinationPath := string(destinationPathAsBytes[:n])
	file, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer file.Close()

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

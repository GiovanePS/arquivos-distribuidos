package server

import (
	"fmt"
	"log"
	"net"
)

type Server struct{}

func (s *Server) Start() {
	ln, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go s.readFile(conn)
	}
}

func (s *Server) readFile(conn net.Conn) {
	buff := make([]byte, 1024)
	for {
		n, err := conn.Read(buff)
		if err != nil {
			log.Fatal(err)
		}

		file := buff[:n]
		fmt.Println(file)
		fmt.Printf("Received %d bytes!\n", n)
	}
}

package main

import (
	"fmt"
	"io"
)

type groupServer interface {
	serve(io.ReadWriter)
	write([]byte) error
}

type streamServer struct {
	io.ReadWriter
}

func (s *streamServer) serve(rw io.ReadWriter) {
	s.ReadWriter = rw
	buf := make([]byte, 1)
	for {
		_, err := s.Read(buf)
		if err != nil {
			return
		}
	}
}

func (s *streamServer) write(buf []byte) error {
	_, err := s.Write(buf)
	return err
}

type matchServer struct {
	io.ReadWriter
	lastMessage []byte
}

func (s *matchServer) serve(rw io.ReadWriter) {
	s.ReadWriter = rw
	buf := make([]byte, 1)
	for {
		_, err := s.Read(buf)
		if err != nil {
			return
		}
	}
}

func (s *matchServer) write(buf []byte) error {
	fmt.Println("write", s)
	_, err := s.Write(buf)
	return err
}

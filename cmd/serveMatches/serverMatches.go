package main

// listen to the coinbase websocket api and server
// matched orders to tcp clients

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/reds/coinbaseX"
)

type clientGroup struct {
	connections map[net.Conn]net.Conn
	sync.Mutex
	lastMessage string
}

func (cg *clientGroup) addClient(c net.Conn) {
	cg.Lock()
	defer cg.Unlock()
	cg.connections[c] = c
}

func (cg *clientGroup) remClient(c net.Conn) {
	cg.Lock()
	defer cg.Unlock()
	delete(cg.connections, c)
}

func (cg *clientGroup) serve(c net.Conn) {
	cg.addClient(c)
	defer cg.remClient(c)
	_, err := c.Write([]byte(cg.getLastMessage()))
	if err != nil {
		log.Println("error writing starting message to client", err)
		return
	}
	buf := make([]byte, 1)
	for {
		_, err := c.Read(buf)
		if err != nil {
			log.Println("error reading from client", err)
			return
		}
	}
}

func (cg *clientGroup) getConnections() []net.Conn {
	res := make([]net.Conn, 0, len(cg.connections))
	cg.Lock()
	defer cg.Unlock()
	for k, _ := range cg.connections {
		res = append(res, k)
	}
	return res
}

func (cg *clientGroup) setLastMessage(msg string) {
	cg.Lock()
	defer cg.Unlock()
	cg.lastMessage = msg
}

func (cg *clientGroup) getLastMessage() string {
	cg.Lock()
	defer cg.Unlock()
	return cg.lastMessage
}

func main() {
	cb, err := coinbaseX.New(filepath.Join(os.Getenv("HOME"), ".ssh", "coinbase.js"))
	if err != nil {
		panic(err)
	}
	q := make(chan *coinbaseX.StreamMsg, 1)
	err = cb.Stream(q)
	if err != nil {
		panic(err)
	}
	cg := &clientGroup{connections: make(map[net.Conn]net.Conn)}
	go func() {
		for m := range q {
			switch m.Type {
			case coinbaseX.StreamMsgMatch:
				msg := fmt.Sprintln(m.Price, m.Size, m.Side)
				cg.setLastMessage(msg)
				for _, c := range cg.getConnections() {
					_, err = c.Write([]byte(msg))
					if err != nil {
						cg.remClient(c)
					}
				}
			case coinbaseX.StreamMsgInternalError:
				if m.Error == io.EOF {
					log.Println("restarting stream")
					go cb.Stream(q)
				}
			}
		}
	}()
	l, err := net.Listen("tcp", ":5898")
	if err != nil {
		panic(err)
	}
	for {
		c, err := l.Accept()
		if err != nil {
			panic(err)
		}
		go cg.serve(c)
	}
}

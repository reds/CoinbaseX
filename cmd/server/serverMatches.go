package main

// listen to the coinbase websocket api and server
// matched orders to tcp clients

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/reds/coinbaseX"
)

func connect(q chan *coinbaseX.StreamMsg, hp string) {
	conn, err := net.Dial("tcp", hp)
	if err != nil {
		panic(err)
	}
	s := bufio.NewScanner(conn)
	for s.Scan() {
		msg, err := coinbaseX.ParseMsg(s.Bytes())
		if err != nil {
			q <- &coinbaseX.StreamMsg{Type: coinbaseX.StreamMsgInternalError, Message: err.Error(), Error: err}
			return
		}
		q <- msg

	}
}

const serverHP = "localhost:5900"

func main() {
	q := make(chan *coinbaseX.StreamMsg, 1)
	go connect(q, serverHP)
	streamGroup := &clientGroup{connections: make(map[net.Conn]net.Conn), hostPort: ":5899",
		server: &streamServer{},
	}
	matchGroup := &clientGroup{connections: make(map[net.Conn]net.Conn), hostPort: ":5898",
		server: &matchServer{},
	}
	go func() {
		for m := range q {
			streamGroup.write(m.Json)
			streamGroup.write([]byte("\n"))
			switch m.Type {
			case coinbaseX.StreamMsgMatch:
				msg := fmt.Sprintln(m.Price, m.Size, m.Side)
				matchGroup.write([]byte(msg))
			case coinbaseX.StreamMsgInternalError:
				if m.Error == io.EOF {
					log.Println("restarting stream")
					go connect(q, serverHP)
				}
			}
		}
	}()
	go streamGroup.listenAndServe()
	matchGroup.listenAndServe()
}

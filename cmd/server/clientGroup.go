package main

import (
	"net"
	"sync"
)

// manage a group tcp clients

type clientGroup struct {
	hostPort    string
	connections map[net.Conn]net.Conn
	sync.Mutex
	server groupServer
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

func (cg *clientGroup) write(data []byte) {
	for _, c := range cg.getConnections() {
		err := cg.server.write(data)
		if err != nil {
			cg.remClient(c)
		}
	}
}

func (cg *clientGroup) listenAndServe() {
	l, err := net.Listen("tcp", cg.hostPort)
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

func (cg *clientGroup) serve(c net.Conn) {
	cg.addClient(c)
	defer cg.remClient(c)
	cg.server.serve(c)
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

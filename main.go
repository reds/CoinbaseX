package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type config struct {
	Passphrase string
	Key        string
	Secret     string
	Apihost    string
	Product_id string
}

var cfg config

func init() {
	buf, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".ssh", "coinbase.js"))
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(buf, &cfg)
	if err != nil {
		panic(err)
	}
}

func doSell(m *StreamMsg, e *env) ([]*CreateOrderRes, error) {
	p := m.Price + e.firstStep
	os := make([]*CreateOrderRes, 0)
	for i := 0; i < e.maxTimes; i++ {
		if e.forReal {
			o, err := CreateOrder("", p, e.sellSize, "sell", cfg.Product_id, "")
			if err != nil {
				// bail at the first error
				return os, err
			}
			os = append(os, o)
		} else {
			log.Println("CreateOrder Test", "", p, e.sellSize, "sell", cfg.Product_id, "")
			// o, err := CreateOrderTest("", p, e.sellSize, "sell", cfg.Product_id, "")
		}
		e.btcs -= e.sellSize
		p += e.step
	}
	return os, nil
}

func doBuy(m *StreamMsg, e *env) ([]*CreateOrderRes, error) {
	return nil, nil
}

type env struct {
	btcs, dollars             float64
	sellMinPrice, buyMaxPrice float64
	sellSize, buySize         float64
	firstStep, step           float64
	maxTimes                  int
	watchList                 map[string]*Order
	watchListLock             sync.Mutex
	forReal                   bool // not testing
}

func (e *env) watchAdd(o *Order) {
	e.watchListLock.Lock()
	defer e.watchListLock.Unlock()
	e.watchList[o.Id] = o
}

func (e *env) watched(id string) *Order {
	e.watchListLock.Lock()
	defer e.watchListLock.Unlock()
	if o, exists := e.watchList[id]; exists {
		return o
	}
	return nil
}

func (e *env) sellp(m *StreamMsg) bool {
	if m.Price > e.sellMinPrice && e.btcs > 0. {
		return true
	}
	return false
}

func (e *env) buyp(m *StreamMsg) bool {
	return false
}

func (e *env) refreshOrders() {
	orders, err := Orders()
	if err != nil {
		log.Println(err)
	}
	e.watchListLock.Lock()
	defer e.watchListLock.Unlock()
	e.watchList = make(map[string]*Order)
	for _, v := range orders {
		e.watchList[v.Id] = v
	}
}

func trade() {
	accts, err := Accounts()
	if err != nil {
		panic(err)
	}
	btcs := .0
	dollars := .0
	for _, v := range accts {
		if v.Currency == "BTC" {
			btcs, _ = v.Balance.Float64()
		}
		if v.Currency == "USD" {
			dollars, _ = v.Balance.Float64()
		}
	}
	q := make(chan *StreamMsg)
	err = stream(q, "")
	if err != nil {
		panic(err)
	}
	fmt.Println(btcs, dollars)
	e := &env{
		btcs:         btcs,
		dollars:      dollars,
		sellMinPrice: 247.7,
		buyMaxPrice:  240.0,
		sellSize:     .2,
		buySize:      .2,
		step:         .1,
		firstStep:    .3,
		maxTimes:     5,
		watchList:    make(map[string]*Order),
	}
	e.refreshOrders()
	for m := range q {
		if o := e.watched(m.Order_id); o != nil {
			fmt.Println("watch", o.Id)
		}
		if m.Type == StreamMsgMatch {
			if e.sellp(m) {
				_, err := doSell(m, e)
				if err != nil {
					fmt.Println("sell error", err)
				}
				e.refreshOrders()
			}
			if e.buyp(m) {
				_, err := doBuy(m, e)
				if err != nil {
					fmt.Println("buy error", err)
				}
				e.refreshOrders()
			}
		}
	}
}

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

func serveMatches() {
	q := make(chan *StreamMsg, 1)
	err := stream(q, "data2/stream.log")
	if err != nil {
		panic(err)
	}
	cg := &clientGroup{connections: make(map[net.Conn]net.Conn)}
	go func() {
		m := 0
		for {
			time.Sleep(time.Minute * 5)
			getBook("data2/bookEndM" + strconv.Itoa(m) + ".log")
			m += 5
		}
	}()
	go func() {
		for m := range q {
			switch m.Type {
			case StreamMsgMatch:
				msg := fmt.Sprintln(m.Price, m.Size, m.Side)
				cg.setLastMessage(msg)
				for _, c := range cg.getConnections() {
					_, err = c.Write([]byte(msg))
					if err != nil {
						cg.remClient(c)
					}
				}
			case StreamMsgInternalError:
				if m.Error == io.EOF {
					log.Println("restarting stream")
					go stream(q, "")
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

func main() {
	serveMatches()
}

func main2() {
	q := make(chan *StreamMsg)
	err := stream(q, "stream.log")
	if err != nil {
		panic(err)
	}
	seq, book, err := getBook("bookStart.log")
	if err != nil {
		panic(err)
	}
	n := 0
	for m := range q {
		if m.Sequence >= seq {
			break
		}
		n++
	}
	go func() {
		for m := range q {
			book.maintainBook(m)
		}
	}()
	go func() {
		h := 0
		for {
			time.Sleep(time.Minute * 5)
			h += 5
			getBook("bookEndM" + strconv.Itoa(h) + ".log")
		}
	}()
	fmt.Println("skipped", n)
	http.HandleFunc("/info", func(w http.ResponseWriter, req *http.Request) {
		handleInfo(w, req, book)
	})
	http.HandleFunc("/bids", func(w http.ResponseWriter, req *http.Request) {
		handleBids(w, req, book)
	})
	http.HandleFunc("/asks", func(w http.ResponseWriter, req *http.Request) {
		handleAsks(w, req, book)
	})
	http.HandleFunc("/bidasks", func(w http.ResponseWriter, req *http.Request) {
		handleBidAsks(w, req, book)
	})
	http.ListenAndServe("localhost:5000", nil)
}

func getTime() (float64, error) {
	form := &url.Values{}
	u := &url.URL{
		Scheme:   "https",
		Host:     cfg.Apihost,
		Path:     "/time",
		RawQuery: form.Encode(),
	}
	resp, err := http.Get(u.String())
	if err != nil {
		return 0, fmt.Errorf("Error url %s: %s", u.String(), err.Error())
	}
	buf, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return 0, fmt.Errorf("Error url %s: %s", u.String(), "reading response body")
	}
	type ts struct {
		Iso   string
		Epoch float64
	}
	var t ts
	err = json.Unmarshal(buf, &t)
	if err != nil {
		return 0, fmt.Errorf("Error url %s: %s", u.String(), "parsing response body")
	}
	return t.Epoch, nil
}

func checkTime() int {
	ots := time.Now().Unix()
	cbts, err := getTime()
	if err != nil {
		panic(err)
	}
	return int(ots - int64(cbts))
}

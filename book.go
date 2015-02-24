package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
)

type book struct {
	bids     *list.List
	bidsLock sync.Mutex
	asks     *list.List
	asksLock sync.Mutex
	msgCnt   map[string]int
}

type bidAsk struct {
	price float64
	size  float64
	id    string
}

var bidsComp = func(a, b float64) bool { return a < b }
var asksComp = func(a, b float64) bool { return a > b }

func httpgetBook() ([]byte, error) {
	return httpCoinbase("GET", "/products/"+cfg.Product_id+"/book", map[string]string{"level": "3"}, "")
}

func (b *book) maintainBook(msg *StreamMsg) {
	switch msg.Type {
	case StreamMsgReceived:
		b.msgCnt["Received"]++
	case StreamMsgOpen:
		b.msgCnt["Open"]++
		ba := &bidAsk{price: msg.Price, size: msg.Remaining_size, id: msg.Order_id}
		if msg.Side == StreamMsgBuy {
			b.addBid(ba)
		} else {
			b.addAsk(ba)
		}
	case StreamMsgDone:
		// filled or canceled
		b.removeOrder(msg.Order_id)
		b.msgCnt["Done"]++
	case StreamMsgMatch:
		b.reduceOrderSize(msg.Maker_order_id, msg.Size)
		fmt.Println("match", msg.Price, msg.Size, msg.Side)
		b.msgCnt["Match"]++
	case StreamMsgChange:
		b.changeOrderSize(msg.Order_id, msg.New_size)
		b.msgCnt["Change"]++
	case StreamMsgError:
		b.msgCnt["Error"]++
	case StreamMsgInternalError:
		b.msgCnt["InternalError"]++
		panic(msg)
	default:
		panic(msg)
	}
}

func (b *book) addBid(ba *bidAsk) {
	addToList(b.bids, b.bidsLock, bidsComp, ba)
}

func (b *book) addAsk(ba *bidAsk) {
	addToList(b.asks, b.asksLock, asksComp, ba)
}

func (b *book) removeOrder(orderid string) {
	removeOrderFromList(b.bids, b.bidsLock, orderid)
	removeOrderFromList(b.asks, b.asksLock, orderid)
}

func (b *book) changeOrderSize(orderid string, newSize float64) {
	changeOrderSizeOnList(b.bids, b.bidsLock, orderid, newSize)
	changeOrderSizeOnList(b.asks, b.asksLock, orderid, newSize)
}

// for a partial match
func (b *book) reduceOrderSize(orderid string, size float64) {
	// fmt.Println("reduce order size", orderid, size)
	reduceOrderSizeOnList(b.bids, b.bidsLock, orderid, size)
	reduceOrderSizeOnList(b.asks, b.asksLock, orderid, size)
}

func (b *book) getBids(n int) map[string]interface{} {
	b.bidsLock.Lock()
	defer b.bidsLock.Unlock()
	if n == 0 {
		n = b.bids.Len()
	}
	cn := 0
	bs := make([]map[string]interface{}, 0, n)
	for e := b.bids.Front(); e != nil; e = e.Next() {
		if cn == n {
			break
		}
		cn++
		a := e.Value.(*bidAsk)
		// want a copy, not a pointer to the orig
		bs = append(bs, map[string]interface{}{"price": a.price, "size": a.size, "id": a.id})
	}
	return map[string]interface{}{"bids": bs}
}

func (b *book) getAsks(n int) map[string]interface{} {
	b.asksLock.Lock()
	defer b.asksLock.Unlock()
	if n == 0 {
		n = b.asks.Len()
	}
	cn := 0
	bs := make([]map[string]interface{}, 0, n)
	for e := b.asks.Front(); e != nil; e = e.Next() {
		if cn == n {
			break
		}
		cn++
		a := e.Value.(*bidAsk)
		// want a copy, not a pointer to the orig
		bs = append(bs, map[string]interface{}{"price": a.price, "size": a.size, "id": a.id})
	}
	return map[string]interface{}{"asks": bs}
}

func (b *book) getBidAsks(n int) map[string]interface{} {
	b.bidsLock.Lock()
	defer b.bidsLock.Unlock()
	b.asksLock.Lock()
	defer b.asksLock.Unlock()
	if n == 0 {
		n = b.asks.Len()
	}
	cn := 0
	bs := make([]map[string]interface{}, 0, n)
	for e := b.bids.Front(); e != nil; e = e.Next() {
		if cn == n {
			break
		}
		cn++
		a := e.Value.(*bidAsk)
		// want a copy, not a pointer to the orig
		bs = append(bs, map[string]interface{}{"price": a.price, "size": a.size, "id": a.id})
	}
	cn = 0
	as := make([]map[string]interface{}, 0, n)
	for e := b.asks.Front(); e != nil; e = e.Next() {
		if cn == n {
			break
		}
		cn++
		a := e.Value.(*bidAsk)
		// want a copy, not a pointer to the orig
		as = append(bs, map[string]interface{}{"price": a.price, "size": a.size, "id": a.id})
	}
	return map[string]interface{}{"bids": bs, "asks": as}
}

func addToList(l *list.List, lock sync.Mutex, comp func(a, b float64) bool, ba *bidAsk) {
	lock.Lock()
	defer lock.Unlock()
	var ipoint *list.Element
	for e := l.Front(); e != nil; e = e.Next() {
		if comp(e.Value.(*bidAsk).price, ba.price) {
			ipoint = e
			break
		}
	}
	if ipoint == nil {
		l.PushBack(ba)
	} else {
		l.InsertBefore(ba, ipoint)
	}
}

func getBook(logFile string) (int64, *book, error) {
	buf, err := httpgetBook()
	if err != nil {
		return 0, nil, err
	}
	// log to file
	if logFile != "" {
		fp, err := os.Create(logFile)
		if err != nil {
			panic(err)
		}
		fmt.Fprintln(fp, string(buf))
		fp.Close()
	}
	return makeBook(buf)
}

func makeBook(buf []byte) (int64, *book, error) {
	bk := &book{msgCnt: make(map[string]int)} // need locks
	var b map[string]interface{}
	err := json.Unmarshal(buf, &b)
	if err != nil {
		return 0, nil, err
	}
	bids, err := makeBAs(b["bids"].([]interface{}), bk.bidsLock, bidsComp)
	if err != nil {
		return 0, nil, err
	}
	asks, err := makeBAs(b["asks"].([]interface{}), bk.asksLock, asksComp)
	if err != nil {
		return 0, nil, err
	}
	bk.bids = bids
	bk.asks = asks
	return int64(b["sequence"].(float64)), bk, nil
}

func makeBAs(in []interface{}, lock sync.Mutex, comp func(float64, float64) bool) (*list.List, error) {
	if len(in) < 2 {
		return nil, fmt.Errorf("too few ")
	}
	lock.Lock()
	l := list.New()
	// first
	ba1 := makeBA(in[0])
	l.PushBack(ba1)
	// second
	ba2 := makeBA(in[1])
	if comp(ba1.price, ba2.price) {
		l.PushFront(ba2)
	} else {
		l.PushBack(ba2)
	}
	lock.Unlock()
	for _, v := range in[2:] {
		addToList(l, lock, comp, makeBA(v))

	}
	return l, nil
}

func makeBA(in interface{}) *bidAsk {
	ba := in.([]interface{})
	price, err := strconv.ParseFloat(ba[0].(string), 64)
	if err != nil {
		panic(err)
	}
	size, err := strconv.ParseFloat(ba[1].(string), 64)
	if err != nil {
		panic(err)
	}
	return &bidAsk{price: price, size: size, id: ba[2].(string)}
}

func removeOrderFromList(l *list.List, lock sync.Mutex, orderid string) {
	lock.Lock()
	defer lock.Unlock()
	var r *list.Element
	for e := l.Front(); e != nil; e = e.Next() {
		a := e.Value.(*bidAsk)
		if a.id == orderid {
			r = e
			break
		}
	}
	if r != nil {
		l.Remove(r)
	}
}

func changeOrderSizeOnList(l *list.List, lock sync.Mutex, orderid string, newSize float64) {
	lock.Lock()
	defer lock.Unlock()
	for e := l.Front(); e != nil; e = e.Next() {
		a := e.Value.(*bidAsk)
		if a.id == orderid {
			a.size = newSize
			return
		}
	}
}

func reduceOrderSizeOnList(l *list.List, lock sync.Mutex, orderid string, size float64) {
	lock.Lock()
	defer lock.Unlock()
	for e := l.Front(); e != nil; e = e.Next() {
		a := e.Value.(*bidAsk)
		if a.id == orderid {
			a.size -= size
			return
		}
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"sync"
	"time"

	"github.com/reds/coinbaseX"
)

func doSell(m *coinbaseX.StreamMsg, e *env) ([]*coinbaseX.CreateOrderRes, error) {
	p := m.Price + e.firstStep
	os := make([]*coinbaseX.CreateOrderRes, 0)
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

func doBuy(m *coinbaseX.StreamMsg, e *env) ([]*CreateOrderRes, error) {
	return nil, nil
}

type env struct {
	btcs, dollars             float64
	sellMinPrice, buyMaxPrice float64
	sellSize, buySize         float64
	firstStep, step           float64
	maxTimes                  int
	watchList                 map[string]*coinbaseX.Order
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

func (e *env) sellp(m *coinbaseX.StreamMsg) bool {
	if m.Price > e.sellMinPrice && e.btcs > 0. {
		return true
	}
	return false
}

func (e *env) buyp(m *coinbaseX.StreamMsg) bool {
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
	q := make(chan *coinbaseX.StreamMsg)
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
		if m.Type == coinbaseX.StreamMsgMatch {
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

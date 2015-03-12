package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"

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

func connectCoinbaseWS(cb *coinbaseX.CoinbaseX, q chan *coinbaseX.StreamMsg) {
	err := cb.Stream(q)
	if err != nil {
		panic(err)
	}
}

func main() {
	cb, err := coinbaseX.New(filepath.Join(os.Getenv("HOME"), ".ssh", "coinbase.js"))
	if err != nil {
		panic(err)
	}
	btcs := flag.Float64("btcs", 1.0, "num btcs")
	priceBump := flag.Float64("bump", .3, "sell price bump")
	extra := flag.Float64("extra", .05, "over the mark amount")
	flag.Parse()
	q := make(chan *coinbaseX.StreamMsg, 1)
	go connect(q, serverHP)
	for {
		buyid, price, err := buyAtMatch(cb, q, *extra, *btcs)
		if err != nil {
			panic(err)
		}
		sellid, err := sellWhenDone(cb, q, buyid, price+*priceBump, *btcs)
		if err != nil {
			panic(err)
		}
		waitForDone(cb, q, sellid)
	}
}

func buyAtMatch(cb *coinbaseX.CoinbaseX, q chan *coinbaseX.StreamMsg, overTheMark, size float64) (string, float64, error) {
	for m := range q {
		switch m.Type {
		case coinbaseX.StreamMsgMatch:
			bo, err := cb.CreateOrder("", m.Price+overTheMark, size, "buy", cb.Config.Product_id, "")
			if err != nil {
				return "", 0., err
			}
			fmt.Println(bo)
			return bo.Id, m.Price, nil
		}
	}
	return "", 0., fmt.Errorf("returned from stream")
}

func sellWhenDone(cb *coinbaseX.CoinbaseX, q chan *coinbaseX.StreamMsg, id string, price, size float64) (string, error) {
	for m := range q {
		if m.Order_id == id {
			fmt.Println("swd", id, m)
		}
		if m.Maker_order_id == id {
			fmt.Println("swd maker", id, m)
		}
		if m.Taker_order_id == id {
			fmt.Println("swd taker", id, m)
		}
		switch m.Type {
		case coinbaseX.StreamMsgDone:
			if m.Order_id == id {
				bo, err := cb.CreateOrder("", price, size, "sell", cb.Config.Product_id, "")
				if err != nil {
					return "", err
				}
				fmt.Println(bo)
				return bo.Id, err
			}
		}
	}
	return "", fmt.Errorf("returned from stream")
}

func waitForDone(cb *coinbaseX.CoinbaseX, q chan *coinbaseX.StreamMsg, id string) {
	for m := range q {
		if m.Order_id == id {
			fmt.Println("wfd", id, m)
		}
		if m.Maker_order_id == id {
			fmt.Println("wfd maker", id, m)
		}
		if m.Taker_order_id == id {
			fmt.Println("wfd taker", id, m)
		}
		switch m.Type {
		case coinbaseX.StreamMsgDone:
			if m.Order_id == id {
				fmt.Println(m)
				return
			}
		}
	}
}

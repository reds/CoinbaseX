package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/reds/coinbaseX"
)

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
	for {
		buyid, price, err := buyAtMatch(cb, q, 1.0)
		if err != nil {
			panic(err)
		}
		sellid, err := sellWhenDone(cb, q, buyid, price+.5, 1.0)
		if err != nil {
			panic(err)
		}
		waitForDone(cb, q, sellid)
	}
}

func buyAtMatch(cb *coinbaseX.CoinbaseX, q chan *coinbaseX.StreamMsg, size float64) (string, float64, error) {
	for m := range q {
		switch m.Type {
		case coinbaseX.StreamMsgMatch:
			bo, err := cb.CreateOrder("", m.Price, size, "buy", cb.Config.Product_id, "")
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
		switch m.Type {
		case coinbaseX.StreamMsgDone:
			if m.Order_id == id {
				fmt.Println(m)
				return
			}
		}
	}
}

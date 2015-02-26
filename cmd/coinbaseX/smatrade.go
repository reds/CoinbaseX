package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/reds/coinbaseX"
)

func main() {
	cb, err := coinbaseX.New(filepath.Join(os.Getenv("HOME"), ".ssh", "coinbase.js"))
	if err != nil {
		panic(err)
	}
	// simplistic test: without using book
	//                  pretending buy/sells are fully filled at match price
	btcs := 1.0
	dollars := 0.0

	// buy: sma short goes below (+spread) sma long
	// sell: sma short goes above (+spread) sma long
	smaB := newSMA(20)
	smaA := newSMA(80)
	spread := 1.0

	q := make(chan *coinbaseX.StreamMsg, 1)
	err = cb.Stream(q)
	if err != nil {
		panic(err)
	}
	for m := range q {
		switch m.Type {
		case coinbaseX.StreamMsgMatch:
			v := m.Price
			vB := smaB.next(v)
			vA := smaA.next(v)
			if vB != 0 && vA != 0 {
				if vB+spread < vA {
					// buy
					if dollars > 0 {
						btcs = dollars / v
						dollars = 0
						fmt.Println("sell at", v, "(", dollars, btcs, ")")
					}
				}
				if vA < vB-spread {
					// sell
					if btcs > 0 {
						dollars = btcs * v
						btcs = 0
						fmt.Println("sell at", v, "(", dollars, btcs, ")")
					}
				}
			}
		case coinbaseX.StreamMsgInternalError:
			if m.Error == io.EOF {
				log.Println("restarting stream")
				go cb.Stream(q)
			}
		}
	}
}

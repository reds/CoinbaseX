package main

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/reds/coinbaseX"
)

func main() {
	cb, err := coinbaseX.New(filepath.Join(os.Getenv("HOME"), ".ssh", "coinbase.js"))
	if err != nil {
		panic(err)
	}
	q := make(chan *coinbaseX.StreamMsg, 1)
	cb.Debug.WriteLog = true
	cb.Debug.StreamLog = "/tmp/stream.log"
	err = cb.Stream(q)
	if err != nil {
		panic(err)
	}
	cb.Debug.BookLog = "/tmp/bookEndM0.log"
	_, _, err = cb.Book()
	if err != nil {
		panic(err)
	}
	h := 0
	for {
		time.Sleep(time.Minute * 5)
		h += 5
		cb.Debug.BookLog = "/tmp/bookEndM" + strconv.Itoa(h) + ".log"
		_, _, err := cb.Book()
		if err != nil {
			panic(err)
		}
	}
}

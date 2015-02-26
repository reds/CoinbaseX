package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/reds/coinbaseX"
)

func main() {
	cb, err := coinbaseX.New(filepath.Join(os.Getenv("HOME"), ".ssh", "coinbase.js"))
	if err != nil {
		panic(err)
	}
	q := make(chan *coinbaseX.StreamMsg)
	err = cb.Stream(q)
	if err != nil {
		panic(err)
	}
	seq, book, err := cb.Book()
	if err != nil {
		panic(err)
	}
	n := 0
	// find the current book sequence number in the stream
	for m := range q {
		if m.Sequence >= seq {
			break
		}
		n++
	}
	go func() {
		for m := range q {
			book.MaintainBook(m)
		}
	}()

	http.HandleFunc("/info", func(w http.ResponseWriter, req *http.Request) {
		coinbaseX.HandleInfo(w, req, book)
	})
	http.HandleFunc("/bids", func(w http.ResponseWriter, req *http.Request) {
		coinbaseX.HandleBids(w, req, book)
	})
	http.HandleFunc("/asks", func(w http.ResponseWriter, req *http.Request) {
		coinbaseX.HandleAsks(w, req, book)
	})
	http.HandleFunc("/bidasks", func(w http.ResponseWriter, req *http.Request) {
		coinbaseX.HandleBidAsks(w, req, book)
	})
	http.ListenAndServe("localhost:5000", nil)
}

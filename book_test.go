package coinbaseX

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

func TestMaintainBook(t *testing.T) {
	// test on saved data
	buf, err := ioutil.ReadFile("data1/bookEndM0.log")
	if err != nil {
		t.Fatal(err)
	}
	seq1, book1, err := makeBook(buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(seq1)
	buf, err = ioutil.ReadFile("data1/bookEndM1070.log")
	if err != nil {
		t.Fatal(err)
	}
	seq2, book2, err := makeBook(buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(seq2)
	dumpBook(book2, "book2.dump")

	fs, err := os.Open("data1/stream.log")
	if err != nil {
		t.Fatal(err)
	}
	s := bufio.NewScanner(fs)
	for s.Scan() {
		var kv map[string]interface{}
		err = json.Unmarshal(s.Bytes(), &kv)
		if err != nil {
			t.Fatal(err)
		}
		msg, err := parseStream(kv)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Sequence == seq1 {
			t.Log("found seq1")
			break
		}
	}
	var wg sync.WaitGroup
	wg.Add(1)
	msgQ := make(chan *StreamMsg)
	go func() {
		for msg := range msgQ {
			book1.MaintainBook(msg)
		}
		wg.Done()
	}()
	for s.Scan() {
		var kv map[string]interface{}
		err = json.Unmarshal(s.Bytes(), &kv)
		if err != nil {
			t.Fatal(err)
		}
		msg, err := parseStream(kv)
		if err != nil {
			t.Fatal(err)
		}
		msgQ <- msg
		if msg.Sequence == seq2 {
			t.Log("found seq2")
			close(msgQ)
			break
		}
	}
	wg.Wait()
	dumpBook(book1, "book1.dump")
}

func TestBook(t *testing.T) {
	cb, err := New(filepath.Join(os.Getenv("HOME"), ".ssh", "coinbase.js"))
	if err != nil {
		t.Fatal(err)
	}
	cb.Debug.WriteLog = true
	cb.Debug.BookLog = "book.log"
	seq, book, err := cb.Book()
	if err != nil {
		panic(err)
	}
	fmt.Println(seq)
	cnt := 0
	for e := book.bids.Front(); e != nil; e = e.Next() {
		fmt.Println(e.Value.(*bidAsk))
		if cnt > 10 {
			break
		}
		cnt++
	}
	fmt.Println()
	cnt = 0
	for e := book.asks.Front(); e != nil; e = e.Next() {
		fmt.Println(e.Value.(*bidAsk))
		if cnt > 10 {
			break
		}
		cnt++
	}
}

func dumpBook(b *Book, fn string) {
	f1, err := os.Create(fn)
	if err != nil {
		panic(err)
	}
	for e := b.bids.Front(); e != nil; e = e.Next() {
		ba := e.Value.(*bidAsk)
		fmt.Fprintf(f1, "%.02f,%.08f,%s\n", ba.price, ba.size, ba.id)
	}
	for e := b.asks.Front(); e != nil; e = e.Next() {
		ba := e.Value.(*bidAsk)
		fmt.Fprintf(f1, "%.02f,%.08f,%s\n", ba.price, ba.size, ba.id)
	}
	f1.Close()
}

func TestjsonParse(t *testing.T) {
	buf := []byte(`{"sequence":11934629,"bids":[["241.01000000","0.64500000","867ec58b-f889-45a5-8045-8a6fa952a2a8"]]}`)
	var m map[string]interface{}
	err := json.Unmarshal(buf, &m)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(m)

	price, err := strconv.ParseFloat("0.64500000", 64)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(price)
}

package coinbaseX

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func Testorders(t *testing.T) {
	cb, err := New(filepath.Join(os.Getenv("HOME"), ".ssh", "coinbase.js"))
	if err != nil {
		t.Fatal(err)
	}
	watch := make(map[string]*Order)
	orders, err := cb.Orders()
	for _, v := range orders {
		watch[v.Id] = v
	}
	q := make(chan *StreamMsg)
	err = cb.Stream(q)
	if err != nil {
		t.Fatal(err)
	}
	for m := range q {
		if v, exists := watch[m.Order_id]; exists {
			fmt.Println(v)
		}
	}
}

func Testparsetime(t *testing.T) {
	ts, err := time.Parse("2006-01-02T15:04:05Z", "2015-02-11T21:39:01.51979Z")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ts)
}

package main

import (
	"encoding/json"
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
	dollars := .0
	dollarsHold := .0
	btcs := .0
	btcsHold := .0
	accts, err := cb.Accounts()
	if err != nil {
		panic(err)
	}
	for _, acct := range accts {
		if acct.Currency == "USD" {
			dollars += jnumber(acct.Balance)
			dollarsHold += jnumber(acct.Hold)
		}
		if acct.Currency == "BTC" {
			btcs += jnumber(acct.Balance)
			btcsHold += jnumber(acct.Hold)
		}
	}
	orders, err := cb.Orders()
	if err != nil {
		panic(err)
	}
	for _, o := range orders {
		if o.Side == "sell" && o.Status == "open" {
			dollars += jnumber(o.Size) * jnumber(o.Price)
		}
	}
	fmt.Println(dollars)
}

func jnumber(n json.Number) float64 {
	v, err := n.Float64()
	if err != nil {
		panic(err)
	}
	return v
}

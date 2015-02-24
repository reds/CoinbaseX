package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
)

func Testcandles(t *testing.T) {
	buf, err := ioutil.ReadFile("candles.log")
	if err != nil {
		t.Fatal(err)
	}
	var c [][]float64
	err = json.Unmarshal(buf, &c)
	if err != nil {
		panic(err)
	}
	fmt.Println(c[0][0])
}

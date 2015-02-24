package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Account struct {
	Id         string
	Currency   string
	Balance    json.Number
	Hold       json.Number
	Available  json.Number
	Profile_id string
}

func Accounts() ([]Account, error) {
	buf, err := httpCoinbase("GET", "/accounts", nil, "")
	if err != nil {
		return nil, err
	}
	var res []Account // map[string]interface{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

type Order struct {
	Id          string
	Created_id  string
	Size        json.Number
	Price       json.Number
	Side        string
	Product_id  string
	Done_at     string
	Done_reason string
	Status      string
	Settled     bool
	Filled_size json.Number
	Fill_fees   json.Number
}

func Orders() ([]*Order, error) {
	buf, err := httpCoinbase("GET", "/orders", nil, "")
	if err != nil {
		return nil, err
	}
	var res []*Order
	err = json.Unmarshal(buf, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func CancelOrder(id string) (bool, error) {
	buf, err := httpCoinbase("DELETE", "/orders/"+id, nil, "")
	if err != nil {
		return false, err
	}
	if string(buf) == "OK" {
		return true, nil
	}
	return false, nil
}

type CreateOrderRes struct {
	Id         string
	Price      json.Number
	Size       json.Number
	Product_id string
	Side       string
	Stp        string
}

func CreateOrder(myid string, price, btcs float64, side, product, stp string) (*CreateOrderRes, error) {
	args := map[string]string{
		//"client_oid": myid,
		"price":      strconv.FormatFloat(price, 'f', -1, 64),
		"size":       strconv.FormatFloat(btcs, 'f', -1, 64),
		"side":       side,
		"product_id": product,
	}
	body, _ := json.Marshal(args)
	buf, err := httpCoinbase("POST", "/orders", nil, string(body))
	if err != nil {
		return nil, err
	}
	var res CreateOrderRes
	err = json.Unmarshal(buf, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func candles(prod string, startts, endts time.Time, granularity int) {
	args := map[string]string{
		"start":       startts.Format("2006-01-02T15:04:05.999999999z"),
		"end":         endts.Format("2006-01-02T15:04:05.999999999z"),
		"granularity": strconv.Itoa(granularity),
	}
	fmt.Println(args)
	buf, err := httpCoinbase("GET", "/products/"+prod+"/candles", args, "")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(buf))
}

func httpCoinbase(method, path string, args map[string]string, body string) ([]byte, error) {
	form := &url.Values{}
	for k, v := range args {
		form.Set(k, v)
	}
	u := &url.URL{
		Scheme:   "https",
		Host:     cfg.Apihost,
		Path:     path,
		RawQuery: form.Encode(),
	}
	ts := time.Now().Unix()
	key, err := base64.StdEncoding.DecodeString(cfg.Secret)
	if err != nil {
		return nil, err
	}
	hm := hmac.New(sha256.New, key)
	hm.Write([]byte(strconv.FormatInt(ts, 10) + method + u.Path + body))
	sig := base64.StdEncoding.EncodeToString(hm.Sum(nil))

	var b bytes.Buffer
	b.WriteString(body)
	req, _ := http.NewRequest(method, u.String(), &b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("CB-ACCESS-KEY", cfg.Key)
	req.Header.Set("CB-ACCESS-SIGN", sig)
	req.Header.Set("CB-ACCESS-TIMESTAMP", strconv.FormatInt(ts, 10))
	req.Header.Set("CB-ACCESS-PASSPHRASE", cfg.Passphrase)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error url %s: %s", u.String(), err.Error())
	}
	buf, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("Error url %s: %s", u.String(), "reading response body")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error bad status code returned %s\n%s\n", resp.Status, string(buf))
	}
	fmt.Println(string(buf))
	return buf, nil
}

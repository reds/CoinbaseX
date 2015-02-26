package coinbaseX

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

func HandleInfo(w http.ResponseWriter, req *http.Request, book *Book) {
	book.bidsLock.Lock()
	defer book.bidsLock.Unlock()
	book.asksLock.Lock()
	defer book.asksLock.Unlock()
	writeJson(w, map[string]interface{}{"bids": book.bids.Len(), "asks": book.asks.Len()})
}

func HandleBids(w http.ResponseWriter, req *http.Request, book *Book) {
	n, _ := strconv.Atoi(req.FormValue("n"))
	writeJson(w, book.getBids(n))
}

func HandleAsks(w http.ResponseWriter, req *http.Request, book *Book) {
	n, _ := strconv.Atoi(req.FormValue("n"))
	writeJson(w, book.getAsks(n))
}

func HandleBidAsks(w http.ResponseWriter, req *http.Request, book *Book) {
	n, _ := strconv.Atoi(req.FormValue("n"))
	writeJson(w, book.getBidAsks(n))
}

func writeJson(w http.ResponseWriter, kv map[string]interface{}) {
	buf, err := json.Marshal(kv)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(buf)
}

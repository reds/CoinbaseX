package coinbaseX

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"golang.org/x/net/websocket"
)

type StreamMsgType int

const (
	StreamMsgReceived StreamMsgType = iota
	StreamMsgOpen
	StreamMsgDone
	StreamMsgMatch
	StreamMsgChange
	StreamMsgError
	StreamMsgInternalError
)

func (m StreamMsgType) String() string {
	switch m {
	case StreamMsgReceived:
		return "RECEIVED:"
	case StreamMsgOpen:
		return "OPEN:    "
	case StreamMsgDone:
		return "DONE:    "
	case StreamMsgMatch:
		return "MATCH:   "
	case StreamMsgChange:
		return "CHANGE:  "
	case StreamMsgError:
		return "ERROR:   "
	case StreamMsgInternalError:
		return "INTERNAL:"
	}
	return "UNKNOWN: "
}

type StreamMsgSide int

const (
	StreamMsgBuy StreamMsgSide = iota
	StreamMsgSell
)

func (s StreamMsgSide) String() string {
	switch s {
	case StreamMsgBuy:
		return "BUY "
	case StreamMsgSell:
		return "SELL"
	default:
		return "UNKNOWN"
	}
}

type StreamMsgReason int

const (
	StreamMsgFilled StreamMsgReason = iota
	StreamMsgCanceled
)

func (s StreamMsgReason) String() string {
	switch s {
	case StreamMsgFilled:
		return "FILLED  "
	case StreamMsgCanceled:
		return "CANCELED"
	default:
		return "UNKNOWN"
	}
}

type StreamMsg struct {
	Type           StreamMsgType
	Sequence       int64
	Order_id       string
	Size           float64
	Price          float64
	Side           StreamMsgSide
	Reason         StreamMsgReason
	Remaining_size float64
	Trade_id       int64
	Maker_order_id string
	Taker_order_id string
	Time           time.Time
	New_size       float64
	Old_size       float64
	Message        string
	Json           []byte
	Error          error
}

func (m *StreamMsg) String() string {
	return fmt.Sprint(m.Type, m.Side, m.Reason, fmt.Sprintf("%8.2f", m.Price), fmt.Sprintf("%8.2f", m.Size), "   ", m.Order_id)
}

func (cb *CoinbaseX) Stream(q chan *StreamMsg) error {
	ws, err := websocket.Dial("wss://ws-feed.exchange.coinbase.com", "wss", "http://chat52.com/coinbase")
	if err != nil {
		return err
	}
	sub := `{"type": "subscribe","product_id":"` + cb.Config.Product_id + `"}`
	err = websocket.Message.Send(ws, sub)
	if err != nil {
		return err
	}
	// get one msg before forking
	var kv map[string]interface{}
	err = websocket.JSON.Receive(ws, &kv)
	if err != nil {
		return err
	}
	msg, err := parseStream(kv)
	if err != nil {
		return err
	}
	q <- msg
	go func() {
		var fp *os.File
		if cb.Debug.WriteLog && cb.Debug.StreamLog != "" {
			var err error
			fp, err = os.Create(cb.Debug.StreamLog)
			if err != nil {
				panic(err)
			}
		}
		for {
			var data []byte
			err := websocket.Message.Receive(ws, &data)
			if err != nil {
				q <- &StreamMsg{Type: StreamMsgInternalError, Message: err.Error(), Error: err}
				return
			}
			if fp != nil {
				fmt.Fprintln(fp, string(data))
			}
			msg, err := ParseMsg(data)
			if err != nil {
				q <- &StreamMsg{Type: StreamMsgInternalError, Message: err.Error(), Error: err}
				return
			}
			q <- msg
		}
	}()
	return nil
}

func ParseMsg(data []byte) (*StreamMsg, error) {
	var kv map[string]interface{}
	err := json.Unmarshal(data, &kv)
	if err != nil {
		return nil, err
	}
	msg, err := parseStream(kv)
	if err != nil {
		return nil, err
	}
	msg.Json = data
	return msg, nil
}

func parseStream(kv map[string]interface{}) (*StreamMsg, error) {
	var sm StreamMsg
	var err error
	if v := kv["sequence"]; v != nil {
		sm.Sequence = int64(v.(float64))
	}
	if v := kv["order_id"]; v != nil {
		sm.Order_id = v.(string)
	}
	if v := kv["side"]; v != nil {
		switch v.(string) {
		case "buy":
			sm.Side = StreamMsgBuy
		case "sell":
			sm.Side = StreamMsgSell
		default:
			return nil, fmt.Errorf("parsing side: %s", v.(string))
		}
	}
	if v := kv["reason"]; v != nil {
		switch v.(string) {
		case "filled":
			sm.Reason = StreamMsgFilled
		case "canceled":
			sm.Reason = StreamMsgCanceled
		default:
			return nil, fmt.Errorf("parsing reason: %s", v.(string))
		}
	}
	if v := kv["price"]; v != nil {
		sm.Price, err = strconv.ParseFloat(v.(string), 64)
		if err != nil {
			return nil, fmt.Errorf("parsing price: %s :%s", v.(string), err.Error())
		}
	}
	if v := kv["size"]; v != nil {
		sm.Size, err = strconv.ParseFloat(v.(string), 64)
		if err != nil {
			return nil, fmt.Errorf("parsing size: %s :%s", v.(string), err.Error())
		}
	}
	if v := kv["remaining_size"]; v != nil {
		sm.Remaining_size, err = strconv.ParseFloat(v.(string), 64)
		if err != nil {
			return nil, fmt.Errorf("parsing remaining_size: %s :%s", v.(string), err.Error())
		}
	}
	if v := kv["trade_id"]; v != nil {
		sm.Trade_id = int64(v.(float64))
	}
	if v := kv["maker_order_id"]; v != nil {
		sm.Maker_order_id = v.(string)
	}
	if v := kv["taker_order_id"]; v != nil {
		sm.Taker_order_id = v.(string)
	}
	if v := kv["time"]; v != nil {
		//   2006-01-02T15:04:05Z07:00
		//   2014-11-07T08:19:27.028459Z
		sm.Time, err = time.Parse("2006-01-02T15:04:05Z", v.(string))
		if err != nil {
			return nil, fmt.Errorf("parsing time: %s :%s", v.(string), err.Error())
		}
	}
	if v := kv["new_size"]; v != nil {
		sm.New_size, err = strconv.ParseFloat(v.(string), 64)
		if err != nil {
			return nil, fmt.Errorf("parsing new_size: %s :%s", v.(string), err.Error())
		}
	}
	if v := kv["old_size"]; v != nil {
		sm.Old_size, err = strconv.ParseFloat(v.(string), 64)
		if err != nil {
			return nil, fmt.Errorf("parsing old_size: %s :%s", v.(string), err.Error())
		}
	}
	if v := kv["message"]; v != nil {
		sm.Message = v.(string)
	}
	switch kv["type"].(string) {
	case "received":
		sm.Type = StreamMsgReceived
	case "open":
		sm.Type = StreamMsgOpen
	case "done":
		sm.Type = StreamMsgDone
	case "match":
		sm.Type = StreamMsgMatch
	case "change":
		sm.Type = StreamMsgChange
	case "error":
		sm.Type = StreamMsgError
	default:
		return nil, fmt.Errorf("parsing type: %s", kv["type"].(string))
	}
	return &sm, nil
}

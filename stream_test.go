package coinbaseX

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStringer(t *testing.T) {
	for _, j := range js() {
		msg, err := ParseMsg([]byte(j))
		if err != nil {
			t.Fatal(err)
		}
		t.Log(msg)
	}
}

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

func js() [][]byte {
	return [][]byte{
		[]byte(`{"type":"received","sequence":23292623,"order_id":"3f179443-01fd-46e1-80ef-99f4930b2b9f","size":"0.30000000","price":"278.40000000","side":"buy","client_oid":"03bfb6c1-c385-11e4-9bd8-df884b2c0122","time":"2015-03-05T22:14:55.868859Z"}`),
		[]byte(`{"type":"open","sequence":23292624,"side":"buy","price":"278.40000000","order_id":"3f179443-01fd-46e1-80ef-99f4930b2b9f","remaining_size":"0.30000000","time":"2015-03-05T22:14:55.869044Z"}`),
		[]byte(`{"type":"received","sequence":23292625,"order_id":"4892ada1-4f42-4115-adbb-817ba79be74a","size":"0.30000000","price":"278.33000000","side":"buy","client_oid":"024a7be0-c385-11e4-9bd8-df884b2c0122","time":"2015-03-05T22:14:55.946218Z"}`),
		[]byte(`{"type":"open","sequence":23292626,"side":"buy","price":"278.33000000","order_id":"4892ada1-4f42-4115-adbb-817ba79be74a","remaining_size":"0.30000000","time":"2015-03-05T22:14:55.946414Z"}`),
		[]byte(`{"type":"received","sequence":23292627,"order_id":"7f8d2b70-9aee-4339-90b2-a8cfa6f8380a","size":"0.30000000","price":"278.87000000","side":"sell","client_oid":"024a54d0-c385-11e4-9bd8-df884b2c0122","time":"2015-03-05T22:14:55.984367Z"}`),
		[]byte(`{"type":"open","sequence":23292628,"side":"sell","price":"278.87000000","order_id":"7f8d2b70-9aee-4339-90b2-a8cfa6f8380a","remaining_size":"0.30000000","time":"2015-03-05T22:14:55.984571Z"}`),
		[]byte(`{"type":"done","price":"263.87000000","side":"buy","remaining_size":"3.03220000","sequence":23292629,"order_id":"ec6dc7b9-4602-454c-8acf-6fa7d121391f","reason":"canceled","time":"2015-03-05T22:14:56.06171Z"}`),
		[]byte(`{"type":"done","price":"278.33000000","side":"buy","remaining_size":"0.30000000","sequence":23292630,"order_id":"4892ada1-4f42-4115-adbb-817ba79be74a","reason":"canceled","time":"2015-03-05T22:14:56.172878Z"}`),
		[]byte(`{"type":"done","price":"278.87000000","side":"sell","remaining_size":"0.30000000","sequence":23292631,"order_id":"7f8d2b70-9aee-4339-90b2-a8cfa6f8380a","reason":"canceled","time":"2015-03-05T22:14:56.230636Z"}`),
		[]byte(`{"type":"done","price":"285.12000000","side":"sell","remaining_size":"118.62900000","sequence":23292632,"order_id":"dafcfbc4-64be-4b84-af1c-1787f7cc5d3e","reason":"canceled","time":"2015-03-05T22:14:56.437926Z"}`),
	}
}

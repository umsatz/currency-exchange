package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	time "time"

	"github.com/gorilla/mux"
	. "github.com/umsatz/currency-exchange/data"
)

var dataDirectory string

func init() {
	flag.StringVar(&dataDirectory, "data", "", "path to data directory")
	flag.Parse()

	if fileInfo, err := os.Stat(dataDirectory); err != nil {
		log.Fatalf(`unable to stat %v: %v`, dataDirectory, err)
	} else if !fileInfo.IsDir() {
		log.Fatalf(`%v is no directory`, dataDirectory)
	}
}

type shortExchangeInfo struct {
	Currency string  `json:"currency"`
	Rate     float32 `json:"rate"`
}

type exchangeInfo struct {
	Date string `json:"date"`
	shortExchangeInfo
}

func LookEnvelop(dateStr string) *Envelop {
	// correct weekend offset
	time, err := time.Parse("2006-01-02", dateStr)
	var date string = time.Format("2006-01-02")
	if time.Weekday() == 0 {
		date = time.AddDate(0, 0, -2).Format("2006-01-02")
	} else if time.Weekday() == 6 {
		date = time.AddDate(0, 0, -1).Format("2006-01-02")
	}

	handle, err := os.OpenFile(fmt.Sprintf(`%v/%v.xml`, dataDirectory, date), os.O_RDONLY, 0660)
	if err != nil {
		fmt.Printf("unable to open file: %#v", err)
		return nil
	}
	defer handle.Close()

	envelop := Envelop{}
	decoder := xml.NewDecoder(handle)
	if err := decoder.Decode(&envelop); err != nil {
		fmt.Printf("unable to decode xml")
		return nil
	}
	return &envelop
}

func LookupCurrencyExchange(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	vars := mux.Vars(req)

	envelop := LookEnvelop(vars["date"])
	if envelop == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	cube := envelop.Cubes[0]

	currency := string(bytes.ToUpper([]byte(vars["currency"])))
	var exchange Exchange
	if currency == "EUR" {
		exchange.Currency = "EUR"
		exchange.Rate = 1.0
	} else {
		for _, ex := range cube.Exchanges {
			if ex.Currency == currency {
				exchange = ex
			}
		}
	}

	if exchange == (Exchange{}) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	info := exchangeInfo{time.Time(cube.Date).Format("2006-01-02"), shortExchangeInfo{exchange.Currency, exchange.Rate}}

	bytes, err := json.Marshal(info)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	io.WriteString(w, string(bytes))
}

type exchangeInfoCollection struct {
	Date      string              `json:"date"`
	Exchanges []shortExchangeInfo `json:"exchanges"`
}

func ListCurrencyExchange(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	vars := mux.Vars(req)

	envelop := LookEnvelop(vars["date"])
	cube := envelop.Cubes[0]

	exchangeInfos := make([]shortExchangeInfo, len(cube.Exchanges))
	for i, info := range cube.Exchanges {
		exchangeInfos[i] = shortExchangeInfo{info.Currency, info.Rate}
	}

	infoCollection := exchangeInfoCollection{time.Time(cube.Date).Format("2006-01-02"), exchangeInfos}
	bytes, err := json.Marshal(infoCollection)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	io.WriteString(w, string(bytes))
}

func logHandler(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%v %v", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	}
}

func main() {
	var port string = os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	l, err := net.Listen("tcp", "0.0.0.0:"+port)
	if nil != err {
		log.Fatalln(err)
	}
	log.Println("listening on %v", l.Addr())

	r := mux.NewRouter()
	r.Handle("/{date}/{currency}", logHandler(http.HandlerFunc(LookupCurrencyExchange))).Methods("GET")
	r.Handle("/{date}", logHandler(http.HandlerFunc(ListCurrencyExchange))).Methods("GET")
	http.Serve(l, r)
}

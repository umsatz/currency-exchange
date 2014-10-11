package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/golang/groupcache"
	"github.com/umsatz/currency-exchange/data"
)

type fileSystemProvider struct {
	dataDirectory string
	cache         *groupcache.Group
}

type shortExchangeInfo struct {
	Currency string  `json:"currency"`
	Rate     float32 `json:"rate"`
}

type exchangeInfo struct {
	Date string `json:"date"`
	shortExchangeInfo
}

func (provider *fileSystemProvider) getExchangeRateData(ctx groupcache.Context, dateStr string, dest groupcache.Sink) error {
	// correct weekend offset, as we miss data for those
	time, err := time.Parse("2006-01-02", dateStr)
	var date string = time.Format("2006-01-02")
	if time.Weekday() == 0 {
		date = time.AddDate(0, 0, -2).Format("2006-01-02")
	} else if time.Weekday() == 6 {
		date = time.AddDate(0, 0, -1).Format("2006-01-02")
	}

	handle, err := os.OpenFile(fmt.Sprintf(`%v/%v.xml`, provider.dataDirectory, date), os.O_RDONLY, 0660)
	if err != nil {
		// we might also miss german holiday informations, try to correct those
		for i := 0; i < 5; i++ {
			date = time.AddDate(0, 0, i*-1).Format("2006-01-02")
			handle, err = os.OpenFile(fmt.Sprintf(`%v/%v.xml`, provider.dataDirectory, date), os.O_RDONLY, 0660)
			if err == nil {
				break
			}
		}
		if err != nil {
			fmt.Printf("unable to open file: %#v", err)
			return nil
		}
	}

	b, e := ioutil.ReadAll(handle)
	handle.Close()

	dest.SetBytes(b)
	return e
}

type listCurrenciesRequest struct {
	provider *fileSystemProvider
	date     string
}

func (req listCurrenciesRequest) Date() string {
	return req.date
}

type lookupCurrencyRequest struct {
	listCurrenciesRequest
	currency string
}

func (request lookupCurrencyRequest) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var cachedData []byte
	request.provider.cache.Get(req, request.date, groupcache.AllocatingByteSliceSink(&cachedData))

	envelop := &data.Envelop{}
	decoder := xml.NewDecoder(bytes.NewReader(cachedData))
	if err := decoder.Decode(&envelop); err != nil {
		fmt.Printf("unable to decode xml")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if envelop == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	cube := envelop.Cubes[0]

	currency := strings.ToUpper(request.currency)
	var exchange data.Exchange
	if currency == "EUR" {
		exchange.Currency = "EUR"
		exchange.Rate = 1.0
	} else {
		for _, ex := range envelop.Exchanges() {
			if ex.Currency == currency {
				exchange = ex
			}
		}
	}

	if exchange == (data.Exchange{}) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	info := exchangeInfo{time.Time(cube.Date).Format("2006-01-02"), shortExchangeInfo{exchange.Currency, exchange.Rate}}
	enc := json.NewEncoder(w)

	if err := enc.Encode(info); err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
}

type exchangeInfoCollection struct {
	Date      string              `json:"date"`
	Exchanges []shortExchangeInfo `json:"exchanges"`
}

func (request listCurrenciesRequest) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var cachedData []byte
	request.provider.cache.Get(req, request.date, groupcache.AllocatingByteSliceSink(&cachedData))

	envelop := data.Envelop{}
	decoder := xml.NewDecoder(bytes.NewReader(cachedData))
	if err := decoder.Decode(&envelop); err != nil {
		fmt.Printf("unable to decode xml")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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

var earliestTimestamp time.Time = time.Date(1999, 1, 4, 0, 0, 0, 0, time.UTC)

type RequestWithDate interface {
	Date() string
}

func ValidateRequestedDate(request RequestWithDate) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		date, err := time.Parse("2006-01-02", request.Date())

		if err != nil {
			bytes, _ := json.Marshal(map[string][]string{
				"errors": []string{"parameter 'date' has an invalid format. please use YYYY-MM-DD."},
			})

			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, string(bytes))
			return
		}

		if date.Before(earliestTimestamp) {
			bytes, _ := json.Marshal(map[string][]string{
				"errors": []string{"Given date is earlier than Jan 4th, 1999. No data available"},
			})

			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, string(bytes))
			return
		}

		if date.After(time.Now()) {
			bytes, _ := json.Marshal(map[string][]string{
				"errors": []string{"Given date is in the future. No data available"},
			})

			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, string(bytes))
			return
		}

		request.(http.Handler).ServeHTTP(w, req)
	}
}

var routingRegexp *regexp.Regexp = regexp.MustCompile(`/(\d{4}-\d{2}-\d{2})/?([A-Za-z]{3})?`)

func NewCurrencyExchangeServer(provider *fileSystemProvider) http.Handler {
	r := http.NewServeMux()

	r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if !routingRegexp.MatchString(req.URL.Path) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// parts consists of three things: [0] => path, [1] => date, [2] => currency
		parts := routingRegexp.FindAllStringSubmatch(req.URL.Path, -1)[0]
		listRequest := listCurrenciesRequest{date: parts[1], provider: provider}

		if parts[2] == "" {
			logHandler(ValidateRequestedDate(listRequest)).ServeHTTP(w, req)
		} else {
			lookupRequest := lookupCurrencyRequest{listCurrenciesRequest: listRequest, currency: parts[2]}
			logHandler(ValidateRequestedDate(lookupRequest)).ServeHTTP(w, req)
		}
	})

	return http.Handler(r)
}

func main() {
	var (
		httpAddress   = flag.String("http.addr", ":8080", "HTTP listen address")
		dataDirectory = flag.String("data", "", "path to data directory")
	)
	flag.Parse()

	if fileInfo, err := os.Stat(*dataDirectory); err != nil {
		log.Fatalf(`unable to stat %v: %v`, *dataDirectory, err)
	} else if !fileInfo.IsDir() {
		log.Fatalf(`%v is no directory`, *dataDirectory)
	}

	groupcache.NewHTTPPool("http://127.0.0.1")
	provider := fileSystemProvider{*dataDirectory, nil}
	provider.cache = groupcache.NewGroup("exchangeRates", 64<<20, groupcache.GetterFunc(provider.getExchangeRateData))

	log.Printf("listening on %s", *httpAddress)
	log.Fatal(http.ListenAndServe(*httpAddress, NewCurrencyExchangeServer(&provider)))
}

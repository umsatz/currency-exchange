package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"sync"
	"time"
)

// these structs reflect the eurofxref xml data structure
type envelop struct {
	Subject string `xml:"subject"`
	Sender  string `xml:"Sender>name"`
	Cubes   []cube `xml:"Cube>Cube"`
}
type cube struct {
	Date      string     `xml:"time,attr"`
	Exchanges []exchange `xml:"Cube"`
}
type exchange struct {
	Currency string  `xml:"currency,attr" json:"currency"`
	Rate     float32 `xml:"rate,attr" json:"rate"`
}

// EUR is not present because all exchange rates are a reference to the EUR
var desiredCurrencies = map[string]struct{}{
	"USD": struct{}{},
	"GBP": struct{}{},
	// "DKK": struct{}{},
	// "JPY": struct{}{},
	// "BGN": struct{}{},
	// "CZK": struct{}{},
	// "HUF": struct{}{},
	// "LTL": struct{}{},
	// "PLN": struct{}{},
	// "RON": struct{}{},
	// "SEK": struct{}{},
	// "CHF": struct{}{},
	// "NOK": struct{}{},
	// "HRK": struct{}{},
	// "RUB": struct{}{},
	// "TRY": struct{}{},
	// "AUD": struct{}{},
	// "BRL": struct{}{},
	// "CAD": struct{}{},
	// "CNY": struct{}{},
	// "HKD": struct{}{},
	// "IDR": struct{}{},
	// "ILS": struct{}{},
	// "INR": struct{}{},
	// "KRW": struct{}{},
	// "MXN": struct{}{},
	// "MYR": struct{}{},
	// "NZD": struct{}{},
	// "PHP": struct{}{},
	// "SGD": struct{}{},
	// "THB": struct{}{},
	// "ZAR": struct{}{},
}

// last 90 days are available at http://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist-90d.xml
var eurHistURL = "http://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist.xml"
var mu *sync.RWMutex
var exchangeRates = map[string][]exchange{}

func downloadExchangeRates() (io.Reader, error) {
	resp, err := http.Get(eurHistURL)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request returned %v", resp.Status)
	}

	return resp.Body, nil
}

func filterExchangeRates(c cube) []exchange {
	var rates []exchange
	for _, ex := range c.Exchanges {
		if _, ok := desiredCurrencies[ex.Currency]; ok {
			rates = append(rates, ex)
		}
	}
	return rates
}

func updateExchangeRates(data io.Reader) error {
	mu.Lock()
	defer mu.Unlock()

	var e envelop
	decoder := xml.NewDecoder(data)
	if err := decoder.Decode(&e); err != nil {
		return err
	}

	for _, c := range e.Cubes {
		if _, ok := exchangeRates[c.Date]; !ok {
			exchangeRates[c.Date] = filterExchangeRates(c)
		}
	}

	runtime.GC()

	return nil
}

func updateExchangeRatesCache() {
	if reader, err := downloadExchangeRates(); err != nil {
		fmt.Printf("Unable to download exchange rates. Is the URL correct?")
	} else {
		if err := updateExchangeRates(reader); err != nil {
			fmt.Printf("Failed to update exchange rates: %v", err)
		}
	}
}

func populateExchangeRateCache(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}

	return updateExchangeRates(f)
}

func exchangeRatesByCurrency(rates []exchange) map[string]float32 {
	var mappedByCurrency = make(map[string]float32)
	for _, rate := range rates {
		mappedByCurrency[rate.Currency] = rate.Rate
	}
	return mappedByCurrency
}

var (
	// accept strings like /1986-09-03 and /1986-09-03/USD
	routingRegexp      = regexp.MustCompile(`/(\d{4}-\d{2}-\d{2})/?$`)
	errRouting         = fmt.Errorf("Routing mismatch. Expected %v", routingRegexp)
	errNoDataAvailable = fmt.Errorf("No data available for requested date")
)

type exchangeResponse struct {
	Date  string             `json:"date"`
	Rates map[string]float32 `json:"rates"`
}

func logHandler(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf(
			"%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			time.Since(start),
		)
	}
}

var cacheControl = (30 * 24 * time.Hour) / time.Second

type httpFunc func(w http.ResponseWriter, req *http.Request) (int, error)

func (f httpFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if status, err := f(w, r); err != nil {
		switch status {
		case http.StatusBadRequest:
			http.Error(w, err.Error(), http.StatusBadRequest)
		case http.StatusNotFound:
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func serveExchangeRates(w http.ResponseWriter, req *http.Request) (int, error) {
	if !routingRegexp.MatchString(req.URL.Path) {
		return http.StatusBadRequest, errRouting
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", cacheControl))

	parts := routingRegexp.FindAllStringSubmatch(req.URL.Path, -1)[0]
	requestedDate := parts[1]

	// force clients to pass valid dates in correct format
	date, err := time.Parse("2006-01-02", requestedDate)
	if err != nil {
		return http.StatusBadRequest, err
	}

	mu.RLock()
	defer mu.RUnlock()

	servedDate := requestedDate
	for {
		// ensure the request is in a valid timespan
		if date.Year() < 1999 {
			return http.StatusBadRequest, fmt.Errorf("%q < 1999-01-01. No data available", date.Format("2006-01-02"))
		}

		// look at previous day to skip weekend and holiday gaps
		if _, ok := exchangeRates[servedDate]; !ok {
			date = date.Add(-24 * time.Hour)
			servedDate = date.Format("2006-01-02")
		} else {
			break
		}
	}

	if _, ok := exchangeRates[servedDate]; !ok {
		return http.StatusNotFound, errNoDataAvailable
	}

	var exs = exchangeRates[servedDate]
	var resp = exchangeResponse{
		Date:  servedDate,
		Rates: exchangeRatesByCurrency(exs),
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func newCurrencyExchangeServer() http.Handler {
	r := http.NewServeMux()

	r.Handle("/", httpFunc(serveExchangeRates))

	return http.Handler(r)
}

func updateExchangeRatesPeriodically() {
	for {
		time.Sleep(1 * time.Hour)

		updateExchangeRatesCache()
	}
}

// Set by make file on build
var (
	Version string
	Commit  string
)

func init() {
	mu = &sync.RWMutex{}
}

func main() {
	var (
		httpAddress  = flag.String("http.addr", ":8080", "HTTP listen address")
		historicData = flag.String("historic.data", "", "path to data directory")
		printVersion = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *printVersion {
		fmt.Printf("%s", Version)
		os.Exit(0)
	}

	if err := populateExchangeRateCache(*historicData); err != nil {
		fmt.Printf("Unable to populate cache: %v", err)
		os.Exit(-1)
	}
	go updateExchangeRatesPeriodically()

	log.Printf("listening on %v", *httpAddress)
	log.Fatal(http.ListenAndServe(*httpAddress, logHandler(newCurrencyExchangeServer())))
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/umsatz/currency-exchange/ecb"
)

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
var exchangeRates = map[string][]ecb.Exchange{}

func filterExchangeRates(c ecb.Cube, currencies map[string]struct{}) []ecb.Exchange {
	var rates []ecb.Exchange
	for _, ex := range c.Exchanges {
		if _, ok := currencies[ex.Currency]; ok {
			rates = append(rates, ex)
		}
	}
	return rates
}

func updateExchangeRates(data io.Reader) error {
	mu.Lock()
	defer mu.Unlock()

	cubes, _ := ecb.Parse(data)

	for _, c := range cubes {
		// because we're using a historic data source the data never changes once it's read
		if _, ok := exchangeRates[c.Date]; !ok {
			exchangeRates[c.Date] = filterExchangeRates(c, desiredCurrencies)
		}
	}

	return nil
}

func updateExchangeRatesCache() {
	resp, err := http.Get(eurHistURL)

	if err != nil {
		log.Printf("Unable to download exchange rates: %q", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP request returned %q", resp.Status)
		return
	}
	defer resp.Body.Close()

	if err := updateExchangeRates(resp.Body); err != nil {
		log.Printf("Failed to update exchange rates: %v", err)
	}
}

func populateExchangeRateCache(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}

	return updateExchangeRates(f)
}

func exchangeRatesByCurrency(rates []ecb.Exchange) map[string]float32 {
	var mappedByCurrency = make(map[string]float32)
	for _, rate := range rates {
		mappedByCurrency[rate.Currency] = rate.Rate
	}
	return mappedByCurrency
}

var (
	errRouting         = fmt.Errorf("Routing mismatch. Must be a date of form YYYY-MM-DD")
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
			"%s\t%s\t%s\t%s\t%s",
			r.RemoteAddr,
			time.Now().Format("2006-01-02T15:04:05 -0700"),
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
	requestedDate := req.URL.Path[1:]
	if strings.HasSuffix(requestedDate, "/") {
		requestedDate = requestedDate[:len(requestedDate)-1]
	}
	// force clients to pass valid dates in correct format
	date, err := time.Parse("2006-01-02", requestedDate)
	if err != nil {
		return http.StatusBadRequest, err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", cacheControl))

	mu.RLock()
	defer mu.RUnlock()

	servedDate := requestedDate
	for {
		// ensure the request is in a valid timespan
		if date.Year() < 1999 {
			return http.StatusBadRequest, fmt.Errorf("%q < 1999-01-01. No data available.", date.Format("2006-01-02"))
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

// Set by make file on build
var (
	Version string
	Commit  string
)

func init() {
	mu = &sync.RWMutex{}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
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
	go func() {
		for {
			time.Sleep(6 * time.Hour)

			updateExchangeRatesCache()
		}
	}()

	log.Printf("listening on %v", *httpAddress)
	log.Fatal(http.ListenAndServe(*httpAddress, logHandler(newCurrencyExchangeServer())))
}

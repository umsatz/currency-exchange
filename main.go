package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/umsatz/currency-exchange/ecb"
	rateHTTP "github.com/umsatz/currency-exchange/http"
)

var (
	// EUR is not present because all exchange rates are a reference to the EUR
	desiredCurrencies = map[string]struct{}{
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

	// last 90 days are available at https://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist-90d.xml
	eurHistURL = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist.xml"
)

func filterExchangeRates(c ecb.Cube, currencies map[string]struct{}) []ecb.Exchange {
	var rates []ecb.Exchange
	for _, ex := range c.Exchanges {
		if _, ok := currencies[ex.Currency]; ok {
			rates = append(rates, ex)
		}
	}
	return rates
}

func updateExchangeRates(data io.Reader) ([]ecb.Cube, error) {
	cubes, err := ecb.Parse(data)
	if err != nil {
		return nil, err
	}

	for i, c := range cubes {
		cubes[i].Exchanges = filterExchangeRates(c, desiredCurrencies)
	}

	return cubes, nil
}

func updateExchangeRatesCache(cache *rateCache) {
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

	cubes, err := updateExchangeRates(resp.Body)
	if err != nil {
		log.Printf("Failed to update exchange rates: %v", err)
	}
	cache.update(cubes)
}

func populateExchangeRateCache(file string) (*rateCache, error) {
	cache := rateCache{
		exchangeRates: map[string][]ecb.Exchange{},
	}

	f, err := os.Open(file)
	if err != nil {
		return &cache, err
	}

	cs, err := updateExchangeRates(f)
	if err != nil {
		return &cache, err
	}

	cache.update(cs)
	return &cache, nil
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

// Set by make file on build
var (
	Version string
	Commit  string
)

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

	cache, err := populateExchangeRateCache(*historicData)
	if err != nil {
		log.Fatalf("Unable to populate cache: %v", err)
	}

	go func() {
		for {
			time.Sleep(6 * time.Hour)

			updateExchangeRatesCache(cache)
		}
	}()

	log.Printf("listening on %v", *httpAddress)
	log.Fatal(http.ListenAndServe(*httpAddress, logHandler(http.Handler(rateHTTP.Handler(cache)))))
}

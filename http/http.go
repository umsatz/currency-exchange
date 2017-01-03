package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/umsatz/currency-exchange/ecb"
)

type Lookup interface {
	Rates(date time.Time) (ecb.Cube, error)
}

var (
	cacheControlDuration = (30 * 24 * time.Hour) / time.Second
	cacheControlHeader   = fmt.Sprintf("max-age=%d", cacheControlDuration)
)

func exchangeRatesByCurrency(rates []ecb.Exchange) map[string]float32 {
	var mappedByCurrency = make(map[string]float32)
	for _, rate := range rates {
		mappedByCurrency[rate.Currency] = rate.Rate
	}
	return mappedByCurrency
}

type exchangeResponse struct {
	Date  string             `json:"date"`
	Rates map[string]float32 `json:"rates"`
}

func Handler(l Lookup) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		requestedDate := req.URL.Path[1:]
		if strings.HasSuffix(requestedDate, "/") {
			requestedDate = requestedDate[:len(requestedDate)-1]
		}

		// force clients to pass valid dates in correct format
		date, err := time.Parse("2006-01-02", requestedDate)
		if err != nil {
			http.Error(w, "Routing mismatch. Must be a date of form YYYY-MM-DD", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", cacheControlHeader)

		cube, err := l.Rates(date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		var resp = exchangeResponse{
			Date:  cube.Date,
			Rates: exchangeRatesByCurrency(cube.Exchanges),
		}

		enc := json.NewEncoder(w)
		if err := enc.Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

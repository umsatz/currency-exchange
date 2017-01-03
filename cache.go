package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/umsatz/currency-exchange/ecb"
)

var errNoDataAvailable = fmt.Errorf("No data available for requested date")

type rateCache struct {
	mu            sync.RWMutex
	exchangeRates map[string][]ecb.Exchange
}

func (c *rateCache) update(cs []ecb.Cube) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cube := range cs {
		// because we're using a historic data source the data never changes once it's read
		if _, ok := c.exchangeRates[cube.Date]; !ok {
			c.exchangeRates[cube.Date] = cube.Exchanges
		}
	}
}

func (c *rateCache) Rates(requestedDate time.Time) (ecb.Cube, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	date := requestedDate
	servedDate := requestedDate.Format("2006-01-02")
	for {
		// ensure the request is in a valid timespan
		if date.Year() < 1999 {
			return ecb.Cube{}, fmt.Errorf("%q < 1999-01-01. No data available.", date.Format("2006-01-02"))
		}

		// look at previous day to skip weekend and holiday gaps
		if _, ok := c.exchangeRates[servedDate]; !ok {
			date = date.Add(-24 * time.Hour)
			servedDate = date.Format("2006-01-02")
		} else {
			break
		}
	}

	if _, ok := c.exchangeRates[servedDate]; !ok {
		return ecb.Cube{}, errNoDataAvailable
	}

	return ecb.Cube{
		Date:      servedDate,
		Exchanges: c.exchangeRates[servedDate],
	}, nil
}

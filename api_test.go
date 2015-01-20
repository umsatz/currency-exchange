package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type testServer struct {
	server *httptest.Server
}

var TestServer testServer

func TestMain(m *testing.M) {
	populateExchangeRateCache("data/eurofxref-hist.xml")
	TestServer = testServer{
		server: httptest.NewServer(newCurrencyExchangeServer()),
	}
	ret := m.Run()
	os.Exit(ret)
}

type requestExpectation struct {
	date          string
	expectedDate  string
	expectedRates map[string]float32
}

func TestListCurrencyExchange(t *testing.T) {
	t.Parallel()

	tests := []requestExpectation{
		{
			// regular dates are returned correctly
			date:         "2010-07-14",
			expectedDate: "2010-07-14",
			expectedRates: map[string]float32{
				"USD": 1.2703,
				"GBP": 0.8343,
			},
		}, {
			// requesting sundays returns fridays data
			date:         "2010-07-11",
			expectedDate: "2010-07-09",
			expectedRates: map[string]float32{
				"USD": 1.2637,
				"GBP": 0.836,
			},
		}, {
			// short holidays are skipped correctly
			date:         "2013-11-31",
			expectedDate: "2013-11-29",
			expectedRates: map[string]float32{
				"USD": 1.3611,
				"GBP": 0.83275,
			},
		}, {
			// longer holidays are skipped correctly
			date:         "2001-04-16",
			expectedDate: "2001-04-12",
			expectedRates: map[string]float32{
				"USD": 0.8849,
				"GBP": 0.6173,
			},
		},
	}

	testListCurrency := func(test requestExpectation, t *testing.T) {
		resp, _ := http.Get(TestServer.server.URL + "/" + test.date)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("resp body did not contain expected %v:\n\t", "200", resp.StatusCode)
		}

		decoder := json.NewDecoder(resp.Body)
		var data exchangeResponse
		if err := decoder.Decode(&data); err != nil {
			t.Fatalf("Unable to decode json response: %#v", err)
		}

		if data.Date != test.expectedDate {
			t.Fatalf("did no respond with correct date: %#v, expected: %#v", data.Date, test.expectedDate)
		}

		if len(data.Rates) < 1 {
			t.Fatalf("did not respond with correct data")
		}

		for k, v := range test.expectedRates {
			var exists = false
			for gk, gv := range data.Rates {
				exists = exists || gk == k && gv == v
			}
			if !exists {
				t.Fatalf("Expected %v response to contain %v with %v\n", data.Date, k, v)
			}
		}
	}

	for _, test := range tests {
		testListCurrency(test, t)
	}
}

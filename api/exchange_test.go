package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestLookupCurrencyExchangeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	t.Parallel()

	provider := fileSystemProvider{"../data"}

	ts := httptest.NewServer(NewCurrencyExchangeServer(&provider))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/2010-07-14/USD")
	if err != nil {
		t.Fatalf("err: %#v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Response body did not contain expected %v:\n\tcode: %v", "200", res.StatusCode)
	}

	decoder := json.NewDecoder(res.Body)
	var data exchangeInfo
	if err := decoder.Decode(&data); err != nil {
		t.Fatalf("Unable to decode json response: %#v", err)
	}
}

type lookupCurrencyTest struct {
	title        string // title of the test
	date         string // requested date
	currency     string // requested currency
	expectedDate string // expected date
}

func TestLookupCurrencyExchange(t *testing.T) {
	t.Parallel()

	tests := []lookupCurrencyTest{
		{
			title:        "regular match on weekday",
			date:         "2010-07-14",
			currency:     "USD",
			expectedDate: "2010-07-14",
		},
	}

	testLookupCurrency := func(test lookupCurrencyTest, t *testing.T) {
		request, _ := http.NewRequest("GET", fmt.Sprintf("/%s/%s", test.date, test.currency), nil)
		response := httptest.NewRecorder()

		provider := fileSystemProvider{"../data"}
		provider.LookupCurrencyExchange(response, request, map[string]string{"date": test.date, "currency": test.currency})

		if response.Code != http.StatusOK {
			t.Fatalf("Response body did not contain expected %v:\n\tcode: %v", "200", response.Code)
		}

		decoder := json.NewDecoder(response.Body)
		var data exchangeInfo
		if err := decoder.Decode(&data); err != nil {
			t.Fatalf("Unable to decode json response: %#v", err)
		}

		if data.Date != test.expectedDate {
			t.Fatalf("did no respond with correct date: %#v, expected: %#v", data.Date, test.expectedDate)
		}
		if data.Currency != test.currency {
			t.Fatalf("did no respond with correct currency: %#v, expected: %#v", data, test.currency)
		}
	}

	for _, test := range tests {
		testLookupCurrency(test, t)
	}
}

func TestListCurrencyExchangeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	t.Parallel()

	provider := fileSystemProvider{"../data"}

	ts := httptest.NewServer(NewCurrencyExchangeServer(&provider))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/2010-07-14")
	if err != nil {
		t.Fatalf("err: %#v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Response body did not contain expected %v:\n\tcode: %v", "200", res.StatusCode)
	}

	decoder := json.NewDecoder(res.Body)
	var data exchangeInfoCollection
	if err := decoder.Decode(&data); err != nil {
		t.Fatalf("Unable to decode json response: %#v", err)
	}
}

type listCurrencyTest struct {
	title        string // title of the test
	date         string // requested date
	expectedDate string // expected date
}

func TestListCurrencyExchange(t *testing.T) {
	t.Parallel()

	tests := []listCurrencyTest{
		{
			title:        "regular match on weekday",
			date:         "2010-07-14",
			expectedDate: "2010-07-14",
		}, {
			title:        "sunday returns friday",
			date:         "2010-07-11",
			expectedDate: "2010-07-09",
		}, {
			title:        "holidays return pre-holiday",
			date:         "2013-11-31",
			expectedDate: "2013-11-29",
		},
	}

	testListCurrency := func(test listCurrencyTest, t *testing.T) {
		request, _ := http.NewRequest("GET", "/"+test.date, nil)
		response := httptest.NewRecorder()

		provider := fileSystemProvider{"../data"}
		provider.ListCurrencyExchange(response, request, map[string]string{"date": test.date})

		if response.Code != http.StatusOK {
			t.Fatalf("Response body did not contain expected %v:\n\t", "200")
		}

		decoder := json.NewDecoder(response.Body)
		var data exchangeInfoCollection
		if err := decoder.Decode(&data); err != nil {
			t.Fatalf("Unable to decode json response: %#v", err)
		}

		if data.Date != test.expectedDate {
			t.Fatalf("did no respond with correct date: %#v, expected: %#v", data.Date, test.expectedDate)
		}

		if len(data.Exchanges) < 1 {
			t.Fatalf("did not respond with correct data")
		}
	}

	for _, test := range tests {
		testListCurrency(test, t)
	}
}

func TestValidateRequestedDateIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	t.Parallel()

	provider := fileSystemProvider{"../data"}

	ts := httptest.NewServer(NewCurrencyExchangeServer(&provider))
	defer ts.Close()

	if res, _ := http.Get(ts.URL + "/1998-12-20"); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("Response body did not contain expected %v:\n\tcode: %v", "200", res.StatusCode)
	}

	if res, _ := http.Get(ts.URL + "/2100-01-01"); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("Response body did not contain expected %v:\n\tcode: %v", "200", res.StatusCode)
	}

	if res, _ := http.Get(ts.URL + "/2100-o1-01"); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("Response body did not contain expected %v:\n\tcode: %v", "200", res.StatusCode)
	}
}

type validateRequestDateTest struct {
	title                string // title of the test
	date                 string // requested date
	expectedResponseCode int
}

func TestValidateRequestedDate(t *testing.T) {
	t.Parallel()

	tests := []validateRequestDateTest{
		{
			title:                "date prior to 1999",
			date:                 "1998-12-20",
			expectedResponseCode: http.StatusBadRequest,
		}, {
			title:                "date in the future",
			date:                 "2100-01-01",
			expectedResponseCode: http.StatusBadRequest,
		}, {
			title:                "date is malformatted",
			date:                 "2100-o1-01",
			expectedResponseCode: http.StatusBadRequest,
		},
	}

	testValidateRequestedDate := func(test validateRequestDateTest, t *testing.T) {
		r, _ := http.NewRequest("GET", "http://localhost:3000/"+test.date, nil)
		w := httptest.NewRecorder()

		m := mux.NewRouter()
		provider := fileSystemProvider{"../data"}
		m.HandleFunc("/{date}", ValidateRequestedDate(VarsHandler(provider.ListCurrencyExchange)))
		m.ServeHTTP(w, r)

		if w.Code != test.expectedResponseCode {
			t.Fatalf("unexpeted response code. expected %v got %v", 400, test.expectedResponseCode)
		}
	}

	for _, test := range tests {
		testValidateRequestedDate(test, t)
	}
}

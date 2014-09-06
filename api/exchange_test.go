package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type currencyListTest struct {
	title        string // title of the test
	date         string // requested date
	expectedDate string // expected date
}

func testListCurrency(test currencyListTest, t *testing.T) {
	request, _ := http.NewRequest("GET", "/"+test.date, nil)
	response := httptest.NewRecorder()

	provider := fileSystemProvider{"../data"}
	provider.ListCurrencyExchange(response, request, map[string]string{"date": test.date})

	if response.Code != http.StatusOK {
		t.Fatalf("Response body did not contain expected %v:\n\tbody: %v", "200", response.Code)
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

func TestListCurrencyExchange(t *testing.T) {
	tests := []currencyListTest{
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

	for _, test := range tests {
		testListCurrency(test, t)
	}
}

// func TestLookupCurrencyExchange(t *testing.T) {
// 	t.Fatalf("fail")
//}

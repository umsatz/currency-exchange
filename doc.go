/*
currency-exchange provides a small HTTP interface to request exchange rates based
on historic data provided by the ECB.

API

GET /{date} - Provide a date you want exchange rates for

  $ curl -s 'http://localhost:8080/2014-02-02'
  {
      "date": "2014-01-31",
      "rates": {
          "GBP": 0.82135,
          "USD": 1.3516
      }
  }

DESIGN

currency-exchange is set up to refresh the cache every hour using the configured
data source. Because the data is so small the cache is never cleared.

If exchange informations are requested for days where no data is available currency-exchange
falls back to the previous available date. The previous date might be some days prior
which is why the response contains the date of the data actually used.
*/
package main

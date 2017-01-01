# currency-exchange

[![Build Status](https://circleci.com/gh/umsatz/currency-exchange/tree/master.svg?style=svg)](https://circleci.com/gh/umsatz/currency-exchange)

api for past and present currency exchange rates

The API requires you to import euro exchange informations as available from the ECB.
See the [Makefile][1] for more details.

# currency-exchange api

Actual JSON api for currency exchange rates.

To run:

```
$ exchange -historic.data=/path/euroxml-hist.xml -http.addr=:8080
```

```
  curl http://localhost:8080/rates/2013-02-20
  {
    "date": "2010-07-14",
    "rates": {
        "GBP": 0.8343,
        "USD": 1.2703
    }
  }
```

[1]:Makefile

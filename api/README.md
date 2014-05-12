# currency-exchange api

Actual JSON api for currency exchange rates.

To run:

```
$ exchange -data=/path/to/imported/data
```

```
  curl http://localhost:8080/rates/2013-02-20/USD
  {
    "date": "2013-02-20",
    "currency": "USD",
    "rate": 1.337
  }
```

```
  curl http://localhost:8080/rates/2013-02-20
  {
    "date": "2013-02-20",
    "exchanges": [
        {
            "currency": "USD",
            "rate": 1.337
        },
        {
            "currency": "JPY",
            "rate": 125.09
        },
        {
            "currency": "BGN",
            "rate": 1.9558
        },
        {
            "currency": "CZK",
            "rate": 25.393
        },
        {
            "currency": "DKK",
            "rate": 7.4604
        },
        {
            "currency": "GBP",
            "rate": 0.8733
        },
        {
            "currency": "HUF",
            "rate": 291.31
        },
        {
            "currency": "LTL",
            "rate": 3.4528
        },
        {
            "currency": "LVL",
            "rate": 0.6997
        },
        {
            "currency": "PLN",
            "rate": 4.1569
        },
        {
            "currency": "RON",
            "rate": 4.3785
        },
        {
            "currency": "SEK",
            "rate": 8.4297
        },
        {
            "currency": "CHF",
            "rate": 1.2347
        },
        {
            "currency": "NOK",
            "rate": 7.4065
        },
        {
            "currency": "HRK",
            "rate": 7.5915
        },
        {
            "currency": "RUB",
            "rate": 40.2302
        },
        {
            "currency": "TRY",
            "rate": 2.3792
        },
        {
            "currency": "AUD",
            "rate": 1.2961
        },
        {
            "currency": "BRL",
            "rate": 2.6145
        },
        {
            "currency": "CAD",
            "rate": 1.3567
        },
        {
            "currency": "CNY",
            "rate": 8.3401
        },
        {
            "currency": "HKD",
            "rate": 10.3678
        },
        {
            "currency": "IDR",
            "rate": 12940.94
        },
        {
            "currency": "ILS",
            "rate": 4.8978
        },
        {
            "currency": "INR",
            "rate": 72.301
        },
        {
            "currency": "KRW",
            "rate": 1442.83
        },
        {
            "currency": "MXN",
            "rate": 16.9117
        },
        {
            "currency": "MYR",
            "rate": 4.14
        },
        {
            "currency": "NZD",
            "rate": 1.5972
        },
        {
            "currency": "PHP",
            "rate": 54.426
        },
        {
            "currency": "SGD",
            "rate": 1.654
        },
        {
            "currency": "THB",
            "rate": 39.869
        },
        {
            "currency": "ZAR",
            "rate": 11.8659
        }
    ]
  }
```
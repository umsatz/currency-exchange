# currency-exchange

api for past and present currency exchange rates

The API requires you to import euro exchange informations as available from the ECB.
See the [Makefile][1] for more details.

# currency-exchange api

Actual JSON api for currency exchange rates.

To run:

```
$ exchange -data=/path/euroxml-hist -http.addr=:8080
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
        ...
        {
            "currency": "ZAR",
            "rate": 11.8659
        }
    ]
  }
```

[1]:Makefile

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
        ...
        {
            "currency": "ZAR",
            "rate": 11.8659
        }
    ]
  }
```
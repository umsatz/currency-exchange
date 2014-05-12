# currency-exchange api

Actual JSON api for currency exchange rates.

To run:

```
$ exchange -data=/path/to/imported/data
```

```
  curl http://localhost:8080/2013-02-01/USD
  {
    "currency":"USD",
    "rate":1.3644
  }
```
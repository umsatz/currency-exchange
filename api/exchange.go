package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	. "github.com/umsatz/currency-exchange/data"
)

func logHandler(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%v %v", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	}
}

func LookupCurrencyExchange(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	vars := mux.Vars(req)

	handle, err := os.OpenFile(fmt.Sprintf(`./data/%v.xml`, vars["date"]), os.O_RDONLY, 0660)
	if err != nil {
		fmt.Printf("unable to open file: %#v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer handle.Close()

	envelop := Envelop{}
	decoder := xml.NewDecoder(handle)
	if err := decoder.Decode(&envelop); err != nil {
		fmt.Printf("unable to decode xml")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	currency := vars["currency"]
	cube := envelop.Cubes[0]
	var exchange Exchange
	for _, ex := range cube.Exchanges {
		if ex.Currency == currency {
			exchange = ex
		}
	}

	if exchange == (Exchange{}) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	bytes, err := json.Marshal(exchange)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	io.WriteString(w, string(bytes))
}

func main() {
	var port string = os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	l, err := net.Listen("tcp", "0.0.0.0:"+port)
	if nil != err {
		log.Fatalln(err)
	}
	log.Println("listening on %v", l.Addr())

	r := mux.NewRouter()
	r.Handle("/{date}/{currency}", logHandler(http.HandlerFunc(LookupCurrencyExchange))).Methods("GET")
	http.Serve(l, r)
}

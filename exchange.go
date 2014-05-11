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
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type ShortDate time.Time

func (date *ShortDate) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	dateStr := ""
	if time.Time(*date) != (time.Time{}) {
		dateStr = time.Time(*date).Format("2006-01-02")
	}
	attr := xml.Attr{name, dateStr}
	return attr, nil
}

func (date *ShortDate) UnmarshalXMLAttr(attr xml.Attr) error {
	time, err := time.Parse("2006-01-02", attr.Value)
	if err != nil {
		date = &ShortDate{}
		err = nil
	} else {
		*date = ShortDate(time)
	}
	return err
}

type Envelop struct {
	Subject string `xml:"subject"`
	Sender  string `xml:"Sender>name"`
	Cubes   []Cube `xml:"Cube>Cube"`
}

type Cube struct {
	Date      ShortDate  `xml:"time,attr"`
	Exchanges []Exchange `xml:"Cube"`
}

type Exchange struct {
	Currency string  `xml:"currency,attr" json:"currency"`
	Rate     float32 `xml:"rate,attr" json:"rate"`
}

func writeEnvelop(env *Envelop) error {
	dataFileName := fmt.Sprintf("./data/%v.xml", time.Time(env.Cubes[0].Date).Format("2006-01-02"))
	file, err := os.OpenFile(dataFileName, os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := xml.NewEncoder(file)
	encoder.Encode(env)
	return nil
}

// TODO move PrepareEnvelops into separate app, schedule every night
func PrepareEnvelops(histPath string) error {
	handle, err := os.Open(histPath)
	if err != nil {
		return err
	}
	defer handle.Close()

	envelop := Envelop{}
	decoder := xml.NewDecoder(handle)
	if err := decoder.Decode(&envelop); err != nil {
		return err
	}

	envelops := make(chan *Envelop, 10)
	var wait sync.WaitGroup
	for i := 0; i < 4; i++ {
		wait.Add(1)
		go func() {
			for env := range envelops {
				writeEnvelop(env)
			}
			wait.Done()
		}()
	}

	for _, cube := range envelop.Cubes {
		envelops <- &Envelop{envelop.Subject, envelop.Sender, []Cube{cube}}
	}
	close(envelops)
	wait.Wait()

	return nil
}

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
	// if err := PrepareEnvelops("./data/eurofxref-hist.xml"); err != nil {
	// 	fmt.Printf(`%#v\n`, err)
	// }

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

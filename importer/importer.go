package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	. "github.com/umsatz/currency-exchange/data"
)

var historyFile string
var outputDirectory string

func WriteXML(env *Envelop) error {
	// TODO do not over-write existing files
	dataFileName := fmt.Sprintf("%v/%v.xml", outputDirectory, time.Time(env.Cubes[0].Date).Format("2006-01-02"))
	file, err := os.OpenFile(dataFileName, os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := xml.NewEncoder(file)
	encoder.Encode(env)
	return nil
}

func init() {
	flag.StringVar(&historyFile, "history", "", "path to eurofxref-hist.xml")
	flag.StringVar(&outputDirectory, "out", "", "path to output directory")
}

func main() {
	flag.Parse()

	handle, err := os.Open(historyFile)
	if err != nil {
		log.Fatalf(`unable to open history file: %#v`, err)
	}
	defer handle.Close()

	envelop := Envelop{}
	decoder := xml.NewDecoder(handle)
	if err := decoder.Decode(&envelop); err != nil {
		log.Fatalf(`unable to decode history file: %#v`, err)
	}

	envelops := make(chan *Envelop, 10)
	var wait sync.WaitGroup
	for i := 0; i < 4; i++ {
		wait.Add(1)
		go func() {
			for env := range envelops {
				WriteXML(env)
			}
			wait.Done()
		}()
	}

	importCount := len(envelop.Cubes)

	for _, cube := range envelop.Cubes {
		envelops <- &Envelop{envelop.Subject, envelop.Sender, []Cube{cube}}
	}
	close(envelops)

	wait.Wait()

	fmt.Printf(`%#v days imported`, importCount)
}

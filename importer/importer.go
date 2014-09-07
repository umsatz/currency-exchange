package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/umsatz/currency-exchange/data"
)

var historyFile string
var outputDirectory string
var importCount int = 0

func WriteXML(env *data.Envelop) error {
	dataFileName := fmt.Sprintf("%v/%v.xml", outputDirectory, time.Time(env.Cubes[0].Date).Format("2006-01-02"))

	if _, err := os.Stat(dataFileName); err != nil {
		file, err := os.OpenFile(dataFileName, os.O_RDWR|os.O_CREATE, 0660)
		if err != nil {
			return err
		}
		defer file.Close()

		encoder := xml.NewEncoder(file)
		encoder.Encode(env)

		importCount += 1
	}
	return nil
}

func init() {
	flag.StringVar(&historyFile, "history", "", "path to eurofxref-hist.xml")
	flag.StringVar(&outputDirectory, "out", "", "path to output directory")
	flag.Parse()

	if fileInfo, err := os.Stat(historyFile); err != nil {
		log.Fatalf(`unable to stat %v: %v`, historyFile, err)
	} else if fileInfo.IsDir() {
		log.Fatalf(`%v is directory - should be an xml file`, historyFile)
	}

	if fileInfo, err := os.Stat(outputDirectory); err != nil {
		os.Mkdir(outputDirectory, os.ModeDir)
	} else if !fileInfo.IsDir() {
		log.Fatalf(`%v is no directory`, outputDirectory)
	}
}

func main() {
	handle, err := os.Open(historyFile)
	if err != nil {
		log.Fatalf(`unable to open history file: %#v`, err)
	}
	defer handle.Close()

	envelop := data.Envelop{}
	decoder := xml.NewDecoder(handle)
	if err := decoder.Decode(&envelop); err != nil {
		log.Fatalf(`unable to decode history file: %#v`, err)
	}

	envelops := make(chan *data.Envelop, 10)
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

	for _, cube := range envelop.Cubes {
		envelops <- &data.Envelop{envelop.Subject, envelop.Sender, []data.Cube{cube}}
	}
	close(envelops)

	wait.Wait()

	fmt.Printf(`%#v days imported`, importCount)
}

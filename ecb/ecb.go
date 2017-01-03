// package ecb handles parsing of the eurofxref XML data
package ecb

import (
	"encoding/xml"
	"io"
)

type envelop struct {
	Subject string `xml:"subject"`
	Sender  string `xml:"Sender>name"`
	Cubes   []Cube `xml:"Cube>Cube"`
}

// Cube contains exchange information on a given date
type Cube struct {
	Date      string     `xml:"time,attr"`
	Exchanges []Exchange `xml:"Cube"`
}

// Exchange describes a specific EUR-<currency> rate on a given date
type Exchange struct {
	Currency string  `xml:"currency,attr" json:"currency"`
	Rate     float32 `xml:"rate,attr" json:"rate"`
}

// Parse extracts all currency exchange informations from a XML data source
func Parse(r io.Reader) ([]Cube, error) {
	var e envelop
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&e); err != nil {
		return nil, err
	}
	return e.Cubes, nil
}

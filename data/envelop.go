package data

import (
	"encoding/xml"
	"time"
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

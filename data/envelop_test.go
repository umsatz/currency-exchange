package data

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

func TestXMLDeserialization(t *testing.T) {
	xmlContent := `<?xml version="1.0" ?>
    <Envelop>
      <subject>Reference rates</subject>
      <Sender>
        <name>European Central Bank</name>
      </Sender>
      <Cube>
        <Cube time="2014-05-12">
          <Cube currency="USD" rate="1.3765"/>
          <Cube currency="DKK" rate="7.4433"/>
        </Cube>
      </Cube>
    </Envelop>`

	envelop := Envelop{}

	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	if err := decoder.Decode(&envelop); err != nil {
		t.Fatalf("unable to decode xml!")
	}

	if envelop.Subject != "Reference rates" {
		t.Fatalf("decoding error: wrong subject")
	}

	if envelop.Sender != "European Central Bank" {
		t.Fatalf("decoding error: wrong sender")
	}

	if len(envelop.Cubes) != 1 {
		t.Fatalf("decoding error: wrong number of cubes")
	}

	cube := envelop.Cubes[0]
	date, err := time.Parse("2006-01-02", "2014-05-12")
	if time.Time(cube.Date) != date || err != nil {
		t.Fatalf("decoding error: wrong date")
	}

	if len(envelop.Exchanges()) != 2 {
		t.Fatalf("decoding error: wrong number of exchanges")
	}

	exchange := envelop.Exchanges()[0]
	if exchange.Currency != "USD" {
		t.Fatalf("did not decode exchange properly")
	}
}

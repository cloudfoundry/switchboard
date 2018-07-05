package monit_client

import (
	"encoding/xml"
	"fmt"
	"io"

	"golang.org/x/net/html/charset"
)

const (
	CHARSET_ENCODING = "ISO-8859-1"
)

type MonitStatus struct {
	XMLName  xml.Name     `xml:"monit"`
	Services []ServiceTag `xml:"service"`
}

type ServiceTag struct {
	XMLName       xml.Name `xml:"service"`
	Name          string   `xml:"name"`
	Status        int      `xml:"status"`
	Monitor       int      `xml:"monitor"`
	PendingAction int      `xml:"pendingaction"`
}

func ParseXML(xmlReader io.Reader) (MonitStatus, error) {
	result := MonitStatus{}
	decoder := xml.NewDecoder(xmlReader)

	decoder.CharsetReader = func(characterSet string, xmlReader io.Reader) (io.Reader, error) {
		return charset.NewReader(xmlReader, CHARSET_ENCODING)
	}
	err := decoder.Decode(&result)

	if err != nil {
		err := fmt.Errorf("Failed to unmarshal the xml with error %s",
			err.Error(),
		)
		return result, err
	}

	return result, nil
}

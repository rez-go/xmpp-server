package xmppvcard

import (
	"encoding/xml"
)

// XEP-0054

const (
	NS          = "vcard-temp"
	ElementName = NS + " vCard"
)

type IQGet struct {
	XMLName xml.Name `xml:"vcard-temp vCard"`
}

type IQSet struct {
	XMLName xml.Name `xml:"vcard-temp vCard"`
	Data    []byte   `xml:",innerxml"`
}

type IQResult struct {
	XMLName  xml.Name `xml:"vcard-temp vCard"`
	FullName string   `xml:"FN,omitempty"`
}

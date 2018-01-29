package xmppcore

import (
	"encoding/xml"
)

//TODO: xml:lang

const ClientIQElementName = JabberClientNS + " iq"

// Standard IQ types
//
// RFC 6120  8.2.3
const (
	IQTypeGet    = "get"
	IQTypeSet    = "set"
	IQTypeResult = "result"
	IQTypeError  = "error"
)

type ClientIQ struct {
	XMLName xml.Name     `xml:"jabber:client iq"`
	ID      string       `xml:"id,attr,omitempty"`
	Type    string       `xml:"type,attr"` // Any of IQType*
	From    *JID         `xml:"from,attr,omitempty"`
	To      *JID         `xml:"to,attr,omitempty"`
	Payload []byte       `xml:",innerxml"` //TODO:FIXME: this would contain all child elements
	Error   *StanzaError `xml:",omitempty"`
}

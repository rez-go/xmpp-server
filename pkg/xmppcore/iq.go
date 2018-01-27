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
	Type    string       `xml:"type,attr"`           // Any of IQType*
	From    string       `xml:"from,attr,omitempty"` //TODO: JID type
	To      string       `xml:"to,attr,omitempty"`   //TODO: JID type
	Payload []byte       `xml:",innerxml"`           //TODO:FIXME: this would mess up if there's another child element like error
	Error   *StanzaError `xml:",omitempty"`
}

package xmppcore

import (
	"encoding/xml"
)

// RFC 6120  8.3.2
const (
	StanzaErrorTypeAuth     = "auth"
	StanzaErrorTypeCancel   = "cancel"
	StanzaErrorTypeContinue = "continue"
	StanzaErrorTypeModify   = "modify"
	StanzaErrorTypeWait     = "wait"
)

// RFC 6120  8.3.2
type StanzaError struct {
	XMLName   xml.Name `xml:"jabber:client error"`
	By        string   `xml:"by,attr,omitempty"`
	Type      string   `xml:"type,attr"`
	Condition interface{}
	Text      string `xml:"text,omitempty"`
}

// RFC 6120  8.3.3.1
type StanzaErrorConditionBadRequest struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas bad-request"`
}

// RFC 6120  8.3.3.2
type StanzaErrorConditionConflict struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas conflict"`
}

// RFC 6120  8.3.3.19
type StanzaErrorConditionServiceUnavailable struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-stanzas service-unavailable"`
}

package xmppcore

import (
	"encoding/xml"
)

const StanzasNS = "urn:ietf:params:xml:ns:xmpp-stanzas"

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
	XMLName         xml.Name `xml:"jabber:client error"`
	By              string   `xml:"by,attr,omitempty"`
	Type            string   `xml:"type,attr"`
	Condition       StanzaErrorCondition
	Text            string      `xml:"text,omitempty"`
	CustomCondition interface{} `xml:",omitempty"`
}

type StanzaErrorCondition struct {
	XMLName xml.Name
}

var (
	StanzaErrorConditionBadRequest            = StanzaErrorCondition{xml.Name{Space: StanzasNS, Local: "bad-request"}}
	StanzaErrorConditionConflict              = StanzaErrorCondition{xml.Name{Space: StanzasNS, Local: "conflict"}}
	StanzaErrorConditionFeatureNotImplemented = StanzaErrorCondition{xml.Name{Space: StanzasNS, Local: "feature-not-implemented"}}
	StanzaErrorConditionServiceUnavailable    = StanzaErrorCondition{xml.Name{Space: StanzasNS, Local: "service-unavailable"}}
)

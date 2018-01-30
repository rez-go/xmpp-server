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
//NOTE: the spec says that this element might contain
// application-specific error condition.
type StanzaError struct {
	XMLName   xml.Name `xml:"jabber:client error"`
	By        string   `xml:"by,attr,omitempty"`
	Type      string   `xml:"type,attr"`
	Condition StanzaErrorCondition
	Text      string `xml:"text,omitempty"`
}

type StanzaErrorCondition struct {
	XMLName xml.Name // Deliberately un-tagged
}

// RFC 6120 section 8.3.3
var (
	StanzaErrorConditionBadRequest            = stanzaErrorCondition("bad-request")
	StanzaErrorConditionConflict              = stanzaErrorCondition("conflict")
	StanzaErrorConditionFeatureNotImplemented = stanzaErrorCondition("feature-not-implemented")
	StanzaErrorConditionServiceUnavailable    = stanzaErrorCondition("service-unavailable")
)

func stanzaErrorCondition(local string) StanzaErrorCondition {
	return StanzaErrorCondition{xml.Name{Space: StanzasNS, Local: local}}
}

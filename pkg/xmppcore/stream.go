package xmppcore

import (
	"encoding/xml"
)

const (
	StreamStreamElementName = JabberStreamsNS + " stream"
	StreamErrorElementName  = JabberStreamsNS + " error"
)

// RFC 6120  4.3.2  Streams Features Format

type NegotiationStreamFeatures struct {
	XMLName    xml.Name        `xml:"stream:features"`
	Mechanisms *SASLMechanisms `xml:"mechanisms,omitempty"`
	//TODO: TLS
	//TODO: allow mods to provide more features
}

// AuthenticatedStreamFeatures is used on the second stream
type AuthenticatedStreamFeatures struct {
	XMLName xml.Name `xml:"stream:features"`
	Bind    BindBind `xml:"bind"`
	//TODO: get more features from the mods
}

// RFC 6120  4.9  Stream Errors

// RFC 6120  4.9.2
//NOTE: the spec says that this element might contain
// application-specific error condition.
type StreamError struct {
	XMLName   xml.Name `xml:"http://etherx.jabber.org/streams error"`
	Condition StreamErrorCondition
	Text      string `xml:"text"`
}

// Per latest revision of RFC 6120, stream error conditions are empty elements.
type StreamErrorCondition struct {
	XMLName xml.Name // Deliberately un-tagged
}

// RFC 6120 section 4.9.3
var (
	StreamErrorConditionBadFormat           = streamErrorCondition("bad-format")
	StreamErrorConditionHostUnknown         = streamErrorCondition("host-unknown")
	StreamErrorConditionInternalServerError = streamErrorCondition("internal-server-error")
	StreamErrorConditionInvalidFrom         = streamErrorCondition("invalid-from")
	StreamErrorConditionNotAuthorized       = streamErrorCondition("not-authorized")
)

func streamErrorCondition(local string) StreamErrorCondition {
	return StreamErrorCondition{xml.Name{Space: StreamsNS, Local: local}}
}

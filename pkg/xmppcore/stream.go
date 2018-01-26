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

// StreamFeatures is used on the second stream
type StreamFeatures struct {
	XMLName xml.Name `xml:"stream:features"`
	Bind    BindBind `xml:"bind"`
	//TODO: get more features from the mods
}

// RFC 6120  4.9  Stream Errors

// RFC 6120  4.9.2
type StreamError struct {
	XMLName         xml.Name `xml:"http://etherx.jabber.org/streams error"`
	Condition       StreamErrorCondition
	Text            string      `xml:"text"`
	CustomCondition interface{} `xml:",omitempty"`
}

// RFC 6120  4.9.3  Defined Stream Error Conditions

// Per latest revision of RFC 6120, stream error conditions are empty elements.
//
// We put a sensible default condition which is internal-server-error
type StreamErrorCondition struct {
	XMLName xml.Name
}

var (
	StreamErrorConditionBadFormat           = StreamErrorCondition{xml.Name{Space: StreamsNS, Local: "bad-format"}}
	StreamErrorConditionHostUnknown         = StreamErrorCondition{xml.Name{Space: StreamsNS, Local: "host-unknown"}}
	StreamErrorConditionInternalServerError = StreamErrorCondition{xml.Name{Space: StreamsNS, Local: "internal-server-error"}}
	StreamErrorConditionInvalidFrom         = StreamErrorCondition{xml.Name{Space: StreamsNS, Local: "invalid-from"}}
	StreamErrorConditionNotAuthorized       = StreamErrorCondition{xml.Name{Space: StreamsNS, Local: "not-authorized"}}
)

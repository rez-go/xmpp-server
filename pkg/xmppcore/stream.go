package xmppcore

import (
	"encoding/xml"
)

const StreamErrorElementName = JabberStreamsNS + " error"

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
	XMLName   xml.Name `xml:"http://etherx.jabber.org/streams error"`
	Condition interface{}
	Text      string `xml:"text"`
}

// RFC 6120  4.9.3  Defined Stream Error Conditions

// RFC 6120  4.9.3.1
type StreamErrorConditionBadFormat struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-streams bad-format"`
}

// RFC 6120  4.9.3.9
type StreamErrorConditionInvalidFrom struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-streams invalid-from"`
}

// RFC 6120  4.9.3.12
type StreamErrorConditionNotAuthorized struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-streams not-authorized"`
}

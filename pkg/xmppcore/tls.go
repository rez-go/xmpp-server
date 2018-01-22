package xmppcore

import (
	"encoding/xml"
)

// RFC 6120  5  STARTTLS Negotiation

const TLSNS = "urn:ietf:params:xml:ns:xmpp-tls"

type TLSStartTLS struct {
	XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls starttls"`
	Required *string  `xml:"required,omitempty"`
}

type TLSProceed struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls proceed"`
}

type TLSFailure struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls failure"`
}

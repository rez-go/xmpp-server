package xmppcore

import (
	"encoding/xml"
)

// RFC 6120  7. Resource Binding

const BindNS = "urn:ietf:params:xml:ns:xmpp-bind"

const BindBindElementName = BindNS + " bind"

type BindBind struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
}

type BindIQSet struct {
	XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
	Resource string   `xml:"resource"`
}

type BindIQResult struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
	JID     *JID     `xml:"jid"`
}

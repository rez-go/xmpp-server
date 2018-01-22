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
	Resource BindResource
}

type BindIQResult struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
	JID     BindJID
}

type BindResource struct {
	XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind resource"`
	CharData string   `xml:",chardata"`
}

type BindJID struct {
	XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind jid"`
	CharData string   `xml:",chardata"` //TODO: should be JID type
}

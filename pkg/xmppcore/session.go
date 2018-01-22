package xmppcore

import (
	"encoding/xml"
)

//NOTE: session is not listed in the new XMPP Core (RFC 6120)
//TODO: check if it's mentioned somewhere else

const SessionNS = "urn:ietf:params:xml:ns:xmpp-session"

const SessionSessionElementName = SessionNS + " session"

type SessionIQSet struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-session session"`
}

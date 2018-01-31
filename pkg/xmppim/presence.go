package xmppim

import (
	"encoding/xml"

	"github.com/exavolt/xmpp-server/pkg/xmppcore"
)

const ClientPresenceElementName = xmppcore.JabberClientNS + " presence"

// RFC 6121  4.7.1
const (
	PresenceTypeError        = "error"
	PresenceTypeProbe        = "probe"
	PresenceTypeSubscribe    = "subscribe"
	PresenceTypeSubscribed   = "subscribed"
	PresenceTypeUnavailable  = "unavailable"
	PresenceTypeUnsubscribe  = "unsubscribe"
	PresenceTypeUnsubscribed = "unsubscribed"
)

// RFC 6121  4.7.
type ClientPresence struct {
	XMLName xml.Name              `xml:"jabber:client presence"`
	ID      string                `xml:"id,attr,omitempty"`
	Type    string                `xml:"type,attr,omitempty"`
	From    string                `xml:"from,attr,omitempty"`
	To      string                `xml:"to,attr,omitempty"`
	Error   *xmppcore.StanzaError `xml:",omitempty"`
	//TODO: 4.7.2.
	CapsC *CapsC `xml:",omitempty"`
	//TODO: X
}

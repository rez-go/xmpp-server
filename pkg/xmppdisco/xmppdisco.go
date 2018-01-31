package xmppdisco

import (
	"encoding/xml"

	"github.com/exavolt/xmpp-server/pkg/xmppcore"
)

// XEP-0030: Service Discovery

const (
	InfoNS  = "http://jabber.org/protocol/disco#info"
	ItemsNS = "http://jabber.org/protocol/disco#items"
)

const (
	InfoQueryElementName  = InfoNS + " query"
	ItemsQueryElementName = ItemsNS + " query"
)

type InfoIQGet struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/disco#info query"`
}

type InfoIQResult struct {
	XMLName  xml.Name   `xml:"http://jabber.org/protocol/disco#info query"`
	Node     string     `xml:"node,attr,omitempty"`
	Identity []Identity `xml:"identity,omitempty"`
	Feature  []Feature  `xml:"feature,omitempty"`
}

type ItemsIQGet struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/disco#items query"`
}

type ItemsIQResult struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/disco#items query"`
	Node    string   `xml:"node,attr,omitempty"`
	Item    []Item   `xml:"item,omitempty"`
}

// https://xmpp.org/registrar/disco-categories.html
const (
	IdentityCategoryAccount       = "account"
	IdentityCategoryAuth          = "auth"
	IdentityCategoryAutomation    = "automation"
	IdentityCategoryClient        = "client"
	IdentityCategoryCollaboration = "collaboration"
	IdentityCategoryComponent     = "component"
	IdentityCategoryConference    = "conference"
	IdentityCategoryDirectory     = "directory"
	IdentityCategoryGateway       = "gateway"
	IdentityCategoryHeadline      = "headline"
	IdentityCategoryHierarchy     = "hierarchy"
	IdentityCategoryProxy         = "proxy"
	IdentityCategoryPubsub        = "pubsub"
	IdentityCategoryServer        = "server"
	IdentityCategoryStore         = "store"
)

type Identity struct {
	Category string `xml:"category,attr"`
	Name     string `xml:"name,attr,omitempty"`
	Type     string `xml:"type,attr"`
}

type Feature struct {
	Var string `xml:"var,attr"`
}

type Item struct {
	JID  xmppcore.JID `xml:"jid,attr"`
	Name string       `xml:"name,attr,omitempty"`
	Node string       `xml:"node,attr,omitempty"`
}

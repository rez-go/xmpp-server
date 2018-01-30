package xmppim

import (
	"encoding/xml"

	"sandbox/gobber/pkg/xmppcore"
)

//TODO: look at RFC 6121 Appendix D.

const (
	RosterNS = "jabber:iq:roster"
)

const (
	RosterQueryElementName = RosterNS + " query"
)

type RosterIQGet struct {
	XMLName xml.Name `xml:"jabber:iq:roster query"`
}

type RosterIQResult struct {
	XMLName xml.Name     `xml:"jabber:iq:roster query"`
	Ver     string       `xml:"ver,attr,omitempty"`
	Item    []RosterItem `xml:"item,omitempty"`
}

const (
	RosterItemAskSubscribe = "subscribe"
)

const (
	RosterItemSubscriptionBoth   = "both"
	RosterItemSubscriptionFrom   = "from"
	RosterItemSubscriptionNone   = "none"
	RosterItemSubscriptionRemove = "remove"
	RosterItemSubscriptionTo     = "to"
)

type RosterItem struct {
	XMLName      xml.Name     `xml:"jabber:iq:roster item"`
	Approved     *bool        `xml:"approved,attr,omitempty"`
	Ask          string       `xml:"ask,attr,omitempty"`
	JID          xmppcore.JID `xml:"jid,attr"`
	Name         string       `xml:"name,attr,omitempty"`
	Subscription string       `xml:"subscription,attr,omitempty"`
	Group        string       `xml:"group,omitempty"`
}

package xmppim

import (
	"encoding/xml"
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

type RosterItem struct {
	XMLName      xml.Name `xml:"jabber:iq:roster item"`
	Approved     string   `xml:"approved,attr,omitempty"` // *bool?
	Ask          string   `xml:"ask,attr,omitempty"`      //TODO: what type?
	JID          string   `xml:"jid,attr"`                //TODO: xmppcore.JID
	Name         string   `xml:"name,attr,omitempty"`
	Subscription string   `xml:"subscription,attr,omitempty"`
	//TODO: Group
}

package xmppim

import (
	"encoding/xml"
)

//TODO: move this to its own package (XEP-0115)

const CapsNS = "http://jabber.org/protocol/caps"

const CapsCElementName = CapsNS + " c"

type CapsC struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/caps c"`
	Ext     string   `xml:"ext,attr,omitempty"` //DEPRECATED
	Hash    string   `xml:"hash,attr"`
	Node    string   `xml:"node,attr"`
	Ver     string   `xml:"ver,attr"`
}

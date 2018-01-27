package xmppim

import (
	"encoding/xml"

	"sandbox/gobber/pkg/xmppcore"
)

const (
	ClientMessageElementName        = xmppcore.JabberClientNS + " message"
	ClientMessageBodyElementName    = xmppcore.JabberClientNS + " body"
	ClientMessageSubjectElementName = xmppcore.JabberClientNS + " subject"
	ClientMessageThreadElementName  = xmppcore.JabberClientNS + " thread"
)

// RFC 6121 section 5.2.2
const (
	MessageTypeChat      = "chat"
	MessageTypeError     = "error"
	MessageTypeGroupChat = "groupchat"
	MessageTypeHeadline  = "headline"
	MessageTypeNormal    = "normal"
)

type ClientMessage struct {
	XMLName xml.Name `xml:"jabber:client message"`
	ID      string   `xml:"id,attr,omitempty"`
	From    string   `xml:"from,attr,omitempty"` //TODO: JID type
	To      string   `xml:"to,attr,omitempty"`   //TODO: JID type
	Type    string   `xml:"type,attr"`           // Any of MessageType*
	Payload []byte   `xml:",innerxml"`           //TODO: this is inefficient. we should stream this
}

type ClientMessageBody struct {
	XMLName xml.Name `xml:"jabber:client body"`
}

type ClientMessageSubject struct {
	XMLName xml.Name `xml:"jabber:client subject"`
}

type ClientMessageThread struct {
	XMLName xml.Name `xml:"jabber:client thread"`
	Parent  string   `xml:"parent,attr"`
}
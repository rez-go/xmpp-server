package main

import (
	"encoding/xml"
	"net"

	"sandbox/gobber/pkg/xmppcore"
)

type Client struct {
	conn             net.Conn
	negotiationState int
	sessionID        string
	xmlDecoder       *xml.Decoder
	jid              xmppcore.JID
}

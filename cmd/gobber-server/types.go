package main

import (
	"encoding/xml"
	"net"

	"sandbox/gobber/pkg/xmppcore"
)

type Client struct {
	conn             net.Conn
	negotiationState int
	streamID         string
	xmlDecoder       *xml.Decoder
	jid              xmppcore.JID
}

type SASLPlainAuthHandler interface {
	HandleSASLPlainAuth(username, password []byte) (bool, error)
}

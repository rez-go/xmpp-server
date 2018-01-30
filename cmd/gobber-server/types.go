package main

import (
	"encoding/xml"
	"net"

	"sandbox/gobber/pkg/xmppcore"
)

type Client struct {
	conn          net.Conn
	streamID      string
	xmlDecoder    *xml.Decoder
	jid           xmppcore.JID
	authenticated bool
	closingStream bool
}

type SASLPlainAuthVerifier interface {
	VerifySASLPlainAuth(username, password []byte) (bool, error)
}

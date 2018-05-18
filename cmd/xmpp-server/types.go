package main

import (
	"encoding/xml"
	"net"

	"github.com/exavolt/go-xmpplib/xmppcore"
)

type Client struct {
	conn          net.Conn
	streamID      string
	xmlDecoder    *xml.Decoder
	jid           xmppcore.JID
	authenticated bool
	resourceBound bool
	closingStream bool
}

func (cl *Client) JID() xmppcore.JID {
	return cl.jid
}

type SASLPlainAuthVerifier interface {
	VerifySASLPlainAuth(username, password []byte) (localpart string, resourcepart string, success bool, err error)
}

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"

	"sandbox/gobber/pkg/xmppcore"
)

func (srv *Server) handleClientSASLAuth(cl *Client, startElem *xml.StartElement) {
	var saslAuth xmppcore.SASLAuth

	err := cl.xmlDecoder.DecodeElement(&saslAuth, startElem)
	if err != nil {
		panic(err)
	}

	if saslAuth.Mechanism != "PLAIN" {
		panic("unsupported SASL mechanism")
	}
	authBytes, err := base64.StdEncoding.DecodeString(saslAuth.CharData)
	if err != nil {
		panic(err)
	}
	authSegments := bytes.SplitN(authBytes, []byte{0}, 3)
	if len(authSegments) != 3 {
		panic("there should be 3 parts here")
	}
	authOK, err := srv.saslPlainAuthHandler.HandleSASLPlainAuth(authSegments[1], authSegments[2])
	if err != nil {
		panic(err)
	}
	if authOK {
		authRespXML, err := xml.Marshal(&xmppcore.SASLSuccess{})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(authRespXML)
		cl.authenticated = true
		cl.jid.Local = string(authSegments[1]) //TODO: normalize
		if len(authSegments[0]) > 0 {
			// If the first segment is provided, we'll have an assumed session
			//TODO: check format, check user existence, check privilege, normalize
			//cl.jid.Local = string(authSegments[0])
		}
		srv.finishClientNegotiation(cl)
	} else {
		authRespXML, err := xml.Marshal(&xmppcore.SASLFailure{
			Condition: xmppcore.StreamErrorConditionNotAuthorized,
			Text:      "Invalid username or password",
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(authRespXML)
	}
}

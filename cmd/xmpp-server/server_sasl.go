package main

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"

	"github.com/exavolt/go-xmpplib/xmppcore"
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
	var assumedLocal string
	// if len(authSegments[0]) > 0 {
	// 	// If the first segment is provided, we'll have an assumed session
	// 	//TODO: check format, check user existence, check privilege, normalize
	// 	var assumedJID xmppcore.JID
	// 	assumedJID, err = xmppcore.ParseJID(string(authSegments[0]))
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	if assumedJID.Local == "" {
	// 		panic("Assume not a JID?")
	// 	}
	// 	if assumedJID.Domain != srv.jid.Domain {
	// 		panic("TODO: handle this")
	// 	}
	// 	assumedLocal = assumedJID.Local
	// }
	localpart, resourcepart, authOK, err := srv.saslPlainAuthVerifier.VerifySASLPlainAuth(
		authSegments[1], authSegments[2])
	if err != nil {
		panic(err)
	}
	if assumedLocal != "" {
		localpart = assumedLocal
	} else if localpart == "" {
		localpart = string(authSegments[1]) //TODO: normalize
	}
	if authOK {
		authRespXML, err := xml.Marshal(&xmppcore.SASLSuccess{})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(authRespXML)
		cl.authenticated = true
		cl.jid.Local = localpart
		cl.jid.Resource = resourcepart
		srv.finishClientNegotiation(cl)
	} else {
		authRespXML, err := xml.Marshal(&xmppcore.SASLFailure{
			Condition: xmppcore.SASLFailureConditionNotAuthorized,
			Text:      "Invalid username or password",
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(authRespXML)
	}
}

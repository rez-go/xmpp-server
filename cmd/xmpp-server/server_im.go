package main

import (
	"encoding/xml"

	"github.com/sirupsen/logrus"

	"github.com/exavolt/xmpp-server/pkg/xmppim"
)

//TODO: move to xmppim

func (srv *Server) handleClientPresence(cl *Client, startElem *xml.StartElement) {
	var presence xmppim.ClientPresence
	err := cl.xmlDecoder.DecodeElement(&presence, startElem)
	if err != nil {
		panic(err)
	}
	//TODO: broadcast to those subscribed
}

func (srv *Server) handleClientMessage(cl *Client, startElem *xml.StartElement) {
	var incoming xmppim.ClientMessage
	//NOTE:SEC: decoding the whole element might not the best practice
	// because it could be cause DoS.
	// generally we want to stream the child elements or limit the
	// decoder's buffer size.
	err := cl.xmlDecoder.DecodeElement(&incoming, startElem)
	if err != nil {
		panic(err)
	}

	if incoming.To == nil || incoming.To.IsEmpty() {
		//TODO: skip the whole stanza
		return //TODO: tell the client
	}

	// We should parse the child elements. For now, let's just
	// relay them as-is.

	//TODO: this is inefficient. probably we want channels here.
	srv.clientsMutex.RLock()
	defer srv.clientsMutex.RUnlock()

	for _, rcl := range srv.clients {
		if rcl.jid.Local == incoming.To.Local {
			outgoing := xmppim.ClientMessage{
				ID:      incoming.ID,
				To:      &rcl.jid,
				From:    cl.jid.BareCopyPtr(), //TODO: optional, bare or full (check the spec)
				Type:    incoming.Type,
				Payload: incoming.Payload,
			}
			msgXML, err := xml.Marshal(&outgoing)
			if err != nil {
				logrus.WithFields(logrus.Fields{"stream": rcl.streamID, "jid": rcl.jid, "stanza": incoming.ID}).
					Warn("Unable to send a message into a recipient")
				continue
			}
			rcl.conn.Write(msgXML)
		}
	}
}

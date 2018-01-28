package main

import (
	"encoding/xml"

	"sandbox/gobber/pkg/xmppcore"
	"sandbox/gobber/pkg/xmppim"
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
	//NOTE: decoding the whole element might not the best practice.
	// generally we want to stream the child elements.
	err := cl.xmlDecoder.DecodeElement(&incoming, startElem)
	if err != nil {
		panic(err)
	}

	toJID, err := xmppcore.ParseJID(incoming.To)
	if err != nil {
		panic(err)
	}
	if toJID.IsEmpty() {
		//TODO: skip the whole stanza
		return //TODO: tell the client
	}

	// We should parse the child elements. For now, let's just
	// relay them as-is.

	//TODO: this is inefficient
	srv.clientsMutex.RLock()
	defer srv.clientsMutex.RUnlock()

	for _, rcl := range srv.clients {
		if rcl.jid.Local == toJID.Local {
			outgoing := xmppim.ClientMessage{
				ID:      incoming.ID,
				To:      rcl.jid.FullString(),
				From:    cl.jid.BareString(), //TODO: optional, bare or full
				Type:    incoming.Type,
				Payload: incoming.Payload,
			}
			msgXML, err := xml.Marshal(&outgoing)
			if err != nil {
				continue //TODO: deal this
			}
			rcl.conn.Write(msgXML)
		}
	}
}

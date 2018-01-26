package main

import (
	"bytes"
	"encoding/xml"
	"io"

	"github.com/sirupsen/logrus"

	"sandbox/gobber/pkg/xmppcore"
	"sandbox/gobber/pkg/xmppdisco"
	"sandbox/gobber/pkg/xmppim"
	"sandbox/gobber/pkg/xmppping"
	"sandbox/gobber/pkg/xmppprivate"
	"sandbox/gobber/pkg/xmppvcard"
)

func (srv *Server) handleClientIQ(cl *Client, startElem *xml.StartElement) {
	var iq xmppcore.ClientIQ
	//NOTE: decoding the whole element might not the best practice.
	// some IQs might better be streamed.
	err := cl.xmlDecoder.DecodeElement(&iq, startElem)
	if err != nil {
		panic(err)
	}

	switch iq.Type {
	case xmppcore.IQTypeSet:
		srv.handleClientIQSet(cl, &iq)
	case xmppcore.IQTypeGet:
		srv.handleClientIQGet(cl, &iq)
	default:
		panic(iq.Type)
	}
}

func (srv *Server) handleClientIQSet(cl *Client, iq *xmppcore.ClientIQ) {
	// Only one payload
	reader := bytes.NewReader(iq.Payload)
	decoder := xml.NewDecoder(reader)

	token, err := decoder.Token()
	if err == io.EOF {
		return
	}
	if err != nil {
		logrus.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid.Full()}).
			Errorf("Unexpected error: %#v", err)
		return
	}

	switch token.(type) {
	case xml.StartElement:
		// Pass
	default:
		panic(token)
	}

	startElem := token.(xml.StartElement)

	var element interface{}
	switch startElem.Name.Space + " " + startElem.Name.Local {
	case xmppcore.BindBindElementName:
		element = &xmppcore.BindIQSet{}
	case xmppcore.SessionSessionElementName:
		element = &xmppcore.SessionIQSet{}
	case xmppvcard.ElementName:
		element = &xmppvcard.IQSet{}
	case xmppprivate.ElementName:
		errorXML, err := xml.Marshal(&xmppcore.StanzaError{
			Type:      xmppcore.StanzaErrorTypeCancel,
			Condition: xmppcore.StanzaErrorConditionFeatureNotImplemented,
		})
		if err != nil {
			panic(err)
		}
		resultXML, err := xml.Marshal(&xmppcore.ClientIQ{
			ID:      iq.ID,
			Type:    xmppcore.IQTypeError,
			From:    srv.domain,
			To:      cl.jid.Full(),
			Payload: errorXML,
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
		return
	default:
		panic(startElem.Name.Space + " " + startElem.Name.Local)
	}

	err = decoder.DecodeElement(element, &startElem)
	if err != nil {
		panic(err)
	}

	switch payload := element.(type) {
	case *xmppcore.BindIQSet:
		//TODO: if not provided, generate. also, if configured, override.
		cl.jid.Resource = payload.Resource.CharData //TODO: normalize
		logrus.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid.Full()}).
			Info("Bound!")

		resultPayloadXML, err := xml.Marshal(&xmppcore.BindIQResult{
			JID: xmppcore.BindJID{CharData: cl.jid.Full()},
		})
		if err != nil {
			panic(err)
		}
		resultXML, err := xml.Marshal(&xmppcore.ClientIQ{
			ID:      iq.ID,
			Type:    xmppcore.IQTypeResult,
			Payload: resultPayloadXML,
		})
		if err != nil {
			panic(err)
		}

		cl.conn.Write(resultXML)
		return
	case *xmppcore.SessionIQSet:
		resultXML, err := xml.Marshal(&xmppcore.ClientIQ{
			ID:   iq.ID,
			Type: xmppcore.IQTypeResult,
			From: cl.jid.Bare(),
			To:   cl.jid.Full(),
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
		return
	case *xmppvcard.IQSet:
		//TODO: save the vCard
		resultXML, err := xml.Marshal(&xmppcore.ClientIQ{
			ID:   iq.ID,
			Type: xmppcore.IQTypeResult,
			From: srv.domain,
			To:   cl.jid.Full(),
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
		return
	}

	// An IQ stanza of type "get" or "set" MUST contain exactly
	// one child element, which specifies the semantics of the
	// particular request.
	if _, err := decoder.Token(); err != io.EOF {
		errorXML, err := xml.Marshal(xmppcore.StanzaError{
			Type:      xmppcore.StanzaErrorTypeModify,
			Condition: xmppcore.StanzaErrorConditionBadRequest,
		})
		if err != nil {
			panic(err)
		}
		resultXML, err := xml.Marshal(xmppcore.ClientIQ{
			ID:      iq.ID,
			Type:    xmppcore.IQTypeError,
			From:    srv.domain,
			To:      cl.jid.Full(),
			Payload: errorXML,
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
	}
}

func (srv *Server) handleClientIQGet(cl *Client, iq *xmppcore.ClientIQ) {
	reader := bytes.NewReader(iq.Payload)
	decoder := xml.NewDecoder(reader)

	token, err := decoder.Token()
	if err == io.EOF {
		return
	}
	if err != nil {
		logrus.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid.Full()}).
			Errorf("Unexpected error: %#v", err)
		return
	}

	switch token.(type) {
	case xml.StartElement:
		// Pass
	default:
		panic(token)
	}

	// RFC 6120  4.9.3.9
	if iq.From != "" && iq.From != cl.jid.Full() {
		logrus.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid.Full(), "stanza": iq.ID}).
			Warnf("Invalid from: %s", iq.From)
		errorXML, err := xml.Marshal(xmppcore.StreamError{
			Condition: xmppcore.StreamErrorConditionInvalidFrom,
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write([]byte(string(errorXML) + "\n</stream:stream>\n"))
		//TODO: close connection, etc.
		return
	}
	//TODO: check RFC 6120 8.1.1.1.
	if iq.To != "" && iq.To != srv.domain {
		logrus.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid.Full(), "stanza": iq.ID}).
			Warnf("Invalid to: %s", iq.To)
		errorXML, err := xml.Marshal(xmppcore.StanzaError{
			Type:      xmppcore.StanzaErrorTypeCancel,
			Condition: xmppcore.StanzaErrorConditionServiceUnavailable,
		})
		if err != nil {
			panic(err)
		}
		resultXML, err := xml.Marshal(xmppcore.ClientIQ{
			ID:      iq.ID,
			Type:    xmppcore.IQTypeError,
			From:    srv.domain,
			To:      cl.jid.Full(),
			Payload: errorXML,
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
		return
	}

	startElem := token.(xml.StartElement)

	var element interface{}
	switch startElem.Name.Space + " " + startElem.Name.Local {
	case xmppdisco.InfoQueryElementName:
		element = &xmppdisco.InfoIQGet{}
	case xmppdisco.ItemsQueryElementName:
		element = &xmppdisco.ItemsIQGet{}
	case xmppvcard.ElementName:
		element = &xmppvcard.IQGet{}
	case xmppim.RosterQueryElementName:
		element = &xmppim.RosterIQGet{}
	case xmppping.ElementName:
		element = &xmppping.IQGet{}
	case xmppprivate.ElementName:
		errorXML, err := xml.Marshal(&xmppcore.StanzaError{
			Type:      xmppcore.StanzaErrorTypeCancel,
			Condition: xmppcore.StanzaErrorConditionFeatureNotImplemented,
		})
		if err != nil {
			panic(err)
		}
		resultXML, err := xml.Marshal(&xmppcore.ClientIQ{
			ID:      iq.ID,
			Type:    xmppcore.IQTypeError,
			From:    srv.domain,
			To:      cl.jid.Full(),
			Payload: errorXML,
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
		return
	default:
		panic(startElem.Name.Space + " " + startElem.Name.Local)
	}

	err = decoder.DecodeElement(element, &startElem)
	if err != nil {
		panic(err)
	}

	switch element.(type) {
	case *xmppdisco.InfoIQGet:
		//TODO: check the target resource etc.
		queryResultXML, err := xml.Marshal(xmppdisco.InfoIQResult{
			Identity: []xmppdisco.Identity{
				{Category: xmppdisco.IdentityCategoryServer, Type: "im", Name: "gobber"},
			},
			Feature: []xmppdisco.Feature{
				{Var: "iq"},
			},
		})
		if err != nil {
			panic(err)
		}
		resultXML, err := xml.Marshal(xmppcore.ClientIQ{
			ID:      iq.ID,
			Type:    xmppcore.IQTypeResult,
			From:    srv.domain, //TODO: server's JID
			To:      cl.jid.Full(),
			Payload: queryResultXML,
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
		return
	case *xmppdisco.ItemsIQGet:
		//TODO: check the target resource etc.
		//TODO: conference, pubsub, etc.
		queryResultXML, err := xml.Marshal(xmppdisco.ItemsIQResult{})
		if err != nil {
			panic(err)
		}
		resultXML, err := xml.Marshal(xmppcore.ClientIQ{
			ID:      iq.ID,
			Type:    xmppcore.IQTypeResult,
			From:    srv.domain,
			To:      cl.jid.Full(),
			Payload: queryResultXML,
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
		return
	case *xmppvcard.IQGet:
		//TODO: check `from` etc.
		resultPayloadXML, err := xml.Marshal(xmppvcard.IQResult{})
		if err != nil {
			panic(err)
		}
		resultXML, err := xml.Marshal(xmppcore.ClientIQ{
			ID:      iq.ID,
			Type:    xmppcore.IQTypeResult,
			From:    srv.domain,
			To:      cl.jid.Full(),
			Payload: resultPayloadXML,
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
		return
	case *xmppim.RosterIQGet:
		resultPayloadXML, err := xml.Marshal(xmppim.RosterIQResult{})
		if err != nil {
			panic(err)
		}
		resultXML, err := xml.Marshal(xmppcore.ClientIQ{
			ID:      iq.ID,
			Type:    xmppcore.IQTypeResult,
			From:    srv.domain,
			To:      cl.jid.Full(),
			Payload: resultPayloadXML,
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
		return
	case *xmppping.IQGet:
		//TODO: support various cases (s2c, c2s, s2s, ...)
		resultXML, err := xml.Marshal(xmppcore.ClientIQ{
			ID:   iq.ID,
			Type: xmppcore.IQTypeResult,
			From: srv.domain,
			To:   cl.jid.Full(),
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
		return
	}

	// An IQ stanza of type "get" or "set" MUST contain exactly
	// one child element, which specifies the semantics of the
	// particular request.
	if _, err := decoder.Token(); err != io.EOF {
		errorXML, err := xml.Marshal(xmppcore.StanzaError{
			Type:      xmppcore.StanzaErrorTypeModify,
			Condition: xmppcore.StanzaErrorConditionBadRequest,
		})
		if err != nil {
			panic(err)
		}
		resultXML, err := xml.Marshal(xmppcore.ClientIQ{
			ID:      iq.ID,
			Type:    xmppcore.IQTypeError,
			From:    srv.domain,
			To:      cl.jid.Full(),
			Payload: errorXML,
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
	}
}

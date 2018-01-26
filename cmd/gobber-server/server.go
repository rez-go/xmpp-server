package main

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/itchyny/base58-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"sandbox/gobber/pkg/xmppcore"
	"sandbox/gobber/pkg/xmppdisco"
	"sandbox/gobber/pkg/xmppim"
	"sandbox/gobber/pkg/xmppping"
	"sandbox/gobber/pkg/xmppprivate"
	"sandbox/gobber/pkg/xmppvcard"
)

//TODO: use locking to ensure that there will be one write into
// the connection.

type Server struct {
	DoneCh chan bool

	name   string
	domain string //TODO: support multiple

	startTime time.Time
	stopCh    chan bool
	stopState int

	listener         net.Listener
	clients          map[string]*Client
	clientsMutex     sync.RWMutex
	clientsWaitGroup sync.WaitGroup
}

func New(cfg *Config) (*Server, error) {
	if cfg == nil {
		return nil, nil
	}
	listener, err := net.Listen("tcp", "localhost:"+cfg.Port)
	if err != nil {
		return nil, err
	}
	srv := &Server{
		DoneCh:   make(chan bool),
		name:     cfg.Name,
		domain:   cfg.Domain, //TODO: normalize
		stopCh:   make(chan bool),
		listener: listener,
		clients:  make(map[string]*Client),
	}
	return srv, nil
}

func (srv *Server) Serve() {
	defer func() {
		srv.stopState = 2
		srv.DoneCh <- true
	}()

	srv.startTime = time.Now()
	go srv.listen()

mainloop:
	for {
		select {
		case <-srv.stopCh:
			close(srv.stopCh)
			break mainloop
		}
	}

	// Stopping
	logrus.Infof("Server is stopping after %s uptime...", srv.Uptime())
	srv.stopState = 1
	srv.listener.Close()

	//TODO: notify all clients that this server is going down
	// use system-shutdown stream error condition

	srv.clientsWaitGroup.Wait()

	logrus.Info("Server cleanly stopped")
}

func (srv *Server) Stop() {
	srv.stopCh <- true
}

func (srv *Server) Stopped() bool {
	return srv.stopState == 2
}

func (srv *Server) Uptime() time.Duration {
	return time.Since(srv.startTime)
}

func (srv *Server) listen() {
	for srv.stopState == 0 {
		conn, err := srv.listener.Accept()
		if err != nil {
			if srv.stopState == 0 {
				logrus.Error("Listener error: ", err)
			}
			break
		}
		if conn == nil {
			continue
		}

		cl, err := srv.newClient(conn)
		if err != nil {
			logrus.Error("Unable to create client: ", err)
			continue
		}

		logrus.WithFields(logrus.Fields{"stream": cl.streamID}).Info("Client connected")
		srv.clientsWaitGroup.Add(1)
		go srv.serveClient(cl)
	}
}

func (srv *Server) newClient(conn net.Conn) (*Client, error) {
	if conn == nil {
		return nil, nil
	}
	sid, err := srv.generateSessionID()
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate session id")
	}
	cl := &Client{
		conn:       conn,
		streamID:   sid,
		xmlDecoder: xml.NewDecoder(conn),
		jid:        xmppcore.JID{Domain: srv.domain},
	}
	srv.clientsMutex.Lock()
	srv.clients[cl.streamID] = cl
	srv.clientsMutex.Unlock()
	return cl, nil
}

func (srv *Server) serveClient(cl *Client) {
	//TODO: on panic, simply close the connection.
	defer func() {
		if cl.conn != nil {
			cl.conn.Close()
			cl.conn = nil
		}

		srv.clientsMutex.Lock()
		delete(srv.clients, cl.streamID)
		srv.clientsMutex.Unlock()

		srv.clientsWaitGroup.Done()
	}()

mainloop:
	for {
		token, err := cl.xmlDecoder.Token()
		if err != nil {
			if xmlErr, _ := err.(*xml.SyntaxError); xmlErr != nil {
				if xmlErr.Line == 1 && xmlErr.Msg == "unexpected EOF" {
					logrus.WithFields(logrus.Fields{"stream": cl.streamID}).
						Info("Client disconnected")
					break mainloop
				}
			}
			logrus.WithFields(logrus.Fields{"stream": cl.streamID}).
				Errorf("Unexpected error: %#v", err)
			break mainloop
		}
		if token == nil {
			logrus.WithFields(logrus.Fields{"stream": cl.streamID}).
				Errorf("Token is nil")
			break mainloop
		}

		//TODO: check for EndElement which closes the stream
		//TODO: check for restricted-xml
		switch token.(type) {
		case xml.EndElement:
			endElem := token.(xml.EndElement)
			if endElem.Name.Space == xmppcore.JabberStreamsNS && endElem.Name.Local == "stream" {
				logrus.WithFields(logrus.Fields{"stream": cl.streamID}).Info("Disconnecting client")
				cl.conn.Write([]byte("</stream:stream>"))
				cl.conn.Close()
				break mainloop
			}
			logrus.WithFields(logrus.Fields{"stream": cl.streamID}).
				Errorf("Unexpected EndElement: %#v", endElem)
			panic(endElem)
		case xml.StartElement:
			// Pass
		default:
			logrus.Warnf("%#v", token)
			continue
		}

		startElem := token.(xml.StartElement)

		switch startElem.Name.Space + " " + startElem.Name.Local {
		case xmppcore.StreamStreamElementName:
			if srv.handlerClientStreamOpen(cl, &startElem) {
				continue
			}
			//TODO: graceful close
			cl.conn.Write([]byte("</stream:stream>"))
			break mainloop
		case xmppcore.SASLAuthElementName:
			srv.handleClientSASLAuth(cl, &startElem)
			continue
		case xmppcore.ClientIQElementName:
			srv.handleClientIQ(cl, &startElem)
			continue
		case xmppim.ClientPresenceElementName:
			srv.handleClientPresence(cl, &startElem)
			continue
		default:
			logrus.WithFields(logrus.Fields{"stream": cl.streamID}).
				Warn("unexpected XMPP stanza: ", startElem.Name)
			continue
		}
	}
}

func (srv *Server) handlerClientStreamOpen(cl *Client, startElem *xml.StartElement) bool {
	var toAttr, fromAttr string
	for _, attr := range startElem.Attr {
		switch attr.Name.Local {
		case "to":
			toAttr = attr.Value //TODO: parse JID
		case "from":
			fromAttr = attr.Value //TODO: parse JID
		}
	}

	if toAttr != srv.domain {
		resultXML, err := xml.Marshal(&xmppcore.StreamError{
			Condition: xmppcore.StreamErrorConditionHostUnknown,
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
		return false
	}
	if fromAttr != "" {
		if !strings.HasSuffix(fromAttr, srv.domain) {
			resultXML, err := xml.Marshal(&xmppcore.StreamError{
				Condition: xmppcore.StreamErrorConditionInvalidFrom,
			})
			if err != nil {
				panic(err)
			}
			cl.conn.Write(resultXML)
			return false
		}
	}

	var err error
	var featuresXML []byte
	if cl.negotiationState == 2 {
		featuresXML, err = xml.Marshal(&xmppcore.StreamFeatures{})
		if err != nil {
			panic(err)
		}
	} else {
		//TODO: get features from the config and mods
		featuresXML, err = xml.Marshal(&xmppcore.NegotiationStreamFeatures{
			Mechanisms: &xmppcore.SASLMechanisms{
				Mechanism: []string{"PLAIN"},
			},
		})
		if err != nil {
			panic(err)
		}
	}

	//TODO: include 'to' if provided
	_, err = fmt.Fprintf(cl.conn, xml.Header+
		"<stream:stream from='%s' xmlns='%s'"+
		" id='%s' xml:lang='en'"+
		" xmlns:stream='%s' version='1.0'>\n"+
		string(featuresXML)+"\n",
		xmlEscape(srv.domain), xmppcore.JabberClientNS,
		xmlEscape(cl.streamID), xmppcore.JabberStreamsNS)
	if err != nil {
		panic(err)
	}

	return true
}

func (srv *Server) finishClientNegotiation(cl *Client) {
	sid, err := srv.generateSessionID()
	if err != nil {
		panic(err)
	}
	srv.clientsMutex.Lock()
	delete(srv.clients, cl.streamID)
	cl.streamID = sid
	srv.clients[cl.streamID] = cl
	srv.clientsMutex.Unlock()
}

func (srv *Server) startClientTLS(cl *Client) {

}

func (srv *Server) handleClientSASLAuth(cl *Client, startElem *xml.StartElement) {
	var saslAuth xmppcore.SASLAuth

	err := cl.xmlDecoder.DecodeElement(&saslAuth, startElem)
	if err != nil {
		panic(err)
	}

	if saslAuth.Mechanism != "PLAIN" {
		panic("unsupported SASL mechanish")
	}
	authBytes, err := base64.StdEncoding.DecodeString(saslAuth.CharData)
	if err != nil {
		panic(err)
	}
	authSegments := bytes.SplitN(authBytes, []byte{0}, 3)
	if len(authSegments) != 3 {
		panic("there should be 3 parts here")
	}
	if string(authSegments[1]) == "admin" && string(authSegments[2]) == "passw0rd" {
		authRespXML, err := xml.Marshal(&xmppcore.SASLSuccess{})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(authRespXML)
		cl.negotiationState = 2
		cl.jid.Local = string(authSegments[1]) //TODO: normalize
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
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
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
	}
}

func (srv *Server) handleClientIQGet(cl *Client, iq *xmppcore.ClientIQ) {
	// There should only one payload (TODO: check the spec on how
	// to handle multiple child element)
	reader := bytes.NewReader(iq.Payload)
	decoder := xml.NewDecoder(reader)

	token, err := decoder.Token()
	if err == io.EOF {
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
	case *xmppping.Ping:
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
}

//TODO: move to xmppim
func (srv *Server) handleClientPresence(cl *Client, startElem *xml.StartElement) {
	var presence xmppim.ClientPresence
	err := cl.xmlDecoder.DecodeElement(&presence, startElem)
	if err != nil {
		panic(err)
	}
	//TODO: broadcast to those subscribed
}

func (srv *Server) generateSessionID() (string, error) {
	//NOTE: this whole function is highly inefficient.
	sid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	i := new(big.Int).SetBytes(sid[:])
	sidEncd, err := base58.BitcoinEncoding.Encode([]byte(i.String()))
	if err != nil {
		return "", err
	}
	return string(sidEncd), nil
}

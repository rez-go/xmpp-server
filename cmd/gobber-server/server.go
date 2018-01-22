package main

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/itchyny/base58-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"sandbox/gobber/pkg/xmppcore"
	"sandbox/gobber/pkg/xmppdisco"
	"sandbox/gobber/pkg/xmppimp"
	"sandbox/gobber/pkg/xmppvcard"
)

type Server struct {
	DoneCh chan bool

	name   string
	domain string

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

		logrus.WithFields(logrus.Fields{"stream": cl.sessionID}).Info("Client connected")
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
		sessionID:  sid,
		xmlDecoder: xml.NewDecoder(conn),
		jid:        xmppcore.JID{Domain: srv.domain},
	}
	srv.clientsMutex.Lock()
	srv.clients[cl.sessionID] = cl
	srv.clientsMutex.Unlock()
	return cl, nil
}

func (srv *Server) serveClient(cl *Client) {
	//TODO: on panic, simply close the connection.
	defer func() {
		srv.clientsMutex.Lock()
		delete(srv.clients, cl.sessionID)
		srv.clientsMutex.Unlock()
		srv.clientsWaitGroup.Done()
	}()

mainloop:
	for {
		token, err := cl.xmlDecoder.Token()
		if err == io.EOF {
			logrus.WithFields(logrus.Fields{"stream": cl.sessionID}).Info("Client disconnected")
			break
		}
		if err != nil || token == nil {
			logrus.WithFields(logrus.Fields{"stream": cl.sessionID}).Warn(err)
			break
		}
		//TODO: check for EndElement which closes the stream
		switch token.(type) {
		case xml.EndElement:
			endElem := token.(xml.EndElement)
			if endElem.Name.Space == xmppcore.JabberStreamsNS && endElem.Name.Local == "stream" {
				cl.conn.Write([]byte("</stream:stream>"))
				cl.conn.Close()
				break mainloop
			}
			logrus.WithFields(logrus.Fields{"stream": cl.sessionID}).
				Errorf("Unexpected EndElement: %#v", endElem)
			panic(endElem)
		case xml.StartElement:
			// Pass
		default:
			logrus.Warnf("%#v", token)
			continue
		}

		startElem := token.(xml.StartElement)
		if startElem.Name.Space == xmppcore.JabberStreamsNS && startElem.Name.Local == "stream" {
			var domain string
			for _, attr := range startElem.Attr {
				if attr.Name.Local == "to" {
					domain = attr.Value //TODO: compare with server's domain
				}
			}
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
			_, err = fmt.Fprintf(cl.conn, xml.Header+
				"<stream:stream from='%s' xmlns='%s'"+
				" id='%s' xml:lang='en'"+
				" xmlns:stream='%s' version='1.0'>\n"+
				string(featuresXML)+"\n",
				xmlEscape(domain), xmppcore.JabberClientNS,
				xmlEscape(cl.sessionID), xmppcore.JabberStreamsNS)
			if err != nil {
				panic(err)
			}

			continue
		}

		var element interface{}
		switch startElem.Name.Space + " " + startElem.Name.Local {
		case xmppcore.SASLAuthElementName:
			element = &xmppcore.SASLAuth{}
		case xmppcore.ClientIQElementName:
			element = &xmppcore.ClientIQ{}
		case xmppimp.ClientPresenceElementName:
			element = &xmppimp.ClientPresence{}
		default:
			logrus.WithFields(logrus.Fields{"stream": cl.sessionID}).
				Warn("unexpected XMPP stanza: ", startElem.Name)
			continue
		}

		err = cl.xmlDecoder.DecodeElement(element, &startElem)
		if err != nil {
			panic(err)
		}

		switch element.(type) {
		case *xmppcore.SASLAuth:
			saslAuth := element.(*xmppcore.SASLAuth)
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
					Condition: &xmppcore.StreamErrorConditionNotAuthorized{},
					Text:      "Invalid username or password",
				})
				if err != nil {
					panic(err)
				}
				cl.conn.Write(authRespXML)
			}
		case *xmppcore.ClientIQ:
			srv.handleClientIQ(cl, element.(*xmppcore.ClientIQ))
		default:
			logrus.WithFields(logrus.Fields{"stream": cl.sessionID}).Errorf("%#v", element)
		}
	}
}

func (srv *Server) finishClientNegotiation(cl *Client) {
	sid, err := srv.generateSessionID()
	if err != nil {
		panic(err)
	}
	srv.clientsMutex.Lock()
	delete(srv.clients, cl.sessionID)
	cl.sessionID = sid
	srv.clients[cl.sessionID] = cl
	srv.clientsMutex.Unlock()
}

func (srv *Server) startClientTLS(cl *Client) {

}

func (srv *Server) handleClientIQ(cl *Client, iq *xmppcore.ClientIQ) {
	switch iq.Type {
	case xmppcore.IQTypeSet:
		srv.handleClientIQSet(cl, iq)
	case xmppcore.IQTypeGet:
		srv.handleClientIQGet(cl, iq)
	default:
		panic(iq.Type)
	}
}

func (srv *Server) handleClientIQSet(cl *Client, iq *xmppcore.ClientIQ) {
	// Only one payload
	reader := bytes.NewReader(iq.Payload)
	decdr := xml.NewDecoder(reader)
	for {
		token, err := decdr.Token()
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
			panic("TODO")
		}

		err = decdr.DecodeElement(element, &startElem)
		if err != nil {
			panic(err)
		}

		switch payload := element.(type) {
		case *xmppcore.BindIQSet:
			//TODO: if not provided, generate. also, if configured, override.
			cl.jid.Resource = payload.Resource.CharData //TODO: normalize
			logrus.WithFields(logrus.Fields{"stream": cl.sessionID, "jid": cl.jid.Full()}).
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
	// Only one payload
	reader := bytes.NewReader(iq.Payload)
	decdr := xml.NewDecoder(reader)
	for {
		token, err := decdr.Token()
		if err == io.EOF {
			break
		}

		switch token.(type) {
		case xml.StartElement:
			// Pass
		default:
			panic(token)
		}

		// RFC 6120  4.9.3.9
		if iq.From != "" && iq.From != cl.jid.Full() {
			logrus.WithFields(logrus.Fields{"stream": cl.sessionID, "jid": cl.jid.Full(), "stanza": iq.ID}).
				Warnf("Invalid from: %s", iq.From)
			errorXML, err := xml.Marshal(xmppcore.StreamError{
				Condition: &xmppcore.StreamErrorConditionInvalidFrom{},
			})
			if err != nil {
				panic(err)
			}
			cl.conn.Write([]byte(string(errorXML) + "\n</stream:stream>\n"))
			//TODO: close connection, etc.
			return
		}
		//TODO: this should be case-by-case, and in federation, we must
		// relay the stanza
		if iq.To != "" && iq.To != srv.domain {
			logrus.WithFields(logrus.Fields{"stream": cl.sessionID, "jid": cl.jid.Full(), "stanza": iq.ID}).
				Warnf("Invalid to: %s", iq.To)
			errorXML, err := xml.Marshal(xmppcore.StanzaError{
				Type:      xmppcore.StanzaErrorTypeCancel,
				Condition: &xmppcore.StanzaErrorConditionServiceUnavailable{},
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
		case xmppimp.RosterQueryElementName:
			element = &xmppimp.RosterIQGet{}
		default:
			panic(startElem.Name.Space)
		}

		err = decdr.DecodeElement(element, &startElem)
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
		case *xmppimp.RosterIQGet:
			resultPayloadXML, err := xml.Marshal(xmppimp.RosterIQResult{})
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
		}
	}
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

package main

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/exavolt/xmpp-server/cmd/xmpp-server/oauth"
	"github.com/exavolt/xmpp-server/pkg/xmppcore"
	"github.com/exavolt/xmpp-server/pkg/xmppim"
)

//NOTE: currently, we don't any plan for having support for serving
// multiple domain as it increases the complexity of the code. we would
// suggest to look at other / higher-level solutions.
//NOTE: it seems that we don't need to acquire lock to net.Conn to
// prevent multiple goroutines from messing up the network stream.
// https://stackoverflow.com/questions/38565654/golang-net-conn-write-in-parallel

// Get from config
const (
	OAuthTokenEndpoint = "http://localhost:8080/oauth/token"
	OAuthClientID      = ""
	OAuthClientSecret  = ""
)

type Server struct {
	DoneCh chan bool

	name string
	jid  xmppcore.JID

	saslPlainAuthVerifier SASLPlainAuthVerifier

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
	saslPlainAuthVerifier := &oauth.Authenticator{
		TokenEndpoint: OAuthTokenEndpoint,
		ClientID:      OAuthClientID,
		ClientSecret:  OAuthClientSecret,
	}
	srv := &Server{
		DoneCh: make(chan bool),
		name:   cfg.Name,
		jid:    xmppcore.JID{Domain: cfg.Domain}, //TODO: normalize
		saslPlainAuthVerifier: saslPlainAuthVerifier,
		stopCh:                make(chan bool),
		listener:              listener,
		clients:               make(map[string]*Client),
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

	srv.clientsMutex.RLock()
	for _, cl := range srv.clients {
		srv.notifyClientSystemShutdown(cl)
	}
	srv.clientsMutex.RUnlock()

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
		xmlDecoder: xml.NewDecoder(conn), //TODO: is there a way to limit the decoder's buffer size?
		jid:        xmppcore.JID{Domain: srv.jid.Domain},
	}
	srv.clientsMutex.Lock()
	srv.clients[cl.streamID] = cl
	srv.clientsMutex.Unlock()
	return cl, nil
}

func (srv *Server) serveClient(cl *Client) {
	defer func() {
		if cl.conn != nil {
			cl.conn.Close()
			cl.conn = nil
		}

		logrus.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid}).
			Info("Client disconnected")

		srv.clientsMutex.Lock()
		delete(srv.clients, cl.streamID)
		srv.clientsMutex.Unlock()

		srv.clientsWaitGroup.Done()
	}()

mainloop:
	for {
		token, err := cl.xmlDecoder.Token()
		if err != nil {
			// Clean disconnection
			if err == io.EOF {
				logrus.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid}).
					Info("Client connection closed")
				break mainloop
			}
			// Un-clean disconnection (the connection is closed while
			// the stream is still open)
			//NOTE: this could be a expected case for every authenticated
			// stream. for each authenticated stream, there will be two
			// streams and we will only close the 'inner' stream.
			if xmlErr, _ := err.(*xml.SyntaxError); xmlErr != nil {
				if xmlErr.Line == 1 && xmlErr.Msg == "unexpected EOF" {
					logrus.WithFields(logrus.Fields{"stream": cl.streamID}).
						Info("Client connection closed without closing the stream")
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

		//TODO: check for restricted-xml
		switch token.(type) {
		case xml.StartElement:
			// Processed after the switch
		case xml.EndElement:
			endElem := token.(xml.EndElement)
			if endElem.Name.Space == xmppcore.JabberStreamsNS && endElem.Name.Local == "stream" {
				if !cl.closingStream {
					cl.closingStream = true
					logrus.WithFields(logrus.Fields{"stream": cl.streamID}).
						Info("Client closed the stream. Disconnecting client....")
					//TODO: should we send a reply or simply close the connection?
					cl.conn.Write([]byte("</stream:stream>"))
					continue
				}
				cl.conn.Close()
				break mainloop
			}
			logrus.WithFields(logrus.Fields{"stream": cl.streamID}).
				Errorf("Unexpected EndElement: %#v", endElem)
			panic(endElem)
		case xml.ProcInst:
			procInst := token.(xml.ProcInst)
			if procInst.Target != "xml" {
				logrus.WithFields(logrus.Fields{"stream": cl.streamID}).
					Errorf("Unexpected processing instruction: %#v", procInst)
				continue
			}
			// Check XML version, encoding?
			continue
		default:
			logrus.Warnf("%#v", token)
			continue
		}

		if cl.closingStream {
			cl.xmlDecoder.Skip()
			continue
		}

		startElem := token.(xml.StartElement)

		switch startElem.Name.Space + " " + startElem.Name.Local {
		case xmppcore.StreamStreamElementName:
			if srv.handleClientStreamOpen(cl, &startElem) {
				continue
			}
			cl.closingStream = true
			cl.conn.Write([]byte("</stream:stream>"))
			//TODO: graceful disconnection (wait until the client close the stream)
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
		case xmppim.ClientMessageElementName:
			srv.handleClientMessage(cl, &startElem)
			continue
		default:
			logrus.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid}).
				Warn("unexpected XMPP stanza: ", startElem.Name)
			cl.xmlDecoder.Skip()
			continue
		}
	}
}

func (srv *Server) notifyClientSystemShutdown(cl *Client) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Got panic while sending system-shutdown notification: %#v", r)
		}
	}()
	cl.closingStream = true
	cl.conn.Write([]byte(`<stream:error><system-shutdown xmlns='urn:ietf:params:xml:ns:xmpp-streams'/></stream:error>\n` +
		`</stream:stream>`))
}

func (srv *Server) handleClientStreamOpen(cl *Client, startElem *xml.StartElement) bool {
	var toAttr, fromAttr string
	for _, attr := range startElem.Attr {
		switch attr.Name.Local {
		case "to":
			toAttr = attr.Value
		case "from":
			fromAttr = attr.Value
		}
	}

	toJID, err := xmppcore.ParseJID(toAttr)
	if err != nil {
		panic("TODO")
	}
	fromJID, err := xmppcore.ParseJID(fromAttr)
	if err != nil {
		panic("TODO")
	}

	if !toJID.Equals(srv.jid) {
		resultXML, err := xml.Marshal(&xmppcore.StreamError{
			Condition: xmppcore.StreamErrorConditionHostUnknown,
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
		return false
	}
	if !fromJID.IsEmpty() && fromJID.Domain != srv.jid.Domain {
		resultXML, err := xml.Marshal(&xmppcore.StreamError{
			Condition: xmppcore.StreamErrorConditionInvalidFrom,
		})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(resultXML)
		return false
	}

	var featuresXML []byte
	if cl.authenticated {
		featuresXML, err = xml.Marshal(&xmppcore.AuthenticatedStreamFeatures{})
		if err != nil {
			panic(err)
		}
	} else {
		//TODO: get features from the config and mods
		var mechanisms []string
		if srv.saslPlainAuthVerifier != nil {
			mechanisms = append(mechanisms, "PLAIN")
		}
		featuresXML, err = xml.Marshal(&xmppcore.NegotiationStreamFeatures{
			Mechanisms: &xmppcore.SASLMechanisms{
				Mechanism: mechanisms,
			},
		})
		if err != nil {
			panic(err)
		}
	}

	//TODO: include 'to' if 'from' was provided
	_, err = fmt.Fprintf(cl.conn, xml.Header+
		"<stream:stream from='%s' xmlns='%s'"+
		" id='%s' xml:lang='en'"+
		" xmlns:stream='%s' version='1.0'>\n"+
		string(featuresXML)+"\n",
		xmlEscape(srv.jid.FullString()), xmppcore.JabberClientNS,
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

func (srv *Server) generateSessionID() (string, error) {
	sid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	sidEncd := base64.RawURLEncoding.EncodeToString(sid[:])
	return string(sidEncd), nil
}

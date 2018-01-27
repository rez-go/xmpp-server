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

	"sandbox/gobber/cmd/gobber-server/oauth"
	"sandbox/gobber/pkg/xmppcore"
	"sandbox/gobber/pkg/xmppim"
)

//TODO: use locking to ensure that there will be one write into
// the connection.

type Server struct {
	DoneCh chan bool

	name   string
	domain string //TODO: support multiple

	plainAuthenticatorFunc SASLPlainAuthenticatorFunc

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
		DoneCh:                 make(chan bool),
		name:                   cfg.Name,
		domain:                 cfg.Domain, //TODO: normalize
		plainAuthenticatorFunc: oauth.Authenticate,
		stopCh:                 make(chan bool),
		listener:               listener,
		clients:                make(map[string]*Client),
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
			// Clean disconnection
			if err == io.EOF {
				logrus.WithFields(logrus.Fields{"stream": cl.streamID}).
					Info("Client disconnected")
				break mainloop
			}
			// Un-clean disconnection (the connection is closed while
			// the stream is still open)
			if xmlErr, _ := err.(*xml.SyntaxError); xmlErr != nil {
				if xmlErr.Line == 1 && xmlErr.Msg == "unexpected EOF" {
					logrus.WithFields(logrus.Fields{"stream": cl.streamID}).
						Info("Client disconnected without closing the stream")
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
				logrus.WithFields(logrus.Fields{"stream": cl.streamID}).
					Info("Client closed the stream. Closing connection....")
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
	authOK, err := srv.plainAuthenticatorFunc(authSegments[1], authSegments[2])
	if err != nil {
		panic(err)
	}
	if authOK {
		authRespXML, err := xml.Marshal(&xmppcore.SASLSuccess{})
		if err != nil {
			panic(err)
		}
		cl.conn.Write(authRespXML)
		cl.negotiationState = 2
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

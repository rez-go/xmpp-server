package main

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/exavolt/go-xmpplib/xmppcore"
	"github.com/exavolt/go-xmpplib/xmppim"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/exavolt/xmpp-server/cmd/xmpp-server/jwt"
)

//NOTE: it seems that we don't need to acquire lock to net.Conn to
// prevent multiple goroutines from messing up the network stream.
// https://stackoverflow.com/questions/38565654/golang-net-conn-write-in-parallel

type Server struct {
	DoneCh chan bool

	name         string
	jid          xmppcore.JID
	groupsDomain string

	saslPlainAuthVerifier SASLPlainAuthVerifier

	startTime time.Time
	stopCh    chan bool
	stopState int

	netListener          net.Listener
	negotiatingClients   map[string]*Client            // key is streamid
	authenticatedClients map[string]map[string]*Client // key is local:resource
	clientsMutex         sync.RWMutex
	clientsWaitGroup     sync.WaitGroup

	//userClientMessageHandler  UserClientMessageHandler
	//groupClientMessageHandler GroupClientMessageHandler
}

func New(
	cfg *Config,
) (*Server, error) {
	if cfg == nil {
		return nil, nil
	}
	netListener, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		return nil, err
	}
	saslPlainAuthVerifier := &jwt.SASLPlainAuthVerifier{}
	srv := &Server{
		DoneCh:                make(chan bool),
		name:                  cfg.Name,
		jid:                   xmppcore.JID{Domain: cfg.Domain}, //TODO: normalize
		groupsDomain:          "groups." + cfg.Domain,
		saslPlainAuthVerifier: saslPlainAuthVerifier,
		stopCh:                make(chan bool),
		netListener:           netListener,
		negotiatingClients:    make(map[string]*Client),
		authenticatedClients:  make(map[string]map[string]*Client),
	}
	return srv, nil
}

// func (srv *Server) WithUserClientMessageHandler(userClientMessageHandler UserClientMessageHandler) *Server {
// 	srv.userClientMessageHandler = userClientMessageHandler // mutex-lock?
// 	return srv
// }

// func (srv *Server) WithGroupClientMessageHandler(groupClientMessageHandler GroupClientMessageHandler) *Server {
// 	srv.groupClientMessageHandler = groupClientMessageHandler // mutex-lock?
// 	return srv
// }

func (srv *Server) Domain() string       { return srv.jid.Domain }
func (srv *Server) GroupsDomain() string { return srv.groupsDomain }

func (srv *Server) Serve() {
	defer func() {
		srv.stopState = 2
		srv.DoneCh <- true
	}()

	srv.startTime = time.Now()
	go srv.listen()
	log.Infof("Ready to accept connections")

mainloop:
	for {
		select {
		case <-srv.stopCh:
			close(srv.stopCh)
			break mainloop
		}
	}

	// Stopping
	log.Infof("Stopping after %s uptime...", srv.Uptime())
	srv.stopState = 1
	srv.netListener.Close()

	srv.clientsMutex.RLock()
	for _, cl := range srv.negotiatingClients {
		srv.notifyClientSystemShutdown(cl)
	}
	for _, ucl := range srv.authenticatedClients {
		for _, cl := range ucl {
			srv.notifyClientSystemShutdown(cl)
		}
	}
	srv.clientsMutex.RUnlock()

	srv.clientsWaitGroup.Wait()

	log.Info("Cleanly stopped")
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
		conn, err := srv.netListener.Accept()
		if err != nil {
			if srv.stopState == 0 {
				log.Error("Listener error: ", err)
			}
			break
		}
		if conn == nil {
			continue
		}

		cl, err := srv.newClient(conn)
		if err != nil {
			log.Error("Unable to create client: ", err)
			continue
		}

		log.WithFields(logrus.Fields{"stream": cl.streamID}).Info("Client connected")
		srv.clientsWaitGroup.Add(1)
		go srv.serveClient(cl)
	}
}

func (srv *Server) newClient(conn net.Conn) (*Client, error) {
	if conn == nil {
		return nil, nil
	}
	streamID, err := srv.generateStreamID()
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate session id")
	}
	cl := &Client{
		conn:       conn,
		streamID:   streamID,
		xmlDecoder: xml.NewDecoder(conn), //TODO: is there a way to limit the decoder's buffer size?
		jid:        xmppcore.JID{Domain: srv.jid.Domain},
	}
	srv.clientsMutex.Lock()
	srv.negotiatingClients[cl.streamID] = cl
	srv.clientsMutex.Unlock()
	return cl, nil
}

func (srv *Server) serveClient(cl *Client) {
	defer func() {
		if cl.conn != nil {
			log.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid}).
				Info("Closing client connection")
			cl.conn.Close()
			cl.conn = nil
		} else {
			log.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid}).
				Info("Client disconnected")
		}

		srv.clientsMutex.Lock()
		if cl.authenticated {
			userClients := srv.authenticatedClients[cl.jid.Local]
			if userClients != nil {
				delete(userClients, cl.jid.Resource)
			}
		} else {
			delete(srv.negotiatingClients, cl.streamID)
		}
		srv.clientsMutex.Unlock()

		srv.clientsWaitGroup.Done()
	}()

mainloop:
	for {
		token, err := cl.xmlDecoder.Token()
		if err != nil {
			// Clean disconnection
			if err == io.EOF {
				log.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid}).
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
					log.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid}).
						Info("Client connection closed without closing the stream")
					break mainloop
				}
			}
			log.WithFields(logrus.Fields{"stream": cl.streamID}).
				Errorf("Unexpected error: %#v", err)
			break mainloop
		}
		if token == nil {
			log.WithFields(logrus.Fields{"stream": cl.streamID}).
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
					log.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid}).
						Info("Client closed the stream. Disconnecting client....")
					//TODO: should we send a reply or simply close the connection?
					cl.conn.Write([]byte("</stream:stream>"))
					continue
				}
				cl.conn.Close()
				break mainloop
			}
			log.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid}).
				Errorf("Unexpected EndElement: %#v", endElem)
			panic(endElem)
		case xml.ProcInst:
			procInst := token.(xml.ProcInst)
			if procInst.Target != "xml" {
				log.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid}).
					Errorf("Unexpected processing instruction: %#v", procInst)
				continue
			}
			// Check XML version, encoding?
			continue
		default:
			log.Warnf("%#v", token)
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
			if !cl.authenticated {
				srv.handleClientSASLAuth(cl, &startElem)
				continue
			}
		case xmppcore.ClientIQElementName:
			if cl.authenticated {
				srv.handleClientIQ(cl, &startElem)
				continue
			}
		case xmppim.ClientPresenceElementName:
			if cl.authenticated {
				srv.handleClientPresence(cl, &startElem)
				continue
			}
		case xmppim.ClientMessageElementName:
			if cl.authenticated {
				srv.handleClientMessage(cl, &startElem)
				continue
			}
		}
		log.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid}).
			Warn("Unexpected XMPP stanza: ", startElem.Name)
		cl.xmlDecoder.Skip()
		continue
	}
}

func (srv *Server) notifyClientSystemShutdown(cl *Client) {
	defer func() {
		if r := recover(); r != nil {
			log.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid}).
				Errorf("Got panic while sending system-shutdown notification: %#v", r)
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
		xmlEscapeString(srv.jid.FullString()), xmppcore.JabberClientNS,
		xmlEscapeString(cl.streamID), xmppcore.JabberStreamsNS)
	if err != nil {
		panic(err)
	}

	return true
}

func (srv *Server) finishClientNegotiation(cl *Client) {
	if cl.jid.Local == "" || cl.jid.Resource == "" {
		panic("unexpected condition")
	}
	newStreamID, err := srv.generateStreamID()
	if err != nil {
		panic(err)
	}
	srv.clientsMutex.Lock()
	delete(srv.negotiatingClients, cl.streamID)
	oldStreamID := cl.streamID
	cl.streamID = newStreamID
	if srv.authenticatedClients[cl.jid.Local] == nil {
		srv.authenticatedClients[cl.jid.Local] = make(map[string]*Client)
	}
	srv.authenticatedClients[cl.jid.Local][cl.jid.Resource] = cl
	srv.clientsMutex.Unlock()
	log.WithFields(logrus.Fields{"stream": cl.streamID, "jid": cl.jid}).
		Infof("Negotiation completed: %s => %s", oldStreamID, cl.streamID)
}

func (srv *Server) generateStreamID() (string, error) {
	idRaw, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	idEncd := base64.RawURLEncoding.EncodeToString(idRaw[:])
	return string(idEncd), nil
}

package xmppcore

import (
	"strings"
)

// https://tools.ietf.org/html/rfc7622

func ParseJID(jidString string) (JID, error) {
	var jid JID
	//TODO: is empty string a JID? no?
	if jidString == "" {
		return jid, nil
	}
	var bareStr string

	slashIdx := strings.Index(jidString, "/")
	if slashIdx >= 0 {
		jid.Resource = jidString[slashIdx+1:]
		bareStr = jidString[:slashIdx]
	} else {
		bareStr = jidString
	}

	atIdx := strings.Index(bareStr, "@")
	if atIdx >= 0 {
		jid.Domain = bareStr[atIdx+1:]
		jid.Local = bareStr[:atIdx]
	} else {
		jid.Domain = bareStr
	}

	return jid, nil
}

//TODO: keep things normalized
//TODO: an utility methods to make it easier to put
// this into XML
type JID struct {
	Local    string
	Domain   string
	Resource string
}

// IsEmpty returns true if all parts are empty.
func (jid JID) IsEmpty() bool {
	return jid.Local == "" && jid.Domain == "" && jid.Resource == ""
}

// Bare returns the "bare JID" string.
//
// RFC 6120  1.4:
// The term "bare JID" refers to an XMPP address of the form
// <localpart@domainpart> (for an account at a server) or of the form
// <domainpart> (for a server).
func (jid JID) Bare() string {
	if jid.Local != "" {
		return jid.Local + "@" + jid.Domain
	}
	return jid.Domain
}

// IsBare returns true if the domain is not empty and the resource is empty.
func (jid JID) IsBare() bool {
	if jid.Domain == "" {
		return false
	}
	if jid.Resource != "" {
		return false
	}
	return true
}

// Full returns the "full JID" string.
//
// RFC 6120  1.4
// The term "full JID" refers to an XMPP address of the form
// <localpart@domainpart/resourcepart> (for a particular authorized client
// or device associated with an account) or of the form
// <domainpart/resourcepart> (for a particular resource or script associated
// with a server).
func (jid JID) Full() string {
	if jid.Resource != "" {
		return jid.Bare() + "/" + jid.Resource
	}
	return jid.Bare()
}

// IsFull returns true if both domain and resource are not empty.
func (jid JID) IsFull() bool {
	if jid.Domain == "" || jid.Resource == "" {
		return false
	}
	return true
}

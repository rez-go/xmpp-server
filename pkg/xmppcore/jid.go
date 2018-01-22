package xmppcore

//TODO: keep things normalized
//TODO: an utility methods to make it easier to put
// this into XML
type JID struct {
	Local    string
	Domain   string
	Resource string
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

// Full returns the "full JID" string.
//
// RFC 6120  1.4
// The term "full JID" refers to an XMPP address of the form
// <localpart@domainpart/resourcepart> (for a particular authorized client
// or device associated with an account) or of the form
// <domainpart/resourcepart> (for a particular resource or script associated
// with a server).
func (jid JID) Full() string {
	return jid.Bare() + "/" + jid.Resource
}

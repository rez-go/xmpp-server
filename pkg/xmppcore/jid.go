package xmppcore

import (
	"encoding/xml"
	"strings"
)

// Related specs:
// https://tools.ietf.org/html/rfc7622
// https://xmpp.org/extensions/xep-0106.html

// ParseJID parses a string.
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

	jid.Domain = strings.TrimSuffix(jid.Domain, ".")

	return jid, nil
}

// JID represents a JID.
//
//TODO: keep things normalized
//TODO: an utility methods to make it easier to put
// this into XML
type JID struct {
	Local    string
	Domain   string
	Resource string
}

// Equals returns true if the other JID is essentially the same.
func (jid JID) Equals(other JID) bool {
	return jid.Local == other.Local &&
		jid.Domain == other.Domain &&
		jid.Resource == other.Resource
}

// String returns the string representation of the JID.
//
// Use FullString() to consistently get "full JID" string representation.
func (jid JID) String() string {
	return jid.FullString()
}

// IsEmpty returns true if all parts are empty.
func (jid JID) IsEmpty() bool {
	return jid.Local == "" && jid.Domain == "" && jid.Resource == ""
}

// BareString returns the "bare JID" string.
//
// RFC 6120  1.4:
// The term "bare JID" refers to an XMPP address of the form
// <localpart@domainpart> (for an account at a server) or of the form
// <domainpart> (for a server).
func (jid JID) BareString() string {
	if jid.Local != "" {
		return jid.Local + "@" + jid.Domain
	}
	return jid.Domain
}

// BareCopy returns a copy of the JID with resource set to empty
func (jid JID) BareCopy() JID {
	return JID{Local: jid.Local, Domain: jid.Domain}
}

// BareCopyPtr returns a pointer to bare copy of a JID.
func (jid JID) BareCopyPtr() *JID {
	v := jid.BareCopy()
	return &v
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

// FullString returns the "full JID" string.
//
// RFC 6120  1.4
// The term "full JID" refers to an XMPP address of the form
// <localpart@domainpart/resourcepart> (for a particular authorized client
// or device associated with an account) or of the form
// <domainpart/resourcepart> (for a particular resource or script associated
// with a server).
func (jid JID) FullString() string {
	if jid.Resource != "" {
		return jid.BareString() + "/" + jid.Resource
	}
	return jid.BareString()
}

// IsFull returns true if both domain and resource are not empty.
func (jid JID) IsFull() bool {
	if jid.Domain == "" || jid.Resource == "" {
		return false
	}
	return true
}

// MarshalXML satisfies encoding/xml.Marshaler interface
func (jid JID) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	if err := encoder.EncodeToken(start); err != nil {
		return err
	}
	// Can we get the tag to determine bare or full?
	if err := encoder.EncodeToken(xml.CharData(jid.FullString())); err != nil {
		return err
	}
	if err := encoder.EncodeToken(start.End()); err != nil {
		return err
	}
	return encoder.Flush()
}

// UnmarshalXML satisfies encoding/xml.Unmarshaler interface
func (jid *JID) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	data := struct {
		CharData string `xml:",chardata"`
	}{}
	if err := decoder.DecodeElement(&data, &start); err != nil {
		return err
	}
	jid1, err := ParseJID(data.CharData)
	if err != nil {
		return err
	}
	*jid = jid1
	return nil
}

// MarshalXMLAttr satisfies encoding/xml.MarshalerAttr interface
func (jid JID) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: jid.FullString()}, nil
}

// UnmarshalXMLAttr satisfies encoding/xml.UnmarshalerAttr interface
func (jid *JID) UnmarshalXMLAttr(attr xml.Attr) error {
	if attr.Value == "" {
		return nil
	}
	jid1, err := ParseJID(attr.Value)
	if err != nil {
		return err
	}
	*jid = jid1
	return nil
}

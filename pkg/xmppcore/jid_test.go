package xmppcore

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJIDEmpty(t *testing.T) {
	jid := JID{}
	assert.Equal(t, "", jid.Local)
	assert.Equal(t, "", jid.Domain)
	assert.Equal(t, "", jid.Resource)
	assert.Equal(t, "", jid.BareString())
	assert.Equal(t, "", jid.FullString())
	assert.Equal(t, "", jid.String())
	assert.True(t, jid.IsEmpty())
	assert.False(t, jid.IsBare())
	assert.False(t, jid.IsFull())
}

func TestJIDDomain(t *testing.T) {
	jid := JID{Domain: "localhost"}
	assert.Equal(t, "localhost", jid.BareString())
	assert.Equal(t, "localhost", jid.FullString())
	assert.Equal(t, "localhost", jid.String())
	assert.False(t, jid.IsEmpty())
	assert.True(t, jid.IsBare())
	assert.False(t, jid.IsFull())
}

func TestJIDBare(t *testing.T) {
	jid := JID{Local: "user", Domain: "localhost"}
	assert.Equal(t, "user@localhost", jid.BareString())
	assert.Equal(t, "user@localhost", jid.FullString())
	assert.Equal(t, "user@localhost", jid.String())
	assert.False(t, jid.IsEmpty())
	assert.True(t, jid.IsBare())
	assert.False(t, jid.IsFull())
}

func TestJIDFull(t *testing.T) {
	jid := JID{Local: "user", Domain: "localhost", Resource: "PC"}
	assert.Equal(t, "user@localhost", jid.BareString())
	assert.Equal(t, "user@localhost/PC", jid.FullString())
	assert.Equal(t, "user@localhost/PC", jid.String())
	assert.False(t, jid.IsEmpty())
	assert.False(t, jid.IsBare())
	assert.True(t, jid.IsFull())
}

func TestJIDBareCopy(t *testing.T) {
	jid := JID{Local: "juliet", Domain: "example.com", Resource: "foobar"}
	assert.True(t, jid.IsFull())
	assert.False(t, jid.IsBare())
	jid1 := jid.BareCopy()
	assert.False(t, jid1.IsFull())
	assert.True(t, jid1.IsBare())
}

func TestJIDBareCopyPtr(t *testing.T) {
	jid := JID{Local: "juliet", Domain: "example.com", Resource: "foobar"}
	assert.True(t, jid.IsFull())
	assert.False(t, jid.IsBare())
	jid1 := jid.BareCopyPtr()
	assert.False(t, jid1.IsFull())
	assert.True(t, jid1.IsBare())
}

func TestJIDEqual(t *testing.T) {
	jid := JID{Local: "juliet", Domain: "example.com", Resource: "foobar"}
	jid1 := JID{Local: "juliet", Domain: "example.com", Resource: "foobar"}
	assert.True(t, jid.Equals(jid1))
}

func TestJIDNotEqual(t *testing.T) {
	jid := JID{Local: "juliet", Domain: "example.com", Resource: "foobar"}
	jid1 := jid
	jid1.Resource = ""
	assert.False(t, jid.Equals(jid1))
}

func TestParseJID(t *testing.T) {
	//TODO: take vectors from RFC 7622 section 3.5
	testData := []struct {
		str string
		jid JID
		err error
	}{
		{"", JID{}, nil},
		{"juliet@example.com", JID{Local: "juliet", Domain: "example.com"}, nil},
		{"juliet@example.com/foo", JID{Local: "juliet", Domain: "example.com", Resource: "foo"}, nil},
		{"example.com", JID{Domain: "example.com"}, nil},
		{"example.com/foobar", JID{Domain: "example.com", Resource: "foobar"}, nil},
		{"a.example.com/b@example.net", JID{Domain: "a.example.com", Resource: "b@example.net"}, nil},

		{"example.com./foobar", JID{Domain: "example.com", Resource: "foobar"}, nil},
	}

	for _, data := range testData {
		jid, err := ParseJID(data.str)
		assert.Equal(t, data.jid.Local, jid.Local)
		assert.Equal(t, data.jid.Domain, jid.Domain)
		assert.Equal(t, data.jid.Resource, jid.Resource)
		assert.Equal(t, data.err, err)
	}
}

func TestJIDMarshal(t *testing.T) {
	jid := JID{Local: "juliet", Domain: "example.com", Resource: "foo"}
	buf, err := xml.Marshal(jid)
	assert.Nil(t, err)
	assert.Equal(t, []byte("<JID>juliet@example.com/foo</JID>"), buf)
}

func TestJIDUnmarshal(t *testing.T) {
	var jid JID
	err := xml.Unmarshal([]byte("<any>juliet@example.com/foo</any>"), &jid)
	assert.Nil(t, err)
	assert.Equal(t, JID{Local: "juliet", Domain: "example.com", Resource: "foo"}, jid)
}

type testJIDAttr struct {
	XMLName xml.Name `xml:"test"`
	JID     JID      `xml:"jid,attr"`
}

func TestJIDMarshalAttr(t *testing.T) {
	v := testJIDAttr{JID: JID{Local: "juliet", Domain: "example.com", Resource: "foo"}}
	buf, err := xml.Marshal(v)
	assert.Nil(t, err)
	assert.Equal(t, []byte(`<test jid="juliet@example.com/foo"></test>`), buf)
}

func TestJIDUnmarshalAttr(t *testing.T) {
	var v testJIDAttr
	err := xml.Unmarshal([]byte("<test jid='juliet@example.com/foo' />"), &v)
	assert.Nil(t, err)
	assert.Equal(t, JID{Local: "juliet", Domain: "example.com", Resource: "foo"}, v.JID)
}

func TestJIDUnmarshalAttrEmptyValue(t *testing.T) {
	var jid JID
	err := jid.UnmarshalXMLAttr(xml.Attr{Name: xml.Name{Local: "jid"}, Value: ""})
	assert.Nil(t, err)
	assert.True(t, jid.IsEmpty())
}

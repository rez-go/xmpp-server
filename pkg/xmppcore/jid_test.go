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
	assert.Equal(t, "", jid.Bare())
	assert.Equal(t, "", jid.Full())
	assert.True(t, jid.IsEmpty())
	assert.False(t, jid.IsBare())
	assert.False(t, jid.IsFull())
}

func TestJIDDomain(t *testing.T) {
	jid := JID{Domain: "localhost"}
	assert.Equal(t, "localhost", jid.Bare())
	assert.Equal(t, "localhost", jid.Full())
	assert.False(t, jid.IsEmpty())
	assert.True(t, jid.IsBare())
	assert.False(t, jid.IsFull())
}

func TestJIDBare(t *testing.T) {
	jid := JID{Local: "user", Domain: "localhost"}
	assert.Equal(t, "user@localhost", jid.Bare())
	assert.Equal(t, "user@localhost", jid.Full())
	assert.False(t, jid.IsEmpty())
	assert.True(t, jid.IsBare())
	assert.False(t, jid.IsFull())
}

func TestJIDFull(t *testing.T) {
	jid := JID{Local: "user", Domain: "localhost", Resource: "PC"}
	assert.Equal(t, "user@localhost", jid.Bare())
	assert.Equal(t, "user@localhost/PC", jid.Full())
	assert.False(t, jid.IsEmpty())
	assert.False(t, jid.IsBare())
	assert.True(t, jid.IsFull())
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

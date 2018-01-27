package xmppcore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJIDEmpty(t *testing.T) {
	jid := JID{}
	assert.Equal(t, "", jid.Local)
	assert.Equal(t, "", jid.Domain)
	assert.Equal(t, "", jid.Resource)
	assert.Equal(t, "", jid.Bare())
	assert.Equal(t, "/", jid.Full())
}

func TestJIDDomain(t *testing.T) {
	jid := JID{Domain: "localhost"}
	assert.Equal(t, "localhost", jid.Bare())
	assert.Equal(t, "localhost/", jid.Full())
}

func TestJIDBare(t *testing.T) {
	jid := JID{Local: "user", Domain: "localhost"}
	assert.Equal(t, "user@localhost", jid.Bare())
	assert.Equal(t, "user@localhost/", jid.Full())
}

func TestJIDFull(t *testing.T) {
	jid := JID{Local: "user", Domain: "localhost", Resource: "PC"}
	assert.Equal(t, "user@localhost", jid.Bare())
	assert.Equal(t, "user@localhost/PC", jid.Full())
}

package xmppcore

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStanzaErrorConditionFeatureNotImplementedEncoding(t *testing.T) {
	def := StanzaErrorConditionFeatureNotImplemented
	xmlBuf, err := xml.Marshal(def)
	assert.Nil(t, err)
	assert.Equal(t,
		[]byte(`<feature-not-implemented xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></feature-not-implemented>`),
		xmlBuf)
}

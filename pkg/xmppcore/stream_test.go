package xmppcore

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamErrorConditionBadFormatEncoding(t *testing.T) {
	def := StreamErrorConditionBadFormat
	xmlBuf, err := xml.Marshal(def)
	assert.Nil(t, err)
	assert.Equal(t,
		[]byte(`<bad-format xmlns="urn:ietf:params:xml:ns:xmpp-streams"></bad-format>`),
		xmlBuf)
}

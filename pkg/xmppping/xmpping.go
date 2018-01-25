// XEP-0199
package xmppping

import (
	"encoding/xml"
)

const NS = "urn:xmpp:ping"

const ElementName = NS + " ping"

type Ping struct {
	XMLName xml.Name `xml:"urn:xmpp:ping ping"`
}

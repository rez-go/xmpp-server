package main

import (
	"bytes"
	"encoding/xml"
)

func xmlEscapeString(s string) string {
	var b bytes.Buffer
	xml.Escape(&b, []byte(s))
	return b.String()
}

package xmppcore

import (
	"encoding/xml"
)

// RFC 6120  6  SASL Negotiation

const SASLNS = "urn:ietf:params:xml:ns:xmpp-sasl"

const SASLAuthElementName = SASLNS + " auth"

// RFC 6120  6.4.1
type SASLMechanisms struct {
	XMLName   xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl mechanisms"`
	Mechanism []string `xml:"mechanism"`
}

// RFC 6120  6.4.2
type SASLAuth struct {
	XMLName   xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl auth"`
	Mechanism string   `xml:"mechanism,attr"`
	CharData  string   `xml:",chardata"`
}

//
// RFC 6120  6.4.6
type SASLSuccess struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl success"`
}

type SASLFailure struct {
	XMLName   xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl failure"`
	Condition SASLFailureCondition
	Text      string `xml:"text"`
}

type SASLFailureCondition struct {
	XMLName xml.Name // Deliberately un-tagged
}

// RFC 6120 section 6.5
var (
	SASLFailureConditionAborted              = saslFailureCondition("aborted")
	SASLFailureConditionAccountDisabled      = saslFailureCondition("account-disabled")
	SASLFailureConditionCredentialsExpired   = saslFailureCondition("credentials-expired")
	SASLFailureConditionEncryptionRequired   = saslFailureCondition("encryption-required")
	SASLFailureConditionIncorrectEncoding    = saslFailureCondition("incorrect-encoding")
	SASLFailureConditionInvalidAuthzid       = saslFailureCondition("invalid-authzid")
	SASLFailureConditionInvalidMechanism     = saslFailureCondition("invalid-mechanism")
	SASLFailureConditionMalformedRequest     = saslFailureCondition("malformed-request")
	SASLFailureConditionMechanismTooWeak     = saslFailureCondition("mechanism-too-weak")
	SASLFailureConditionNotAuthorized        = saslFailureCondition("not-authorized")
	SASLFailureConditionTemporaryAuthFailure = saslFailureCondition("temporary-auth-failure")
)

func saslFailureCondition(local string) SASLFailureCondition {
	return SASLFailureCondition{XMLName: xml.Name{Space: SASLNS, Local: local}}
}

package jwt

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

type SASLPlainAuthVerifier struct {
}

func (handler *SASLPlainAuthVerifier) VerifySASLPlainAuth(
	username, jwtBytes []byte,
) (localpart string, resource string, success bool, err error) {
	//TODO: user JWT parser lib
	jwtStr := string(jwtBytes)
	jwtParts := strings.SplitN(jwtStr, ".", 3)
	if len(jwtParts) != 3 {
		return "", "", false, errors.New("invalid password format")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(jwtParts[1])
	if err != nil {
		return "", "", false, err
	}
	var claimMap map[string]interface{}
	err = json.Unmarshal(payloadJSON, &claimMap)
	if err != nil {
		return "", "", false, err
	}
	var userIDStr string
	if subVal, ok := claimMap["sub"]; ok {
		if subStr, ok := subVal.(string); ok {
			userIDStr = subStr
		}
	}
	if userIDStr == "" {
		return "", "", false, errors.New("sub is missing")
	}
	return userIDStr, "", true, nil
}

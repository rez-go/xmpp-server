package oauth

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
)

// Only the password_grant is currently implemented.

type Authenticator struct {
	TokenEndpoint string
	ClientID      string
	ClientSecret  string
}

func (handler *Authenticator) VerifySASLPlainAuth(
	username, password []byte,
) (localpart string, resource string, success bool, err error) {
	client := &http.Client{}

	reqData := url.Values{}
	reqData.Set("grant_type", "password")
	reqData.Set("username", string(username))
	reqData.Set("password", string(password))
	reqBody := bytes.NewBuffer([]byte(reqData.Encode()))

	req, err := http.NewRequest("POST", handler.TokenEndpoint, reqBody)
	req.SetBasicAuth(handler.ClientID, handler.ClientSecret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return "", "", false, errors.New("authentication gateway error")
	}
	if resp.StatusCode >= 400 {
		return "", "", false, errors.New("authentication module error")
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", false, nil
	}
	return string(username), "", true, nil
}

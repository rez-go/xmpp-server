package oauth

import (
	"bytes"
	"net/http"
	"net/url"
)

func Authenticate(username, password []byte) (bool, error) {
	client := &http.Client{}

	reqData := url.Values{}
	reqData.Set("grant_type", "password")
	reqData.Set("username", string(username))
	reqData.Set("password", string(password))
	reqBody := bytes.NewBuffer([]byte(reqData.Encode()))

	req, err := http.NewRequest("POST", "http://localhost:8080/oauth/token", reqBody)
	req.Header.Add("Authorization", "Basic .") //TODO: client creds
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, nil
}

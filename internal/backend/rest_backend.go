package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/LNOpenMetrics/lnmetrics.utils/hash/sha256"
	"github.com/LNOpenMetrics/lnmetrics.utils/log"
)

type RestBackend struct {
	BaseUrl *string
	//token   *string
}

// create a new instance of rest backend
func NewRestBackend(baseUrl string) Backend {
	return &RestBackend{BaseUrl: &baseUrl}
}

// verify the message with a c-lightning node over the rest API.
func (instance *RestBackend) VerifyMessage(message *string, signature *string, pubkey *string) (bool, error) {
	restMethodName := "checkmessage"
	toVerify := sha256.SHA256(message)
	// call the rest apu and pass the toVerify and the signature received, with
	// the pub key information.
	postBody, _ := json.Marshal(map[string]string{
		"message": toVerify,
		"zbase":   *signature,
		"pubkey":  *pubkey,
	})

	log.GetInstance().Debug(fmt.Sprintf("verify message request: %s", string(postBody)))

	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post(strings.Join([]string{*instance.BaseUrl, restMethodName}, "/"), "application/json", responseBody)
	//Handle Error
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var response CheckMessageResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return false, err
	}

	return response.Verified, nil
}

// Rest response wrapper
type CheckMessageResponse struct {
	Verified bool `json:"verified"`
}

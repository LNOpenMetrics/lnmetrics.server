package backend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
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
	log.GetInstance().Info(fmt.Sprintf("Hash of the payload received: %s", toVerify))
	// call the rest apu and pass the toVerify and the signature received, with
	// the pub key information.

	data := url.Values{}
	data.Set("message", toVerify)
	data.Set("zbase", *signature)
	data.Set("pubkey", *pubkey)

	postData := data.Encode()
	url := strings.Join([]string{*instance.BaseUrl, restMethodName}, "/")

	log.GetInstance().Debug(fmt.Sprintf("rest backend url: %s", url))
	log.GetInstance().Debug(fmt.Sprintf("verify message request: %s", string(postData)))

	client := &http.Client{}
	request, err := http.NewRequest(http.MethodPost, url, strings.NewReader(postData))
	if err != nil {
		log.GetInstance().Error(fmt.Sprintf("Backend Error: %s", err))
		return false, err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Content-Length", strconv.Itoa(len(postData)))

	resp, err := client.Do(request)
	if err != nil {
		log.GetInstance().Error(fmt.Sprintf("Error from backend: %s", err))
		return false, nil
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

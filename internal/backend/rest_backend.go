package backend

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
	return false, nil
}

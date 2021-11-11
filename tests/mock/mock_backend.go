package mock

type MockBackend struct{}

func (instance *MockBackend) VerifyMessage(message *string, signature *string, pubkey *string) (bool, error) {
	return true, nil
}

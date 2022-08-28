package backend

// Backend Interface that the server
// uses to verify and give the access to
// a lightning node.
type Backend interface {
	// VerifyMessage Method to very a message signed with a lightning node
	// The method take the following parameters:
	// message: It is the original message as string
	// signature: It is the signature of the message
	// pubkey: It is the node pub key that signed the message
	// as result we can get an error if some I/O error happen, and a boolean value
	// that it is the result of the verify operation
	VerifyMessage(message *string, signature *string, pubkey *string) (bool, error)
}

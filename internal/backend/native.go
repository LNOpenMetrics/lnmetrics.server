package backend

import (
	"github.com/LNOpenMetrics/lnmetrics.utils/sign/ln"
)

type NativeBackend struct {
	signer *ln.LNSigner
}

func NewNativeBackend() *NativeBackend {
	return &NativeBackend{
		signer: ln.NewLNSigner(),
	}
}

func (self *NativeBackend) VerifyMessage(message *string, signature *string, pubkey *string) (bool, error) {
	return self.signer.VerifyMsg(pubkey, signature, message)
}

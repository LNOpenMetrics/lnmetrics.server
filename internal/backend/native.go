package backend

import (
	"github.com/LNOpenMetrics/lnmetrics.utils/hash/sha256"
	"github.com/LNOpenMetrics/lnmetrics.utils/log"
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
	toVerify := sha256.SHA256(message)
	verified, err := self.signer.VerifyMsg(pubkey, signature, &toVerify)
	if !verified {
		// sanity check to make sure that all work as expected
		log.GetInstance().Infof("Msg from %s failed to verify", *pubkey)
	}
	return verified, err
}

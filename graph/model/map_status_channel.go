package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
)

type StatusChannelMap map[string]*StatusChannel

func MarshalStatusChannelMap(t StatusChannelMap) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		b, _ := json.Marshal(t)
		w.Write(b)
	})
}

func UnmarshalStatusChannelMap(v interface{}) (StatusChannelMap, error) {
	value, ok := v.(StatusChannelMap)
	if !ok {
		return nil, errors.New(fmt.Sprintf("Failed to unmarshal ScalarType: #%v", v))
	}
	return value, nil
}

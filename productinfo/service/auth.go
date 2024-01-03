package main

import (
	"context"
	"encoding/base64"
)

type basicAuth struct {
	username string
	password string
}

func (b basicAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	auth := b.username + ":" + b.password
	enc := base64.StdEncoding.EncodeToString([]byte(auth))
	return map[string]string{
		"authorization": "Basic " + enc,
	}, nil
}

func (b basicAuth) RequireTransportSecurity() bool {
	return true
}

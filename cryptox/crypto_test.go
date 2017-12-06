// Copyright 2017 Aleksey Blinov. All rights reserved.

package cryptox

import (
	"crypto/ecdsa"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPKCS8PrivateKeyFromFile(t *testing.T) {
	s, err := PKCS8PrivateKeyFromFile("test_data/pk_valid.p8")
	assert.NoError(t, err)
	assert.IsType(t, &ecdsa.PrivateKey{}, s)
}

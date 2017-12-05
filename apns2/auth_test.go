// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/baobabus/go-apns/cryptox"
	"github.com/stretchr/testify/assert"
)

var (
	auth_test_jwtAsHeader = regexp.MustCompile("bearer [a-zA-Z0-9\\-_]+\\.[a-zA-Z0-9\\-_]+\\.[a-zA-Z0-9\\-_]+")
)

func TestJWTSignerDefaults(t *testing.T) {
	signingKey, err := cryptox.PKCS8PrivateKeyFromFile("../cryptox/test_data/pk_valid.p8")
	if err != nil {
		t.Fatal(err)
	}
	s := &JWTSigner{
		KeyID:      "ABC123DEFG",
		TeamID:     "DEF123GHIJ",
		SigningKey: signingKey,
	}
	now := time.Now()
	tk, err := s.GetToken()
	if err != nil {
		t.Fatal(err)
	}
	assert.Exactly(t, DefaultJWTSigningMethod, s.signingMethod)
	assert.Exactly(t, DefaultTokenLifeSpan, s.tokenLifeSpan)
	assert.True(t, tk.IssuedAt.Unix() - now.Unix() < 1)
	assert.Exactly(t, tk.IssuedAt.Add(DefaultTokenLifeSpan).Unix(), tk.ExpiresAt.Unix())
	assert.True(t, auth_test_jwtAsHeader.MatchString(tk.AsHeader))
}

func TestJWTSignerCustom(t *testing.T) {
	signingKey, err := cryptox.PKCS8PrivateKeyFromFile("../cryptox/test_data/pk_valid.p8")
	if err != nil {
		t.Fatal(err)
	}
	lifespan := time.Minute
	s := &JWTSigner{
		KeyID:         "ABC123DEFG",
		TeamID:        "DEF123GHIJ",
		SigningKey:    signingKey,
		TokenLifeSpan: lifespan,
	}
	now := time.Now()
	tk, err := s.GetToken()
	if err != nil {
		t.Fatal(err)
	}
	assert.Exactly(t, DefaultJWTSigningMethod, s.signingMethod)
	assert.Exactly(t, lifespan, s.tokenLifeSpan)
	assert.True(t, tk.IssuedAt.Unix() - now.Unix() < 1)
	assert.Exactly(t, tk.IssuedAt.Add(lifespan).Unix(), tk.ExpiresAt.Unix())
	assert.True(t, auth_test_jwtAsHeader.MatchString(tk.AsHeader))
}

func TestJWTSignerRefresh(t *testing.T) {
	signingKey, err := cryptox.PKCS8PrivateKeyFromFile("../cryptox/test_data/pk_valid.p8")
	if err != nil {
		t.Fatal(err)
	}
	lifespan := 250 * time.Microsecond
	s := &JWTSigner{
		KeyID:         "ABC123DEFG",
		TeamID:        "DEF123GHIJ",
		SigningKey:    signingKey,
		TokenLifeSpan: lifespan,
	}
	tk1, err := s.GetToken()
	if err != nil {
		t.Fatal(err)
	}
	tk2, err := s.GetToken()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, tk1, tk2)
	time.Sleep(lifespan)
	tk3, err := s.GetToken()
	if err != nil {
		t.Fatal(err)
	}
	assert.NotEqual(t, tk1, tk3)
	assert.True(t, tk1.IssuedAt.Before(tk3.IssuedAt))
	assert.True(t, tk1.ExpiresAt.Before(tk3.IssuedAt))
}

func TestJWTSignerSignRequest(t *testing.T) {
	signingKey, err := cryptox.PKCS8PrivateKeyFromFile("../cryptox/test_data/pk_valid.p8")
	if err != nil {
		t.Fatal(err)
	}
	s := &JWTSigner{
		KeyID:      "ABC123DEFG",
		TeamID:     "DEF123GHIJ",
		SigningKey: signingKey,
	}
	req, err := http.NewRequest("POST", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, len(req.Header))
	err = s.SignRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(req.Header))
	assert.Equal(t, 1, len(req.Header["Authorization"]))
	h := req.Header.Get("Authorization")
	assert.True(t, auth_test_jwtAsHeader.MatchString(h))
}

func TestNoSignerSignRequest(t *testing.T) {
	s := NoSigner
	req, err := http.NewRequest("POST", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, len(req.Header))
	err = s.SignRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, len(req.Header))
}

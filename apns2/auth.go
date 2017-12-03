// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"crypto/ecdsa"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// RequestSigner must be implemented by all APN service request signers.
// Provider token signing allows authenticating with APN service on per request
// basis, if needed.
type RequestSigner interface {

	// SignRequest gives the signer a chance to sign the request.
	// Any headers and the request body is guaranteed to have been
	// set up at this point.
	SignRequest(r *http.Request) error

}

// DefaultTokenLifeSpan specifies the time duration for which
// provier tokens are considered to be valid. At present APN service
// stops honoring authentication tokens that are older than 1 hour.
// Initial global default value allows 10 minutes of safety margin.
// If changed, any provider token authenticators created thereafter
// will use the new value.
var DefaultTokenLifeSpan = 50 * time.Minute

// DefaultJWTSigningMethod method for APN requests is ES256.
var DefaultJWTSigningMethod = jwt.SigningMethodES256

// Provider token-based signer that uses JSON Web Tokens to sign individual
// requests to APN service. It is safe to use in concurrent goroutines.
type JWTSigner struct {
	// A 10-character key identifier, obtained from Apple developer account.
	KeyID string

	// A 10-character Team ID, obtained from Apple developer account.
	TeamID string

	// Private key for signing generated tokens.
	SigningKey *ecdsa.PrivateKey

	// Method to use for signing generated tokens.
	SigningMethod *jwt.SigningMethodECDSA

	// The duration for which generated tokens are considered valid by apns2.
	// This is currently required to not exceed one hour.
	TokenLifeSpan time.Duration

	mu sync.Mutex
	// Last generated token. This should not be accessed directly.
	// Use GetToken() method, which may generated a new token
	// before returing it if needed.
	currentToken atomic.Value
	// SigningMethod or, if nil, DefaultSigningMethod
	signingMethod *jwt.SigningMethodECDSA
	tokenLifeSpan time.Duration
}

// JWT is an implementation of provider token in the form of
// Javascript Web Token, that can be written to HTTP authorization header.
// It is intended to remain immutable once created, and is safe to use
// in concurrent goroutines.
type JWT struct {
	IssuedAt  time.Time
	ExpiresAt time.Time
	JwtToken  *jwt.Token
	AsHeader  string
}

// SignRequest adds authorization header to the supplied request.
// The header is an encrypted JSON Web Token containing signer's credentials.
// The token is guaranteed to be valid at the time of the call.
func (s *JWTSigner) SignRequest(r *http.Request) error {
	t, err := s.GetToken()
	if err != nil {
		return err
	}
	r.Header.Set("authorization", t.AsHeader)
	return nil
}

// GetToken returns provider authentication token that is guaranteed
// to be valid at the time of the call.
func (s *JWTSigner) GetToken() (*JWT, error) {
	now := time.Now()
	// This is very heavy on read and atomics are said to be much faster
	// than RWMutex. Not that it is important in this case, though.
	res := s.currentToken.Load()
	if res != nil && res.(*JWT).ExpiresAt.After(now) {
		return res.(*JWT), nil
	}
	// We could safely forgo a mutex here and generate more than one
	// new token concurrently, let them all get used and then overwritten,
	// but lets do it cleanly and not annoy APN servers.
	s.mu.Lock()
	defer s.mu.Unlock()
	// Check again in case someone else got here first.
	res = s.currentToken.Load()
	if res != nil && res.(*JWT).ExpiresAt.After(now) {
		return res.(*JWT), nil
	}
	if s.signingMethod == nil {
		if s.SigningMethod == nil {
			s.signingMethod = DefaultJWTSigningMethod
		} else {
			s.signingMethod = s.SigningMethod
		}
	}
	if s.tokenLifeSpan == 0 {
		if s.TokenLifeSpan > 0 {
			s.tokenLifeSpan = s.TokenLifeSpan
		} else {
			s.tokenLifeSpan = DefaultTokenLifeSpan
		}
	}
	t := &jwt.Token{
		Header: map[string]interface{}{
			"alg": s.signingMethod.Name,
			"kid": s.KeyID,
		},
		Claims: jwt.MapClaims{
			"iss": s.TeamID,
			"iat": now.Unix(),
		},
		Method: s.signingMethod,
	}
	ss, err := t.SignedString(s.SigningKey)
	if err != nil {
		return nil, err
	}
	tkn := &JWT{
		IssuedAt:  now,
		ExpiresAt: now.Add(s.tokenLifeSpan),
		JwtToken:  t,
		AsHeader:  fmt.Sprintf("bearer %v", ss),
	}
	s.currentToken.Store(tkn)
	return tkn, nil
}

type noSigner struct{}

func (s noSigner) SignRequest(r *http.Request) error {
	return nil
}

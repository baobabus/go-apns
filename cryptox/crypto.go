// The MIT License (MIT)
//
// Copyright (c) 2016 Adam Jones
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//
// Modifications copyright 2017 Aleksey Blinov. All rights reserved.

package cryptox

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"

	"golang.org/x/crypto/pkcs12"
)

const (
	PEM_X509     string = "CERTIFICATE"
	PEM_RSA             = "RSA PRIVATE KEY"
	PEM_PKCS8           = "ENCRYPTED PRIVATE KEY"
	PEM_PKCS8INF        = "PRIVATE KEY"
)

var (
	ErrPKCS8NotPem           = errors.New("PKCS8PrivateKey: invalid .p8 PEM file")
	ErrPKCS8NotECDSA         = errors.New("PKCS8PrivateKey: key must be of type ecdsa.PrivateKey")
	ErrPEMMissingPrivateKey  = errors.New("PEM: private key not found")
	ErrPEMMissingCertificate = errors.New("PEM: certificate not found")
)

// PKCS8PrivateKeyFromFile loads a .p8 certificate from a local file and returns a
// *ecdsa.PrivateKey.
func PKCS8PrivateKeyFromFile(filename string) (*ecdsa.PrivateKey, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return PKCS8PrivateKeyFromBytes(bytes)
}

// PKCS8PrivateKeyFromBytes decodes a .p8 certificate from an in memory byte slice and
// returns an *ecdsa.PrivateKey.
func PKCS8PrivateKeyFromBytes(bytes []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(bytes)
	if block == nil {
		return nil, ErrPKCS8NotPem
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	switch pk := key.(type) {
	case *ecdsa.PrivateKey:
		return pk, nil
	default:
		return nil, ErrPKCS8NotECDSA
	}
}

// ClientCertFromP12File loads a PKCS#12 certificate from a local file and returns a
// tls.Certificate.
//
// Use "" as the password argument if the PKCS#12 certificate is not password
// protected.
func ClientCertFromP12File(filename string, password string) (tls.Certificate, error) {
	p12bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return tls.Certificate{}, err
	}
	return ClientCertFromP12Bytes(p12bytes, password)
}

// ClientCertFromP12Bytes loads a PKCS#12 certificate from an in memory byte array and
// returns a tls.Certificate.
//
// Use "" as the password argument if the PKCS#12 certificate is not password
// protected.
func ClientCertFromP12Bytes(bytes []byte, password string) (tls.Certificate, error) {
	key, cert, err := pkcs12.Decode(bytes, password)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  key,
		Leaf:        cert,
	}, nil
}

// ClientCertFromPemFile loads a PEM certificate from a local file and returns a
// tls.Certificate. This function is similar to the crypto/tls LoadX509KeyPair
// function, however it supports PEM files with the cert and key combined
// in the same file, as well as password protected key files which are both
// common with APNs certificates.
//
// Use "" as the password argument if the PEM certificate is not password
// protected.
func ClientCertFromPemFile(filename string, password string) (tls.Certificate, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return tls.Certificate{}, err
	}
	return ClientCertFromPemBytes(bytes, password)
}

// ClientCertFromPemBytes loads a PEM certificate from an in memory byte array and
// returns a tls.Certificate. This function is similar to the crypto/tls
// X509KeyPair function, however it supports PEM files with the cert and
// key combined, as well as password protected keys which are both common with
// APNs certificates.
//
// Use "" as the password argument if the PEM certificate is not password
// protected.
func ClientCertFromPemBytes(bytes []byte, password string) (tls.Certificate, error) {
	var cert tls.Certificate
	var block *pem.Block
	for {
		block, bytes = pem.Decode(bytes)
		if block == nil {
			break
		}
		var isRSA bool
		switch block.Type {
		case PEM_X509:
			cert.Certificate = append(cert.Certificate, block.Bytes)
		case PEM_RSA:
			isRSA = true
			fallthrough
		case PEM_PKCS8, PEM_PKCS8INF:
			key, err := decryptPrivateKey(block, password, isRSA)
			if err != nil {
				return tls.Certificate{}, err
			}
			cert.PrivateKey = key
		}
	}
	if len(cert.Certificate) == 0 {
		return tls.Certificate{}, ErrPEMMissingCertificate
	}
	if cert.PrivateKey == nil {
		return tls.Certificate{}, ErrPEMMissingPrivateKey
	}
	if c, err := x509.ParseCertificate(cert.Certificate[0]); err == nil {
		cert.Leaf = c
	}
	return cert, nil
}

// RootCAFromPemFile loads a PEM certificate from a local file and returns a
// tls.Certificate.
func RootCAFromPemFile(filename string) (tls.Certificate, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return tls.Certificate{}, err
	}
	return RootCAFromPemBytes(bytes)
}

// RootCAFromPemBytes loads a PEM certificate from an in memory byte array and
// returns a tls.Certificate.
func RootCAFromPemBytes(bytes []byte) (tls.Certificate, error) {
	var cert tls.Certificate
	var block *pem.Block
	for {
		block, bytes = pem.Decode(bytes)
		if block == nil {
			break
		}
		if block.Type == PEM_X509 {
			cert.Certificate = append(cert.Certificate, block.Bytes)
		}
	}
	if len(cert.Certificate) == 0 {
		return tls.Certificate{}, ErrPEMMissingCertificate
	}
	// This should not be needed:
	// if c, err := x509.ParseCertificate(cert.Certificate[0]); err == nil {
	// 	cert.Leaf = c
	// }
	return cert, nil
}

func decryptPrivateKey(block *pem.Block, password string, isRSA bool) (crypto.PrivateKey, error) {
	bytes := block.Bytes
	if x509.IsEncryptedPEMBlock(block) {
		var err error
		bytes, err = x509.DecryptPEMBlock(block, []byte(password))
		if err != nil {
			return nil, err
		}
	}
	return parsePrivateKey(bytes, isRSA)
}

func parsePrivateKey(bytes []byte, isRSA bool) (res crypto.PrivateKey, err error) {
	// Thanks to @jameshfisher for brining up PKCS8 case
	if isRSA {
		res, err = x509.ParsePKCS1PrivateKey(bytes)
	} else {
		res, err = x509.ParsePKCS8PrivateKey(bytes)
	}
	return res, err
}

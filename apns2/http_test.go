// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"testing"

	"time"

	"github.com/baobabus/go-apnsmock/apns2mock"
	"github.com/stretchr/testify/assert"
)

var (
	apnsMockComms_Typical = apns2mock.CommsCfg{
		MaxConcurrentStreams: 500,
		MaxConns:             1000,
		ConnectionDelay:      1*time.Second,
		ResponseTime:         20*time.Millisecond,
	}
	apnsMockComms_NoDelay = apns2mock.CommsCfg{
		MaxConcurrentStreams: 500,
		MaxConns:             1000,
		ConnectionDelay:      0,
		ResponseTime:         0,
	}
)

func mustNewMockServer(t *testing.T) *apns2mock.Server {
	res, err := apns2mock.NewServer(apnsMockComms_NoDelay, apns2mock.AllOkayHandler, apns2mock.AutoCert, apns2mock.AutoKey)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func mustNewHTTPClient(t *testing.T, s *apns2mock.Server) *HTTPClient {
	res, err := NewHTTPClient(s.URL, CommsFast, nil, s.RootCertificate)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestGetClientConnNoHTTP2Incursion(t *testing.T) {
	s := mustNewMockServer(t)
	defer s.Close()
	c := mustNewHTTPClient(t, s)
	cc, err := c.getClientConn(); 
	if err != nil {
		t.Fatal(err)
	}
	if cc != nil {
		t.Fatal("Should not have gotten a connection")
	}
}

func TestGetClientConn(t *testing.T) {
	s := mustNewMockServer(t)
	defer s.Close()
	c := mustNewHTTPClient(t, s)
	c.precise = true
	cc, err := c.getClientConn(); 
	if err != nil {
		t.Fatal(err)
	}
	if cc == nil {
		t.Fatal("Should have gotten a connection")
	}
}

func TestReservedStreamNoContention(t *testing.T) {
	s := mustNewMockServer(t)
	defer s.Close()
	c := mustNewHTTPClient(t, s)
	st, err := c.ReservedStream(nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, uint32(1), c.cnt)
	st.Close()
	assert.Equal(t, uint32(0), c.cnt)
}

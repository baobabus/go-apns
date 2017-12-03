// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"testing"

	"time"

	"github.com/baobabus/go-apnsmock/apns2mock"
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

func newMockServer(t *testing.T) (*apns2mock.Server, error) {
	return apns2mock.NewServer(apnsMockComms_NoDelay, apns2mock.AllOkayHandler, apns2mock.AutoCert, apns2mock.AutoKey)
}

func TestConnectNoHTTP2Incursion(t *testing.T) {
	s, err := newMockServer(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	c, err := NewHTTPClient(s.URL, CommsFast, nil, s.RootCertificate)
	if err != nil {
		t.Fatal(err)
	}
	cc, err := c.GetClientConn(); 
	if err != nil {
		t.Fatal(err)
	}
	if cc != nil {
		t.Fatal("Should not have gotten a connection")
	}
}

func TestConnect(t *testing.T) {
	s, err := newMockServer(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	c, err := NewHTTPClient(s.URL, CommsFast, nil, s.RootCertificate)
	if err != nil {
		t.Fatal(err)
	}
	c.precise = true
	cc, err := c.GetClientConn(); 
	if err != nil {
		t.Fatal(err)
	}
	if cc == nil {
		t.Fatal("Should have gotten a connection")
	}
}

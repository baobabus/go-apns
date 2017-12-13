// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"testing"

	"golang.org/x/net/http2"
)

func TestDialOk(t *testing.T) {
	s := mustNewMockServer(t)
	defer s.Close()
	d := makeDialer(commsTest_Fast)
	tc := s.Client().Transport.(*http2.Transport).TLSClientConfig
	c, err := d("tcp", s.URL[8:], tc)
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("Should have connected")
	}
}

func TestDialTimeout(t *testing.T) {
	s := mustNewMockServerWithCfg(t, apnsMockComms_30ms)
	defer s.Close()
	d := makeDialer(commsTest_Fast)
	tc := s.Client().Transport.(*http2.Transport).TLSClientConfig
	c, err := d("tcp", s.URL[8:], tc)
	if err == nil || err.Error() != "tls: DialWithDialer timed out" {
		t.Fatal("Should have gotten error tls: DialWithDialer timed out")
	}
	if c == nil {
		t.Fatal("Should not have connected")
	}
}

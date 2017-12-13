// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

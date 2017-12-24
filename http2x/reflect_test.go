// Copyright 2017 Aleksey Blinov. All rights reserved.

package http2x

import (
	"net/http"
	"testing"

	"golang.org/x/net/http2"
)

func TestGetClientConnPool(t *testing.T) {
	res, err := GetClientConnPool(nil)
	if res != nil || err == nil {
		t.Fatal("Should have failed to get connection")
	}
	if err != ErrUnsupportedTransport {
		t.Fatal("Wrong error: ", err)
	}
	res, err = GetClientConnPool(http.DefaultTransport)
	if res != nil || err == nil {
		t.Fatal("Should have failed to get connection")
	}
	if err != ErrUnsupportedTransport {
		t.Fatal("Wrong error: ", err)
	}
	tr := &http2.Transport{}
	res, err = GetClientConnPool(tr)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("Should have gotten connection pool")
	}
}

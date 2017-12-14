// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"testing"

	"github.com/baobabus/go-apns/cryptox"
	"github.com/baobabus/go-apnsmock/apns2mock"
	"github.com/stretchr/testify/assert"
)

func mustNewClient_Signer_Good(t tester, s *apns2mock.Server) *Client {
	//t.Helper()
	tsk, err := cryptox.PKCS8PrivateKeyFromBytes([]byte(testTokenKey_Good))
	if err != nil {
		t.Fatal(err)
	}
	res := &Client{
		Gateway: s.URL,
		RootCA:  s.RootCertificate,
		Signer: &JWTSigner{
			KeyID:      "ABC123DEFG",
			TeamID:     "DEF123GHIJ",
			SigningKey: tsk,
		},
		CommsCfg: commsTest_Fast,
		ProcCfg:  MinBlockingProcConfig,
		Callback: NoCallback,
	}
	return res
}

func TestClient_Signer_Good_1(t *testing.T) {
	s := mustNewMockServer(t)
	defer s.Close()
	c := mustNewClient_Signer_Good(t, s)
	err := c.Start(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Stop()
	tcs := []struct {
		ntf *Notification
		exp *Result
		cb  chan *Result
	}{
		{
			testNotif_Good,
			&Result{
				Response: &Response{
					StatusCode:      200,
					RejectionReason: "",
				},
				Err: nil,
			},
			make(chan *Result, 1),
		},
		{
			testNotif_BadDevice,
			&Result{
				Response: &Response{
					StatusCode:      400,
					RejectionReason: ReasonBadDeviceToken,
				},
				Err: nil,
			},
			make(chan *Result, 1),
		},
	}
	for _, tc := range tcs {
		err = c.Push(tc.ntf, DefaultSigner, NoContext, tc.cb)
		if err != nil {
			t.Fatal(err)
		}
		r := <-tc.cb
		if r == nil && tc.exp != nil {
			t.Fatal("Should have gotten a result")
		}
		if r.Response == nil && tc.exp.Response != nil {
			t.Fatal("Should have gotten a response")
		}
		assert.Equal(t, tc.exp.Response.StatusCode, r.Response.StatusCode)
		assert.Equal(t, tc.exp.Response.RejectionReason, r.Response.RejectionReason)
		if r.Err != nil && tc.exp.Err == nil {
			t.Fatal("Error in result:", r.Err)
		}
	}
}

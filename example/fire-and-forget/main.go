// Copyright 2017 Aleksey Blinov. All rights reserved.

package main

import (
	"log"

	"github.com/baobabus/go-apns/apns2"
	"github.com/baobabus/go-apns/cryptox"
)

func main() {

	// Load and parse our token signing key
	signingKey, err := cryptox.PKCS8PrivateKeyFromFile("token_signing_pk.p8")
	if err != nil {
		log.Fatal("Token signing key error: ", err)
	}

	// Set up our client
	client := &apns2.Client{
		Gateway:  apns2.Gateway.Production,
		Signer:   &apns2.JWTSigner{
			KeyID: "ABC123DEFG", // Your key ID
			TeamID: "DEF123GHIJ", // Your team ID
			SigningKey: signingKey,
		},
		CommsCfg: apns2.CommsFast,
		ProcCfg:  apns2.UnlimitedProcConfig,
	}

	// Start processing
	err = client.Start(nil)
	if err != nil {
		log.Fatal("Client start error: ", err)
	}

	// Mock motification and recipients
	header := &apns2.Header{ Topic: "com.example.Alert" }
	payload := &apns2.Payload{ APS: &apns2.APS{Alert: "Ping!"} }
	recipients := []string{
		"00fc13adff785122b4ad28809a3420982341241421348097878e577c991de8f0",
		"10fc13adff785122b4ad28809a3420982341241421348097878e577c991de8f0",
		"20fc13adff785122b4ad28809a3420982341241421348097878e577c991de8f0",
	}

	// Push to all recipients
	for _, rcpt := range recipients {
		notif := &apns2.Notification{
			Recipient: rcpt,
			Header:    header,
			Payload:   payload,
		}
		err := client.Push(notif, apns2.DefaultSigner, apns2.NoContext, apns2.DefaultCallback)
		if err != nil {
			log.Fatal("Push error: ", err)
		}
	}

	// Perform soft shutdown allowing the processing to complete.
	client.Stop()
}

// Copyright 2017 Aleksey Blinov. All rights reserved.

package apns2

import (
	"encoding/json"
	"sync/atomic"
)

// Payload is the container for the actual data to be delivered
// to the notification recipient.
// How a payload is utilized is not constrained, but the intent is
// to never modify it once created and assigned.
// The same payload can then be sent to any number of recipients.
type Payload struct {
	APS  *APS
	Raw  map[string]interface{}
	json atomic.Value
}

type APS struct {
	Alert            interface{}
	Badge            interface{}
	Category         string
	ContentAvailable bool
	MutableContent   bool
	Sound            string
	ThreadID         string
	URLArgs          []string
}

type Alert struct {
	Action       string   `json:"action,omitempty"`
	ActionLocKey string   `json:"action-loc-key,omitempty"`
	Body         string   `json:"body,omitempty"`
	LaunchImage  string   `json:"launch-image,omitempty"`
	LocArgs      []string `json:"loc-args,omitempty"`
	LocKey       string   `json:"loc-key,omitempty"`
	Title        string   `json:"title,omitempty"`
	Subtitle     string   `json:"subtitle,omitempty"`
	TitleLocArgs []string `json:"title-loc-args,omitempty"`
	TitleLocKey  string   `json:"title-loc-key,omitempty"`
}

func (p *Payload) MarshalJSON() ([]byte, error) {
	res := p.json.Load()
	if res != nil {
		return res.([]byte), nil
	}
	// We could protect this with a Mutex, but for improved throughput
	// it is probably better to avoid resource contention here and just
	// duplicate the work in case we have concurrent calls.
	m := p.mergedMap()
	j, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	p.json.Store(j)
	return j, nil
}

func (p *Payload) mergedMap() map[string]interface{} {
	if p.APS == nil {
		return p.Raw
	}
	res := make(map[string]interface{})
	// 1. Shallow copy the original raw map
	for k, v := range p.Raw {
		res[k] = v
	}
	// 2. Overwrite APS fields
	if aps, ok := res["aps"]; !ok {
		res["aps"] = make(map[string]interface{})
	} else if _, ok := aps.(map[string]interface{}); !ok {
		res["aps"] = make(map[string]interface{})
	}
	p.APS.addToMap(res["aps"].(map[string]interface{}))
	return res
}

func (a APS) addToMap(m map[string]interface{}) {
	if a.Alert != nil {
		m["alert"] = a.Alert
	}
	if a.Badge != nil {
		m["badge"] = a.Badge
	}
	if a.Category != "" {
		m["category"] = a.Category
	}
	if a.ContentAvailable {
		m["content-available"] = 1
	}
	if a.MutableContent {
		m["mutable-content"] = 1
	}
	if a.Sound != "" {
		m["sound"] = a.Sound
	}
	if a.ThreadID != "" {
		m["thread-id"] = a.ThreadID
	}
	if len(a.URLArgs) > 0 {
		m["url-args"] = a.URLArgs
	}
}

package audit

import (
	"testing"
	"time"
)

func TestValidateDetectsPayloadTampering(t *testing.T) {
	e1, err := NewEvent("e1", "c1", "甲", "created", 1, 1, time.Now(), map[string]any{"value": 1}, "")
	if err != nil {
		t.Fatal(err)
	}
	e2, err := NewEvent("e2", "c1", "乙", "updated", 2, 2, time.Now(), map[string]any{"value": 2}, e1.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate([]Event{e2, e1}); err != nil {
		t.Fatal(err)
	}
	e2.Payload = []byte(`{"value":3}`)
	if err := Validate([]Event{e1, e2}); err == nil {
		t.Fatal("篡改载荷后审计链应失效")
	}
}

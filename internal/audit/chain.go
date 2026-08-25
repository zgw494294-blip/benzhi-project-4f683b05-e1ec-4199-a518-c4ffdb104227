package audit

import (
	"encoding/json"
	"fmt"
	"sort"

	"seed-vault-release/internal/domain"
)

func Stable(events []Event) []Event {
	out := append([]Event(nil), events...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Sequence == out[j].Sequence {
			return out[i].ID < out[j].ID
		}
		return out[i].Sequence < out[j].Sequence
	})
	return out
}

func Validate(events []Event) error {
	ordered := Stable(events)
	previous := ""
	for i, event := range ordered {
		wantSequence := uint64(i + 1)
		if event.Sequence != wantSequence {
			return fmt.Errorf("审计序号不连续：期望%d，实际%d", wantSequence, event.Sequence)
		}
		if event.PreviousDigest != previous {
			return fmt.Errorf("审计事件%d的前序摘要不一致", event.Sequence)
		}
		if domain.DigestText(string(event.Payload)) != event.PayloadDigest {
			return fmt.Errorf("审计事件%d的载荷摘要校验失败", event.Sequence)
		}
		if event.ComputeDigest() != event.Digest {
			return fmt.Errorf("审计事件%d摘要校验失败", event.Sequence)
		}
		previous = event.Digest
	}
	return nil
}

func LastDigest(events []Event) string {
	ordered := Stable(events)
	if len(ordered) == 0 {
		return ""
	}
	return ordered[len(ordered)-1].Digest
}

func ValidateReleaseEvents(events []Event, cert *domain.ReleaseCertificate) (bool, bool) {
	if cert == nil {
		return false, false
	}
	freezeSequence := uint64(0)
	issue := false
	for _, event := range Stable(events) {
		if event.CaseRevision != cert.CaseRevision {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if event.Action == "case.frozen" && payload["manifestDigest"] == cert.ManifestDigest {
			freezeSequence = event.Sequence
		}
		if event.Action == "certificate.issued" && freezeSequence > 0 && event.Sequence > freezeSequence {
			serial, _ := payload["serialNumber"].(float64)
			issue = uint64(serial) == cert.SerialNumber && payload["manifestDigest"] == cert.ManifestDigest
		}
	}
	return freezeSequence > 0, issue
}

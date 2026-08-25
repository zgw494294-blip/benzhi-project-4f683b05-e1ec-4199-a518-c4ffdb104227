package audit

import (
	"encoding/json"
	"time"

	"seed-vault-release/internal/domain"
)

type Event struct {
	ID             string          `json:"id"`
	CaseID         string          `json:"caseId"`
	Sequence       uint64          `json:"sequence"`
	Actor          string          `json:"actor"`
	Action         string          `json:"action"`
	CaseRevision   uint64          `json:"caseRevision"`
	At             time.Time       `json:"at"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	PayloadDigest  string          `json:"payloadDigest"`
	PreviousDigest string          `json:"previousDigest"`
	Digest         string          `json:"digest"`
}

func NewEvent(id, caseID, actor, action string, revision, sequence uint64, at time.Time, payload any, previous string) (Event, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	e := Event{ID: id, CaseID: caseID, Sequence: sequence, Actor: actor, Action: action, CaseRevision: revision, At: at.UTC(), Payload: b, PayloadDigest: domain.DigestText(string(b)), PreviousDigest: previous}
	e.Digest = e.ComputeDigest()
	return e, nil
}

func (e Event) ComputeDigest() string {
	type digestFields struct {
		ID             string `json:"id"`
		CaseID         string `json:"caseId"`
		Sequence       uint64 `json:"sequence"`
		Actor          string `json:"actor"`
		Action         string `json:"action"`
		CaseRevision   uint64 `json:"caseRevision"`
		At             string `json:"at"`
		PayloadDigest  string `json:"payloadDigest"`
		PreviousDigest string `json:"previousDigest"`
	}
	b, _ := json.Marshal(digestFields{e.ID, e.CaseID, e.Sequence, e.Actor, e.Action, e.CaseRevision, e.At.UTC().Format(time.RFC3339Nano), e.PayloadDigest, e.PreviousDigest})
	return domain.DigestText(string(b))
}

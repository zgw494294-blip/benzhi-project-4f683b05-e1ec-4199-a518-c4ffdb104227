package canceled_add_lot_commit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"seed-vault-release/internal/application"
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/httpui"
	"seed-vault-release/internal/repository"
)

type cancelAfterCheckContext struct {
	mu       sync.Mutex
	checked  bool
	canceled chan struct{}
}

func newCancelAfterCheckContext() *cancelAfterCheckContext {
	return &cancelAfterCheckContext{canceled: make(chan struct{})}
}

func (*cancelAfterCheckContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *cancelAfterCheckContext) Done() <-chan struct{}     { return c.canceled }
func (*cancelAfterCheckContext) Value(any) any               { return nil }

func (c *cancelAfterCheckContext) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.checked {
		c.checked = true
		close(c.canceled)
		return nil
	}
	return context.Canceled
}

func TestCanceledAddLotDoesNotCommit(t *testing.T) {
	store, err := repository.Open(filepath.Join(t.TempDir(), "canceled-add-lot.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	service := application.New(store)
	now := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	created, err := service.CreateCase(application.CreateCaseCommand{
		Context:        application.Context{Actor: "接收员甲", Role: domain.RoleReceiver, IdempotencyKey: "cancel-create"},
		AccessionCode:  "CANCEL-LOT-001",
		SpeciesName:    "银杉",
		CollectionSite: "广西花坪保护区",
		CollectedAt:    now.Add(-48 * time.Hour),
		ReceivedAt:     now.Add(-24 * time.Hour),
		Owner:          "接收员甲",
	})
	if err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(application.AddLotCommand{
		Context:          application.Context{Actor: "接收员甲", Role: domain.RoleReceiver, ExpectedRevision: created.Revision, IdempotencyKey: "cancel-add-lot"},
		LotCode:          "LOT-CANCELED",
		ContainerCode:    "BOX-CANCELED",
		SeedCount:        300,
		MoisturePercent:  7.8,
		ArrivalCondition: "完整、干燥",
		ReservedCount:    80,
	})
	if err != nil {
		t.Fatal(err)
	}

	requestContext := newCancelAfterCheckContext()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+created.ID+"/lots", bytes.NewReader(body)).WithContext(requestContext)
	res := httptest.NewRecorder()
	httpui.New(service).Handler().ServeHTTP(res, req)

	stored, err := store.GetCase(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	events, err := store.Timeline(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Code == http.StatusOK || stored.Revision != created.Revision || len(stored.Lots) != 0 || len(events) != 1 {
		t.Fatalf("已取消的登记仍被提交: status=%d revision=%d lots=%d events=%d", res.Code, stored.Revision, len(stored.Lots), len(events))
	}
}

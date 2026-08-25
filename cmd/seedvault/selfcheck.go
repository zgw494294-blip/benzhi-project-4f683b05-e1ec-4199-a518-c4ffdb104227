package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"seed-vault-release/internal/application"
	"seed-vault-release/internal/domain"
)

type selfClient struct {
	base     string
	client   *http.Client
	sequence int
}

func (c *selfClient) post(ctx context.Context, path string, body any, out any) error {
	return c.write(ctx, http.MethodPost, path, body, out)
}

func (c *selfClient) patch(ctx context.Context, path string, body any, out any) error {
	return c.write(ctx, http.MethodPatch, path, body, out)
}

func (c *selfClient) write(ctx context.Context, method, path string, body any, out any) error {
	c.sequence++
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("%s %s返回%d: %s", method, path, res.StatusCode, string(data))
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

func (c *selfClient) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(res.Body)
		return fmt.Errorf("GET %s返回%d: %s", path, res.StatusCode, string(data))
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func scContext(actor, role string, revision uint64, key string) map[string]any {
	return map[string]any{"actor": actor, "role": role, "expectedRevision": revision, "idempotencyKey": key}
}

func runSelfcheck(ctx context.Context, base string) error {
	c := &selfClient{base: base, client: &http.Client{Timeout: 4 * time.Second}}
	var health map[string]any
	if err := c.get(ctx, "/healthz", &health); err != nil {
		return fmt.Errorf("健康检查失败: %w", err)
	}
	now := time.Now().UTC()
	create := map[string]any{"actor": "接收员甲", "role": "receiver", "idempotencyKey": "sc-create", "accessionCode": "SC-2026-001", "speciesName": "银杉", "collectionSite": "广西花坪保护区", "collectedAt": now.Add(-48 * time.Hour), "receivedAt": now.Add(-24 * time.Hour), "owner": "接收员甲"}
	var aggregate domain.AccessionCase
	if err := c.post(ctx, "/api/v1/cases", create, &aggregate); err != nil {
		return err
	}
	caseID := aggregate.ID
	revise := scContext("接收员甲", "receiver", aggregate.Revision, "sc-revise")
	revise["collectionSite"] = "广西花坪保护区核心区"
	if err := c.patch(ctx, "/api/v1/cases/"+caseID+"/base-data", revise, &aggregate); err != nil {
		return err
	}
	lot := scContext("接收员甲", "receiver", aggregate.Revision, "sc-lot")
	lot["items"] = []map[string]any{{"lotCode": "LOT-A", "containerCode": "C-001", "seedCount": 300, "moisturePercent": 7.8, "arrivalCondition": "完整、干燥、无虫蛀", "reservedCount": 80}}
	if err := c.post(ctx, "/api/v1/cases/"+caseID+"/lots/batch", lot, &aggregate); err != nil {
		return err
	}
	if err := c.post(ctx, "/api/v1/cases/"+caseID+"/sampling/generate", scContext("接收员甲", "receiver", aggregate.Revision, "sc-sample-gen"), &aggregate); err != nil {
		return err
	}
	actual := map[string]int{aggregate.Lots[0].ID: 55}
	confirm := scContext("接收员甲", "receiver", aggregate.Revision, "sc-sample-confirm")
	confirm["actualCounts"] = actual
	confirm["deviationReasons"] = map[string]string{aggregate.Lots[0].ID: "器皿损耗补量"}
	if err := c.post(ctx, "/api/v1/cases/"+caseID+"/sampling/confirm", confirm, &aggregate); err != nil {
		return err
	}
	initialTests := scContext("检验员乙", "tester", aggregate.Revision, "sc-tests-batch")
	initialTests["items"] = []map[string]any{
		{"lotId": aggregate.Lots[0].ID, "testType": "germination", "replicateCounts": []int{25, 25}, "testedCount": 50, "passedCount": 35, "contaminatedCount": 0, "threshold": 80},
		{"lotId": aggregate.Lots[0].ID, "testType": "contamination", "replicateCounts": []int{25, 25}, "testedCount": 50, "passedCount": 49, "contaminatedCount": 1, "threshold": 5},
	}
	if err := c.post(ctx, "/api/v1/cases/"+caseID+"/tests/batch", initialTests, &aggregate); err != nil {
		return err
	}
	if len(aggregate.Findings) != 1 {
		return fmt.Errorf("未为异常发芽检验生成发现项")
	}
	findingID := aggregate.Findings[0].ID
	assign := scContext("检验员乙", "tester", aggregate.Revision, "sc-assign")
	assign["assignee"] = "检验员乙"
	assign["dueAt"] = now.Add(7 * 24 * time.Hour)
	if err := c.post(ctx, "/api/v1/cases/"+caseID+"/findings/"+findingID+"/assign", assign, &aggregate); err != nil {
		return err
	}
	evidence := scContext("检验员乙", "tester", aggregate.Revision, "sc-evidence")
	evidence["description"] = "温湿度复核与培养基更换记录"
	evidence["content"] = "影像记录EV-001及操作台账摘要"
	if err := c.post(ctx, "/api/v1/cases/"+caseID+"/findings/"+findingID+"/evidence", evidence, &aggregate); err != nil {
		return err
	}
	retest := scContext("检验员乙", "tester", aggregate.Revision, "sc-retest")
	retest["lotId"] = aggregate.Lots[0].ID
	retest["testType"] = "germination"
	retest["replicateCounts"] = []int{25, 25}
	retest["testedCount"] = 50
	retest["passedCount"] = 46
	retest["contaminatedCount"] = 0
	retest["threshold"] = 80
	retest["isRetest"] = true
	if err := c.post(ctx, "/api/v1/cases/"+caseID+"/tests", retest, &aggregate); err != nil {
		return err
	}
	retestID := aggregate.Tests[len(aggregate.Tests)-1].ID
	remediate := scContext("检验员乙", "tester", aggregate.Revision, "sc-remediate")
	remediate["retestId"] = retestID
	if err := c.post(ctx, "/api/v1/cases/"+caseID+"/findings/"+findingID+"/remediate", remediate, &aggregate); err != nil {
		return err
	}
	if err := c.post(ctx, "/api/v1/cases/"+caseID+"/findings/"+findingID+"/close", scContext("检验员丙", "tester", aggregate.Revision, "sc-close"), &aggregate); err != nil {
		return err
	}
	if aggregate.Status != domain.StatusReview {
		return fmt.Errorf("关闭发现项后状态为%s，期望待审核", aggregate.Status)
	}
	if err := c.post(ctx, "/api/v1/cases/"+caseID+"/approve", scContext("审核员丁", "reviewer", aggregate.Revision, "sc-approve"), &aggregate); err != nil {
		return err
	}
	if aggregate.Status != domain.StatusReleased || aggregate.Certificate == nil {
		return fmt.Errorf("案卷未成功放行")
	}
	queries := []string{
		fmt.Sprintf("serialNumber=%d", aggregate.Certificate.SerialNumber),
		"accessionCode=" + url.QueryEscape(aggregate.AccessionCode),
		"manifestDigest=" + url.QueryEscape(aggregate.Certificate.ManifestDigest),
	}
	for _, query := range queries {
		var verification application.CertificateVerification
		if err := c.get(ctx, "/api/v1/certificates/verify?"+query, &verification); err != nil {
			return err
		}
		if !verification.Valid {
			return fmt.Errorf("凭据校验失败: %s", verification.Message)
		}
	}
	var detail application.CaseDetail
	if err := c.get(ctx, "/api/v1/cases/"+caseID, &detail); err != nil {
		return err
	}
	if !detail.AuditValid || len(detail.Timeline) < 9 {
		return fmt.Errorf("审计时间线不完整或摘要链无效")
	}
	return nil
}

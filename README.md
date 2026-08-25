# seed-vault-release

`seed-vault-release` 是面向野生植物种子库的入藏检验工作台。它只处理一条受控业务链路：到库建档、种子批次登记、抽样确认、发芽与污染检验、异常整改复验、审核冻结，以及签发不可变的保藏放行凭据。

系统由 Go 服务直接提供响应式 HTML、CSS、JavaScript 页面和同源 JSON HTTP 接口，不需要 Node 构建链。业务数据、状态索引、幂等结果、冻结清单、凭据及审计摘要链保存于本地 bbolt 数据库。所有聚合变更使用 `expectedRevision` 防止并发覆盖，并在单个数据库写事务内提交。

## 构建

```text
go build ./cmd/seedvault
```

## 运行

```text
go run ./cmd/seedvault -addr=127.0.0.1:19081 -data=./data/seedvault
```

然后访问 `http://127.0.0.1:19081/`。默认地址也是 `127.0.0.1:19081`。可以通过 `-addr=127.0.0.1:<port>` 更换回环端口；如果没有显式传入 `-addr`，也可以设置 `PORT`，服务会绑定到 `127.0.0.1:<PORT>`。程序拒绝非回环地址。

页面顶部可切换种质接收员、发芽检验员和保藏审核员身份。接收员可在抽样确认前受控修订基础资料、原子批量登记种子批次，并在抽样偏离计划时登记原因和数量平衡。检验员通过批次 × 检验类型矩阵批量补齐初检；异常发现项按照负责人、期限、只追加证据、同类型通过复验和独立复核的顺序关闭。资料完整后，审核员可冻结案卷并签发凭据。

工作台支持按发现项的未到期、临期、逾期状态筛选案卷。放行凭据可使用凭据序号、案卷编号或完整清单摘要中的任一种精确条件查询；核验结果会逐项对照只追加凭据、案卷版本、冻结清单、规范摘要以及冻结和签发审计事件，不执行写入。

## 测试与自检

运行全部自动化测试：

```text
go test ./...
```

运行有界 HTTP 冒烟自检：

```text
go run ./cmd/seedvault -selfcheck -addr=127.0.0.1:19081
```

自检会使用临时数据目录，启动真实回环 HTTP 监听，通过公开接口走完包含异常、整改、复验、独立关闭、审核、凭据校验和审计链校验的完整流程，随后主动关闭服务并清理临时数据。

## 数据与接口约定

- 正常运行的数据文件默认为 `data/seedvault/seedvault.db`，可用 `-data` 修改目录。
- 写请求需提供 `actor`、`role`、`expectedRevision` 和唯一的 `idempotencyKey`；创建案卷无需已有 revision。
- `GET /healthz` 用于健康检查；`GET /api/v1/cases` 可使用 `status` 和 `timeliness` 筛选，`GET /api/v1/cases/{caseID}` 返回最新 revision、抽样平衡、检验矩阵、发现项台账和完整审计时间线。
- `PATCH /api/v1/cases/{caseID}/base-data` 受控修订基础资料；`POST /api/v1/cases/{caseID}/lots/batch` 和 `POST /api/v1/cases/{caseID}/tests/batch` 分别执行全有或全无的批次登记与初检录入。
- `POST /api/v1/cases/{caseID}/findings/{findingID}/assign`、`.../evidence`、`.../remediate` 和 `.../close` 依次处理整改责任、只追加证据、复验提交和独立关闭。
- `GET /api/v1/certificates/verify` 接受 `serialNumber`、`accessionCode` 或 `manifestDigest` 中恰好一个查询参数；原有 `GET /api/v1/certificates/{serial}/verify` 仍可使用。

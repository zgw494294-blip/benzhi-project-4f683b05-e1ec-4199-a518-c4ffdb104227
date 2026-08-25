# BENZHI_README

基于 Go 实现的seed-vault-release Web 项目，一款后端服务，已完整实现野生植物种质材料从到库建档、批次登记、抽样确认、发芽与污染检验、异常整改复验、独立关闭、审核冻结到签发并校验不可变保藏放行凭据的浏览器工作台。

## 项目说明
- 项目：benzhi-project-4f683b05-e1ec-4199-a518-c4ffdb104227
- 项目用途：已完整实现野生植物种质材料从到库建档、批次登记、抽样确认、发芽与污染检验、异常整改复验、独立关闭、审核冻结到签发并校验不可变保藏放行凭据的浏览器工作台。
- Go 工具链：`golang:1.22`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run ./cmd/seedvault -selfcheck -addr=127.0.0.1:19081
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-4f683b05-e1ec-4199-a518-c4ffdb104227-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-4f683b05-e1ec-4199-a518-c4ffdb104227-arm64 linux/arm64
docker run -it benzhi-project-4f683b05-e1ec-4199-a518-c4ffdb104227-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/seedvault -selfcheck -addr=127.0.0.1:19081`

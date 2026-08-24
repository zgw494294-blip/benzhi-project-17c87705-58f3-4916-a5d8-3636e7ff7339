# BENZHI_README

基于 Go 实现的口述史资料受控发布工作台 Web 项目，一款后端服务，已完整实现口述史资料受控发布工作台，包含 49 个生产 Go 文件、SQLite WAL 持久化、同源 JSON API、原生浏览器工作台、中文 README、领域及端到端测试。固定回归命令均已实际运行通过；自检启动真实回环 HTTP 服务，验证页面交付和从建案到幂等签发发布包的完整流程后自行退出。

## 项目说明
- 项目：benzhi-project-17c87705-58f3-4916-a5d8-3636e7ff7339
- 项目用途：已完整实现口述史资料受控发布工作台，包含 49 个生产 Go 文件、SQLite WAL 持久化、同源 JSON API、原生浏览器工作台、中文 README、领域及端到端测试。固定回归命令均已实际运行通过；自检启动真实回环 HTTP 服务，验证页面交付和从建案到幂等签发发布包的完整流程后自行退出。
- Go 工具链：`golang:1.24`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run ./cmd/oralarchive -selfcheck -addr=127.0.0.1:19081
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-17c87705-58f3-4916-a5d8-3636e7ff7339-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-17c87705-58f3-4916-a5d8-3636e7ff7339-arm64 linux/arm64
docker run -it benzhi-project-17c87705-58f3-4916-a5d8-3636e7ff7339-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/oralarchive -selfcheck -addr=127.0.0.1:19081`

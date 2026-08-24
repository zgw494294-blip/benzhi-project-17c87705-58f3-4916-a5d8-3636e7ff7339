# 口述史资料受控发布工作台

本项目是一套由 Go 提供的本地浏览器工作台，用于把口述史采集案卷、知情同意范围、内容寻址音频、逐段转录与脱敏、受访者确认和最终发布串成可审计流程。只有同意范围有效、敏感内容均已裁定且受访者确认后的资料，才能签发不可变发布包。

系统数据默认保存在 `.oralarchive-data/`。SQLite 数据库使用 WAL 模式，音频对象和发布清单按 SHA-256 摘要保存。浏览器资源由 Go 内嵌提供，不需要 Node 构建链。

## 构建

```bash
go build ./...
```

## 运行

```bash
go run ./cmd/oralarchive -addr=127.0.0.1:19081
```

打开 `http://127.0.0.1:19081` 使用工作台。也可以设置 `PORT`，服务会绑定 `127.0.0.1:<PORT>`。`-data` 可指定本地数据目录；监听地址必须是回环地址。

## 测试

```bash
go test ./...
```

完整 HTTP 冒烟流程会启动真实回环服务，依次完成建案、同意冻结、音频上传、转录、脱敏裁定、确认和幂等发布，然后自行退出：

```bash
go run ./cmd/oralarchive -selfcheck -addr=127.0.0.1:19081
```

JSON API 使用 `X-Actor` 和 `X-Role` 标识操作人及角色。变更命令携带 `expectedVersion` 与 `idempotencyKey`，并发版本冲突返回 `409` 和统一的 `error` 对象。

新增流程入口：`POST /api/cases/{caseID}/segments/batch` 原子批量导入转录，`PATCH /api/cases/{caseID}/segments/{segmentID}/timecode` 定向修订时间码，`POST /api/cases/{caseID}/confirmation` 的退回分支会重置指定片段并保留确认历史，发布完成后可由发布员访问 `GET /api/cases/{caseID}/release/verify` 核验清单、片段摘要和音频对象。

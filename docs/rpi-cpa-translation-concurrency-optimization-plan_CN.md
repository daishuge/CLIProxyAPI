# RPi CPA 翻译插件并发错误优化计划

日期：2026-05-03

## 背景

Immersive Translate 通过 OpenAI-compatible 配置连接 RPi 上的 CPA：

- 插件：Edge 扩展 `amkbmndfnliijdhojkpoglbnaaahippg`，Immersive Translate `1.28.5`。
- 入口：`http://m.daishuge.win:8317/v1/chat/completions`。
- 模型：`gpt-5.3-codex-spark`。
- RPi 服务：`cliproxyapi.service`，实际运行 `CLIProxyAPI 6.10.0-ppap.8`。
- 上游：CPA 的 `codex` executor 通过 `socks5://127.0.0.1:7890` 访问 `https://chatgpt.com/backend-api/codex/responses`。

现象是翻译插件高并发时经常返回 `500`，错误为：

```text
Post "https://chatgpt.com/backend-api/codex/responses": EOF
```

这不是单纯的 RPi 算力问题。RPi CPU、内存、磁盘和本地端口状态都正常；`/healthz` 正常；代理链路也能通过 `socks5://127.0.0.1:7890` 访问 `chatgpt.com`。

## 关键证据

日志路径：

- 主日志：`/home/daishuge/project/cliproxyapi/logs/main.log`
- 错误日志：`/home/daishuge/project/cliproxyapi/logs/error-v1-chat-completions-*.log`

2026-05-03 的插件侧失败集中在 `gpt-5.3-codex-spark`：

- 插件来源 IP：`192.168.50.1`
- 插件 Origin：`chrome-extension://amkbmndfnliijdhojkpoglbnaaahippg`
- 失败状态：`500`
- 失败耗时：p50 约 `5.003s`，p90 约 `5.007s`
- 失败信息：`Post "https://chatgpt.com/backend-api/codex/responses": EOF`

受控并发复现结果：

| 并发 | 结果 |
| --- | --- |
| 1 | 1/1 成功 |
| 2 | 2/2 成功 |
| 4 | 4/4 成功 |
| 6 | 6/6 成功 |
| 8 | 8/8 成功 |
| 10 | 10/10 成功，单个请求曾拉到约 6.3 秒 |
| 12 | 12/12 成功 |
| 13 | 13/13 成功 |
| 14 | 10/14 成功，4 个同款 EOF |
| 16 | 1/16 成功，15 个同款 EOF |

结论：当前单 Codex OAuth 账号 + 单代理链路的稳定并发窗口约在 `12-13`，`14+` 开始明显不稳定，`16` 基本雪崩。

## 根因判断

根因不是一个点，而是三层叠加：

1. 上游 Codex web backend 或代理链路对单账号突发并发存在窗口限制，超过约 `13` 后会出现 EOF。
2. CPA 当前没有对 `codex` provider / 单 auth / 单模型做服务端并发闸门，插件突发请求会原样冲到上游。
3. CPA 的 HTTP client/transport 构造方式会放大问题：`NewProxyAwareHTTPClient` 每次请求都会创建新的 `http.Client` 和 proxy transport，SOCKS/TLS 连接复用能力弱，高并发时更容易变成一批新连接同时打上游。

代码责任边界：

- 上游 EOF 本身不是 CPA 能完全控制的。
- CPA 可以优化“不要把突发并发直接放大到上游”，也可以优化连接复用和排队策略。
- 插件侧 `limit` 只能缓解，不能从根上保护其他客户端或未来更大的突发。

## 优化目标

1. 同一 Codex OAuth auth 的上游并发可控，超过阈值时排队，而不是直接打爆上游。
2. 对代理连接进行复用，减少高并发时的 SOCKS/TLS 建连压力。
3. 不用盲目增加 `request-retry` 掩盖问题，避免失败时继续放大请求风暴。
4. 日志能保留足够证据，便于下次定位真实错误比例、模型、客户端和 request id。
5. 默认行为保持兼容，只有配置开启时才限制并发。

## 开发计划

### P0：保留复现用例

目的：后续每个优化都必须用同一套压力用例验收，避免只凭感觉改。

建议新增一个手动压测脚本或文档化现有探针：

- 输入：OpenAI-compatible `/v1/chat/completions`
- 模型：`gpt-5.3-codex-spark`
- payload：4 段字幕翻译，模拟 Immersive Translate 的 `%%` 分隔请求
- 并发档位：`10 / 12 / 13 / 14 / 16 / 24`
- 记录：status、elapsed_ms、错误 body head

验收口径：

- 优化前：`16` 并发能稳定复现 EOF。
- 优化后：`16` 和 `24` 并发不能出现 EOF 雪崩；允许排队导致总耗时增加。

### P1：复用 proxy-aware HTTP transport

涉及文件：

- `internal/runtime/executor/helps/proxy_helpers.go`
- `sdk/proxyutil/proxy.go`
- `internal/runtime/executor/helps/proxy_helpers_test.go`
- `sdk/proxyutil/proxy_test.go`

当前问题：

- `CodexExecutor.Execute` 每次请求调用 `helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 0)`。
- `NewProxyAwareHTTPClient` 每次都 `buildProxyTransport(proxyURL)`。
- SOCKS5 场景下每个 transport 自己维护连接池，因此请求之间难以复用 idle connections。

建议实现：

- 在 `helps` 层增加 process-wide transport cache，key 为 proxy mode + proxy URL。
- `NewProxyAwareHTTPClient` 可以继续每次返回新的 `http.Client`，但复用同一个 `Transport`。
- `direct` 和具体 proxy URL 都可以缓存。
- 只有 context 里传入的 custom `RoundTripper` 不缓存，因为它通常是调用方注入的 runtime 对象。
- 不在日志里打印完整 proxy URL，避免 proxy URL 含凭证。

测试：

- 同一个 proxy URL 两次创建 client，`client.Transport` 指针相同。
- 不同 proxy URL 的 transport 不相同。
- `direct` transport 可以复用，并且 `Proxy == nil`。
- 无 proxy 且 context 有 `cliproxy.roundtripper` 时仍优先使用 context transport。

收益：

- 降低突发翻译请求时的建连压力。
- 对所有 executor 都有收益，不只 Codex。
- 行为兼容，风险低。

限制：

- 这只能降低 EOF 概率，不能保证 `16+` 并发稳定，因为上游单账号窗口仍存在。

### P2：增加 provider/auth/model 级并发闸门

涉及文件：

- `internal/config/config.go`
- `config.example.yaml`
- `sdk/cliproxy/auth/conductor.go`
- `sdk/cliproxy/auth/conductor_overrides_test.go`
- 管理面板配置页可后置支持

建议配置形态：

```yaml
upstream-concurrency:
  default: 0
  providers:
    codex: 12
  queue-timeout-seconds: 30
```

说明：

- `default: 0` 表示默认不限制，保持兼容。
- `providers.codex: 12` 表示所有 Codex OAuth/API-key 请求进入同一个 provider 级闸门。
- 后续如果需要更细，可以扩展到 auth/model：

```yaml
upstream-concurrency:
  providers:
    codex: 12
  auths:
    codex-admin@camerondai.com-pro: 12
  models:
    codex/gpt-5.3-codex-spark: 12
```

第一版建议只做 provider 级，复杂度最低，能覆盖当前问题。

实现方式：

- 在 `auth.Manager` 中维护 limiter map。
- 在 `executeMixedOnce` / `executeStreamMixedOnce` / `executeCountMixedOnce` 选中 auth 后、调用 executor 前 acquire。
- acquire 使用请求 `context.Context`，客户端断开或超时后自动释放等待。
- executor 返回后 defer release。
- 排队超时返回明确错误，例如 `503 upstream concurrency queue timeout`，不要伪装成上游 `500`。
- 日志记录 provider、auth id、model、wait_ms、limit，但不要记录 secret。

测试：

- fake executor 统计同时运行数，配置 `codex: 2`，并发 8 时最大运行数不超过 2。
- context cancellation 时等待中的请求能退出，不泄漏 permit。
- 未配置时不改变现有并发行为。
- stream 和 non-stream 都释放 permit。

收益：

- 这是最直接的稳定性修复。
- 插件可以继续发较高并发，CPA 会把突发削成上游能承受的波形。
- `16` 并发会变成排队完成，而不是 15 个 EOF。

风险：

- 长 streaming 请求也会占用 permit，可能影响其他 Codex 请求。
- 如果只配 provider 级，`gpt-5.5` 和 `gpt-5.3-codex-spark` 会共享同一窗口。
- 后续可扩展 model/auth 级配置来细分。

### P3：调整瞬时 5xx 冷却与 retry 策略

涉及文件：

- `sdk/cliproxy/auth/conductor.go`
- `sdk/cliproxy/auth/conductor_overrides_test.go`
- `internal/runtime/executor/codex_executor.go`

当前逻辑：

- 500/502/503/504 会把模型标为 transient error，并设置约 1 分钟冷却。
- 现在配置 `max-retry-interval: 30`，所以这类 1 分钟冷却不会触发等待后 retry。
- 单纯提高 `request-retry` 没意义；把 `max-retry-interval` 拉到 60 秒可能让插件请求卡很久，也可能放大请求堆积。

建议：

- P2 完成前，不建议改 retry。
- P2 完成后，如果仍出现少量 EOF，可以只对“请求未向下游输出任何字节前的 transport EOF”做一次短抖动重试，例如 `100-500ms` jitter。
- 不要对所有 500 统一即时重试，避免真实上游故障时形成二次风暴。
- 可以把 `Post ... EOF` 归类为 transient transport error，日志单独打 `upstream_transport_eof`，便于统计。

2026-05-07 实施状态：

- 已在 Codex HTTP executor 层只针对连接级 transient transport error 增加一次短抖动重试，覆盖 `EOF`、`unexpected EOF`、`connection reset by peer`、idle connection closed、HTTP/2 GOAWAY 等尚未向下游输出内容的失败。
- 重试仍在已有 upstream concurrency permit 内执行，避免绕过 P2 并发闸门。
- 最终失败统一归类为 `502` / `upstream_transport_eof`，便于从错误日志和客户端响应中区分上游传输断流与普通应用错误。
- 已补测试覆盖 EOF 后第二次请求成功、错误归类，以及请求日志中的 `Cookie` / `Set-Cookie` 脱敏。

收益：

- 对偶发 EOF 有帮助。

风险：

- retry 是吞吐放大器。没有 P2 的并发闸门时，retry 可能让雪崩更严重。

### P4：扩容账号，而不是只调高并发

当前只有 1 个 Codex OAuth 账号参与该模型请求。要真正提高吞吐，需要增加可用上游身份。

建议：

- 增加第二个/第三个 Codex OAuth auth。
- 保持 `routing.strategy: round-robin`。
- 每个 auth 仍保留单账号并发上限，例如 12。

预期：

- 1 个账号：稳定并发约 12。
- 2 个账号：理论上可到约 24，但仍要实测。
- 3 个账号：理论上可到约 36，但仍受代理、上游风控和账号配额影响。

风险：

- 多账号涉及登录、凭证和配额管理。
- 需要确保不同账号的使用符合服务条款和账号策略。

### P5：改进日志和版本可观测性

涉及文件：

- `sdk/logging/request_logger.go`
- `internal/api/handlers/management/logs.go`
- `internal/buildinfo/*`
- 部署脚本或 release 流程

当前问题：

- `error-logs-max-files` 太小，只保留最近少量错误，旧错误文件被清理后主日志会出现 `failed to read error log info ... no such file`。
- `version.txt` 显示 `6.10.0-ppap.6`，实际运行二进制是 `6.10.0-ppap.8`，状态工具会被误导。

建议：

- RPi 配置把 error log 保留量调到至少 `100`。
- 错误日志索引里保留 request id、status、duration、model、origin、ua、error message 摘要。
- 管理面板增加按模型、status、client origin、错误类型聚合。
- 部署时自动更新 `version.txt`，或状态工具优先读取运行中二进制 `--version` 输出。

收益：

- 下次可以快速区分“插件突发”“上游 EOF”“配额/认证”“模型不存在”。
- 避免版本漂移造成误判。

## 推荐落地顺序

1. P0：固化复现脚本和验收标准。
2. P1：先做 HTTP transport 复用，低风险，收益稳定。
3. P2：做 Codex provider 并发闸门，配置默认关闭；RPi 上开启 `codex: 12`。
4. P5：提高日志保留和版本可观测性。
5. P3：只有在 P1/P2 后仍有少量 EOF 时，再做精细 retry。
6. P4：需要更高吞吐时再加账号扩容。

## 本轮实现范围

本轮按低风险优先实现 P0 / P1 / P2，并把 P5 的生产配置建议纳入部署：

- P0：新增 `test/translation_concurrency_probe.py`，用于按 `10 / 12 / 13 / 14 / 16 / 24` 并发档位请求 `/v1/chat/completions`，输出 status、耗时和错误摘要。
- P1：`NewProxyAwareHTTPClient` 继续每次返回独立 `http.Client`，但同一个 `proxy-url` / `direct` 会复用同一个 `http.Transport`，从而保留连接池，减少高并发时的 SOCKS/TLS 重建压力；无 proxy 且 context 注入 `RoundTripper` 时仍优先使用调用方 transport。
- P2：新增 `upstream-concurrency` 配置。默认 `default: 0` 保持不限流；生产可设置 `providers.codex: 12` 和 `queue-timeout-seconds: 30`，让突发请求在 CPA 内排队，排队超时返回明确的 `503 upstream_concurrency_queue_timeout`，而不是继续把突发直接打到上游。
- P5：现网部署时同步保留永久日志配置，并把 `error-logs-max-files` 维持为永久保留口径，便于后续分析错误比例。

P3 暂不做。原因是 retry 会放大请求量，必须先有 P2 闸门保护；如果 P1/P2 后仍有少量 EOF，再只针对“未向下游输出前的 transport EOF”做短 jitter retry。

## 生产配置建议

短期不改代码时：

- Immersive Translate 并发建议控制在 `8-10`。
- 不建议设到 `14+`。
- `request-retry` 不建议盲目增大。

代码优化部署后：

```yaml
upstream-concurrency:
  providers:
    codex: 12
  queue-timeout-seconds: 30
error-logs-max-files: 100
```

## 验收标准

部署前：

- 记录当前版本和配置。
- 保留一组 `10 / 13 / 14 / 16` 并发基线。

部署后：

- `/healthz` 正常。
- `/v1/models` 正常。
- 并发 `10 / 13`：全 200。
- 并发 `16`：不再出现 EOF 雪崩；应排队后全部成功，或少量请求因明确 queue timeout 返回 503。
- 并发 `24`：不出现大批 EOF；如果超过队列等待上限，错误应是明确的 queue timeout。
- 主日志能看到排队等待统计，而不是只有上游 EOF。
- 翻译插件实际使用 10 分钟内不再出现成批 `500 EOF`。

## 当前判断

这件事不是“纯上游问题”，也不是“纯代码 bug”。更准确的判断是：

- 上游单账号/单代理突发并发窗口是真实瓶颈。
- CPA 当前缺少对这个瓶颈的保护，属于可靠性设计缺口。
- `NewProxyAwareHTTPClient` 每请求重建 transport 属于代码层可优化点，会放大突发连接压力。
- 最有效的修复是“transport 复用 + 服务端并发闸门 + 更好的日志”，而不是单纯加 retry 或继续提高插件并发。

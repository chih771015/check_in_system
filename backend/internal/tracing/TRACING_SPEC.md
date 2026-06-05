# tracing — 規格（輕量）

> 對應檔案：`backend/internal/tracing/tracing.go`
> 上層：[ARCHITECTURE_SPEC.md](../../../ARCHITECTURE_SPEC.md)

## 1. 定位與職責
把 OpenTelemetry 全域 TracerProvider 接到 Jaeger（OTLP/gRPC）。安裝成 global provider 後，所有用 `otel.Tracer(...)` 的函式庫（gin middleware、gorm plugin、otelhttp transport）自動沿用。

## 2. 對外契約
`Init(ctx) (Shutdown, error)`：建立 exporter + provider + propagator，回 `Shutdown(ctx)`（程序結束時 flush）。

## 3. 行為與設定
| env | 預設 | 作用 |
|-----|------|------|
| OTEL_TRACES_EXPORTER | — | `=none` → 完全停用，回 no-op Shutdown（E2E compose 用此，避免 retry 洗 log）|
| OTEL_EXPORTER_OTLP_ENDPOINT | jaeger:4317 | OTLP gRPC 接收端 |
| OTEL_SERVICE_NAME | translator-checkin | service tag |
| DEPLOY_ENV | dev | deployment.environment |

- Sampler：開發用 `AlwaysSample()`（每請求都取樣）；流量上來改 ratio-based。
- Batcher timeout 5s；insecure gRPC（同 Docker 網路）。
- Resource 直接 `NewWithAttributes`（不 merge `resource.Default()`，避免 semconv schema URL 衝突）。

## 4. 不變式
| 不變式 | 保證 |
|--------|------|
| collector 不可達不得使程式崩潰 | 機制保證（main 對 Init error 只 log，繼續啟動；exporter 背景重試）|
| span 不外洩 PII | 由 [cmd/server](../../cmd/server/SERVER_SPEC.md) 的 gorm `WithoutQueryVariables` + `scrubSensitiveSpanAttributes` 保證（非本檔）|

## 5. 協作者
被 [cmd/server](../../cmd/server/SERVER_SPEC.md) `main` 初始化；span 由 gin/gorm/otelhttp/各 service 自動產生。背景見 [CHANGELOG-P2-JAEGER.md](../../../CHANGELOG-P2-JAEGER.md)。

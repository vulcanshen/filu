---
description: filu project conventions (Go / Bubble Tea)
globs: *
---

# filu Project Rules

filu 是 kbu `u`-family 的成員,共用 [Vulcan's TUI Design Principle](https://github.com/vulcanshen/thoughts/blob/main/vtp.md) 與技術棧。設計權威見 `.forge/meta/IDEA.md`,以及 kbu repo 的 `docs/kbu-implementation.md`(平行實作參照)。

## Code Quality
- `gofmt` / `go vet` 乾淨才算完成。
- 遵循 Effective Go 慣例;命名、錯誤處理比照 kbu 既有 code。
- 平台分岔操作(metadata / hidden / roots / trash / open)一律走 platform interface,unix 實作用 build tag(`_darwin.go` / `_linux.go` 或 `//go:build darwin || linux`);**不寫 Windows 實作**(`GOOS=windows` 應編譯失敗,這是刻意的)。
- 保持 `CGO_ENABLED=0` 可靜態編譯(同 kbu);需要 cgo 的方案(如 macOS Cocoa trash)先討論。

## Testing
- 用 table-driven test(比照 kbu),`go test ./...` 綠燈。
- Bubble Tea model 用 programmatic model test(送 msg、斷言 state / render),不靠真終端。

## Commits
- Conventional Commits(`feat:` / `fix:` / `refactor:` / `docs:` …,比照 kbu)。
- 每次改動可追溯到 IDEA.md 的某個功能或決定。

## Async / UI
- 長時操作(複製、搬移、watch)跑 goroutine,進度/事件經 channel → `tea.Msg` 餵回 UI,比照 kbu 的 log-stream / watch / PTY 套路。

## ZLC 紀律(來自設計原則)
- core-key 只有 4 個(Tab/Enter/Esc/Space)+ `?`;新動作先判 contextual(→ Space menu)還是 non-contextual(→ `?`)。
- letter hotkey ⊆ Space menu(完整性:光靠 Space 就能做完該 focus 的所有 contextual 動作)。
- 一元素一語意(§B 專職化);明度當 z-axis;popup 走四類 taxonomy。

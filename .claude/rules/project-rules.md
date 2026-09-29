---
description: filu project conventions (Go / Bubble Tea)
globs: *
---

# filu Project Rules

filu 是 terminu family 的成員,遵循 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.22/principle)(tdp),與 kbu 共用技術棧。設計權威見 `.forge/meta/IDEA.md`;開發者備忘(運作方式、設計決定、偏離 tdp)在 `docs/dev-remarks.md`,尚未符合 tdp 的地方在 `docs/filu-terminu-fix.md`。

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

## tdp 紀律
- core key(`Tab` / `Enter` / `Esc` / `Space` / `?` / `q`)的意義全 app 不變(tdp K1);新動作先依作用對象判 item / panel / global operation(tdp P3),item / panel 放進 Space menu 對應的區,global 放進 Space menu 最後那一列打開的 global operation popup(tdp M2、M4);`?` 只讀,是 key reference(tdp K6)。
- letter hotkey 是 menu 某一列的捷徑,只能靠熱鍵觸發的動作是違反(tdp M3)。
- 一元素一語意(tdp P4);明度當 z-axis(tdp D2);每個 popup 只屬於一類(tdp F1)。
- 有意不照 tdp 做的,寫進 `docs/dev-remarks.md`「偏離 tdp」並附理由;沒寫理由的列進 `docs/filu-terminu-fix.md`。

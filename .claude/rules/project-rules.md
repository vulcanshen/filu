---
description: filu project conventions (Go / Bubble Tea / terminu-go)
globs: *
---

# filu Project Rules

filu 是 terminu family 的成員，遵循 [terminu design](https://github.com/vulcanshen/terminu)（tdp，目前跟到 v0.11.5），畫面建在家族共用的
[terminu-go](https://github.com/vulcanshen/terminu-go) 上（Bubble Tea v1）。現行的權威是 `docs/dev-remarks.md`（運作方式、設計決定、
暫定）與 README；`.forge/meta/IDEA.md` 停在 v0.2.0 以前，只當演化史讀。

## Code Quality
- `make check` 通過才算完成：fmt-check、vet、`go test -race`，有 `go.work` 時加 `check-lib`（關掉 `go.work`、照 `go.mod` 的 terminu-go 版本編譯與測試）。
- 遵循 Effective Go 慣例；命名、錯誤處理比照 filu 既有的 code。
- 平台分岔操作（metadata / trash / open）一律用 build tag（`_darwin.go` / `_linux.go` 或 `//go:build darwin || linux`）；**不寫 Windows 實作**（`GOOS=windows` 應編譯失敗，這是刻意的）。
- 保持 `CGO_ENABLED=0` 可靜態編譯；需要 cgo 的方案（如 macOS Cocoa trash）先討論。
- `go.mod` 寫發布過的 terminu-go 版本，不用 `replace`；本機用不進版控的 `go.work` 接 `../terminu-go`。

## Testing
- 用 table-driven test，`make check` 綠燈。
- 畫面用 programmatic test：`internal/ui/harness_test.go` 把按鍵送進 `App.Update`、跑完它發出的 command、讀 `Frame()`，不靠真終端。
- 整個畫面的跨尺寸、跨寬度測試是 app 的事（`screen_test.go`：每一列都剛好是終端機寬，icon 一格／兩格、powerline 符號一格／兩格，
  filu 會開的每一類 popup）；單獨量 popup 的框歸 lib（tdp L4、E5、E8）。
- 修一個錯先寫會紅的測試；改動行為時做 mutation（改回舊行為要紅）。

## Commits
- Conventional Commits（`feat:` / `fix:` / `refactor:` / `docs:` …）。
- 每次改動可追溯到 tdp 的條目、`docs/dev-remarks.md` 的決定，或 user 的要求。

## Async / UI
- 長時操作（複製、搬移、打包、fd、ripgrep、目錄監看）跑 goroutine，結果經 channel 送回 `App.OnMsg`（terminu-go），不在 update 裡等。

## tdp 紀律
- **畫面是 terminu-go 的**（tdp E8）：Space menu、key reference、popup、confirm、input、toast、finder、viewer、PTY、splash、寬度都用
  lib 的，lib 做得到的不自己再做一份。lib 做不到或做錯，先用最合理的暫時做法，程式標 `TODO(lib FL<n>)`、`TODO(tdp FL<n>)`，寫進
  dev-remarks，並回報 terminu。
- core key（`Tab` / `Enter` / `Esc` / `Space` / `?` / `q` / `1`–`9`）的意義全 app 不變（tdp K1）；`j k u d g G h l` 留給移動，任何動作都不佔（tdp K12）。
- 新動作先依作用對象判 item / panel / global operation（tdp P3），登記成 `terminu.Action`：item、panel 放在 panel 的 Actions，global 放在
  `App.Global`（tdp M2、M4）；每個動作都要能從 menu 找到，letter hotkey 只是 menu 某一列的捷徑（tdp M3）；label 照 tdp M5 寫。
- 會先問的動作用 `Action.Ask`，confirm 才會疊在開它的 Space menu 上（tdp F4、T1）。
- 一元素一語意（tdp P4）；亮的地方就是有 focus 的地方（tdp P6、F8）；每個 popup 只屬於一類（tdp F1）。
- 設定、狀態、資料、快取照 tdp E3 放（用「刪掉會怎樣」分）；環境變數寫成 `FILU__<名字>`（tdp E2）。
- splash 不揭露：不寫進 README、CHANGELOG、release notes、menu、key reference（tdp S2）。
- 文件與程式註解引用 tdp 一律寫 `tdp` 加編號：rules 寫條目（`tdp K3`），元件與 project 寫節編號（`tdp panel.1`），整個元件都用到時寫檔名（`tdp confirm`）；不寫 `Rules X`、不寫路徑。
- tdp 寫「app 可以自行決定」的地方，filu 怎麼定寫進 `docs/dev-remarks.md`；tdp 沒放行、filu 又照不了的，當成 tdp 的缺口回報 terminu（tdp P5），不自己另做一套。

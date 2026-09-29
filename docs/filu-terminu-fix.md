# filu — terminu fix

filu 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.20/principle)（tdp v0.1.20）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與 `docs/dev-remarks.md` 裡描述該行為的段落。

盤點日期：2026-09-29。依據 filu `main` 的 `4814c5f`（已對齊 v0.1.19，工作區乾淨）。**這一輪只對 v0.1.19 → v0.1.20 的改動**
（`git -C ~/Documents/sideproj/terminu diff v0.1.19 v0.1.20 -- principle/`）：

- K11 / D3：模式名夾在兩個框線接頭之間（雙線 `╡Drag╞`、單線 `┤Visual├`），模式色加粗、盡量一個詞，放不下先截標題，panel 膠囊
  跟著外框換色（照 kbu `248f883`，user 要求寫回 tdp）。
- D6：icon 寬度量的是游標實際前進幾格（拿掉 Maple Mono NF CN 這個例子，filu 第六輪回報的），補上參考實作的完整清單 —— 就是 filu
  第六輪整理的照搬清單。

tdp 連結（兩份 README、`docs/dev-remarks.md` 兩處、`.claude/rules/project-rules.md`）已由 terminu session 從 v0.1.19 改成 v0.1.20，
只改網址，跟這份清單一起留在工作樹，還沒 commit。


## 先看

- 清單與改釘的連結先一起 commit，再動程式；commit 只加自己改的路徑。
- 修完的一條補 model test、做 mutation；同一個 commit 同步 README 兩份與 dev-remarks，CHANGELOG 記 `[Unreleased]`。
- **修完拿 v0.1.20 全文再逐條對一次**，修完刪掉這份清單，「已經符合」搬進 dev-remarks「對照 tdp 時確認過的」。
- 不 push、不發版。把這一輪寫進 `~/Documents/sideproj/terminu/.local/family-fix/filu/README.md`（加一節「第七輪：跟上 v0.1.20」）。


## 1. 選取模式的模式名沒有夾在框線接頭之間 —— K11、D3（v0.1.20）

**現況**（`internal/ui/popup.go` `drawPopupBoxMode()`）：上框是 `╭─` + 標題 + 橫線 + 模式名 + `─╮`，模式名直接接在橫線後面，沒有
接頭：`╭─ <標題> ───────────Selection─╮`（`detailyank.go` 的呼叫端傳 `m.title`）。名字 `Selection`（`selectkeys.go` 的 `selectModeName`）已經是一個詞；名字用 `tStyle`
（框色加粗），選取中框是 Yellow（`modeColor`），所以名字已經是模式色加粗；窄時先截標題、名字留著 —— 這幾點已經符合。

**規則**：K11（v0.1.20）—— 模式名夾在兩個框線接頭之間，像框上嵌了一個標籤。D3 —— 接頭跟框同色、線型跟著框：雙線框用 `╡` `╞`，
單線框用 `┤` `├`；模式名用模式色加粗；盡量一個詞；放不下時先截標題，模式名留著。

**怎麼改**：popup 是單線框，名字畫成 `┤Selection├`：接頭用 `bStyle`（框色），名字照舊 `tStyle`；`room` 多扣兩個接頭的寬度
（`dispWidth(mode) + 3`：兩個接頭加名字後那一格 `─`）。結果：`╭─ <標題> ─────────┤Selection├─╮`。

- 測試：選取模式的上框含 `┤Selection├`；接頭是框色、名字是 Yellow 加粗（量名字本身的 SGR，不量整行）；上框寬度不變（L4），
  icon 佔兩格時也不變（D6，`d6_test.go` 的量法）；窄框時標題先截、名字與兩個接頭都在。
- mutation：拿掉接頭；接頭改成名字的顏色；`room` 不扣接頭的寬度。
- 參考 kbu `internal/ui/app.go` 的上框標籤（`248f883`：`jl, jr := "┤", "├"`，雙線時換成 `╡` `╞`）。


## 已經符合、不用修的（對照 v0.1.20 的改動）

- **D6（v0.1.20 的寫法）**：filu 就是參考實作。CPR 探測量的是游標實際前進幾格（`iconwidth_unix.go` 的 `DetectIconWidth()`），
  `FILU_ICON_WIDTH` 手動覆寫、`filu iconwidth` 查看；filu 沒有 Windows 版，探測只在 unix 的寫法不影響它。`internal/ui` 除了
  `width.go` 找不到 `lipgloss.Width` 等呼叫（viewport 的 `ansi.Cut()` 是切字串自己的格數，第六輪已註明例外）。
- **D3 的模式名一個詞、先截標題**：`Selection` 一個詞；`drawPopupBoxMode()` 窄時先截標題。
- **D3 的膠囊跟著換色**：filu 的模式在 popup 裡，popup 沒有 `[N] label` 膠囊，不適用。


## 待確認

沒有。

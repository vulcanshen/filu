# filu — terminu fix

filu 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.21/principle)（tdp v0.1.21）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與 `docs/dev-remarks.md` 裡描述該行為的段落。

盤點日期：2026-09-29。依據 `main` 的 `ec5b405`（已對齊 v0.1.20，工作區乾淨）。**這一輪只對 v0.1.20 → v0.1.21 的改動**
（`git -C ~/Documents/sideproj/terminu diff v0.1.20 v0.1.21 -- principle/`）：

- D5：選取文字的模式照 vim 移動 —— `h/j/k/l`、`w/b/e`、`0/$`、`gg/G`、`u/d`（user 要求寫回 tdp，filu 轉達）。
- D6：環境變數命名 `<APP>__<NAME>`（app 名後兩個底線、變數名全大寫單底線分隔），共用名 `<APP>__CONFIG` / `__STATE` / `__DATA`
  / `__CACHE`（都指向**目錄**）/ `__ICON_WIDTH`；給別的程式讀的變數例外；**改名不留舊名**（user 2026-09-29 裁定）。
- D6：疊 popup 時 popup 比畫面寬或高（調整終端機大小的那一格）：起點取 0、超出的部分切掉，不可以 panic（kbu、locku 照搬時抓到，
  filu 的參考實作有這個 bug）。

tdp 連結（README 兩份、docs、`.claude/rules`）已由 terminu session 從 v0.1.20 改成 v0.1.21，只改網址，跟這份清單一起留在工作樹，
還沒 commit。


## 先看

- 清單與改釘的連結先一起 commit，再動程式；commit 只加自己改的路徑。
- 每修一處補 model test、做 mutation；同一個 commit 同步 README 兩份與 dev-remarks；CHANGELOG 記 `[Unreleased]`。
- **環境變數改名是破壞性改動**：CHANGELOG `[Unreleased]` 要寫一條「改名，舊名不再讀」並列出新舊對照；CHANGELOG 裡已發版的舊段落不改。
  `.checkpoints/` 是本機筆記，不用改。改完 `grep -rn '<舊名>'` 除了 CHANGELOG 舊段落應該是零。
- **修完拿 v0.1.21 全文再逐條對一次**，修完刪掉這份清單。不 push、不發版；把這一輪寫進 terminu `.local/family-fix/filu/README.md`。


## 1. `compositeDisp()` 在 popup 比畫面大時 panic —— D6（v0.1.21，參考實作，先做）

**現況**（`internal/ui/width.go`）：`compositeDisp()` 用 `clampSpan(placeOffset(...)+off, bgW-fgW)` 算起點。popup 比畫面寬（`fgW > bgW`）
或高時，`bgW-fgW` 是負的，`clampSpan()` 把上下限對調，回傳負的起點：x 負的讓 `strings.Repeat(" ", x-dispWidth(left))` panic，y 負的讓
`bgLines[y+i]` 越界 panic。`overlay.Composite` 原本只是讓那一列超出畫面。調整終端機大小的那一格（popup 還是用舊尺寸畫的）就會遇到，
整個 TUI 當掉。kbu（`TestD6_CompositeDisp`）、locku 照搬時抓到、在自己那份修掉；sshu、webu 照搬的是 filu 這一版，也有同樣的 bug。

**規則**：D6（v0.1.21）—— 疊 popup 時 popup 可能比畫面寬或高（調整終端機大小的那一格還是舊尺寸）：起點取 0、超出畫面的部分切掉，
**不可以 panic**；測試的邊界要含這種情況。

**怎麼改**：起點最小取 0（`x`、`y` 各自 `max(0, …)`，不再對調上下限）；popup 的列比畫面寬時，放上去之前先切到畫面寬（`dispClip(line, bgW-x)`）；
比畫面高時多出的列不畫（迴圈已有 `y+i >= bgH` 的 break）。照 kbu 的做法（kbu `internal/ui/width.go` 的 `compositeDisp()`，`b0ccea9`）。
測試：`compositeDisp()` 的邊界補三種 —— popup 比畫面寬、比畫面高、兩者都大；不 panic、結果每一列剛好畫面寬、列數等於畫面高。可以直接照搬
kbu 的 `TestD6_CompositeDisp`。mutation：拿掉 `max(0, …)`、拿掉切寬。**sshu、webu 等這一條做完再照搬**，把 commit 寫進紀錄。


## 2. 環境變數沒照家族命名 —— D6（v0.1.21）

**現況**：filu 讀的變數都是單底線，而且 `FILU_CONFIG`、`FILU_STATE` 指向**檔案**：

| 現在 | 意思 | 改成 |
|---|---|---|
| `FILU_CONFIG`（`internal/ui/config.go`） | 設定**檔**的路徑 | `FILU__CONFIG`：設定**目錄**，設定檔照原本的檔名放在裡面 |
| `FILU_STATE`（`internal/ui/persist.go`） | 狀態**檔**的路徑 | `FILU__STATE`：狀態目錄，狀態檔照原本的檔名放在裡面 |
| `FILU_LAST_DIR_FILE`（`cmd/filu/main.go`、`internal/ui/quit.go`） | 離開時寫最後目錄的檔案 | `FILU__LAST_DIR_FILE` |
| `FILU_REPAINT`（`internal/ui/app.go`） | 除錯用 | `FILU__REPAINT` |
| `FILU_ICON_WIDTH`（`internal/ui/iconwidth_unix.go`） | icon 寬度覆寫 | `FILU__ICON_WIDTH` |

**規則**：D6（v0.1.21）—— `<大寫 app 名>__<變數名>`，變數名全大寫、單字之間一個底線；app 自己讀的變數（含測試用、傳給自己子程序的）
都照這個寫。共用名：`<APP>__CONFIG`（設定目錄）、`<APP>__STATE`（狀態目錄）、`<APP>__DATA`（資料目錄）、`<APP>__CACHE`（快取目錄）、
`<APP>__ICON_WIDTH`。給別的程式讀的變數例外。**改名不留舊名**（user 裁定）。

**怎麼改**：程式照上表改名；`FILU__CONFIG`、`FILU__STATE` 改成目錄語意（讀到的是目錄，檔名 filu 自己接）。一起改的地方：`.local/demos/` 的六個
tape（`demo-basics`、`demo-finders`、`demo-preview`、`demo-marks`、`demo-shell`、`demo-favorites`，現在設的是檔案路徑，改成目錄）、
`docs/dev-remarks.md`、README 兩份（若有提）、測試。shell 整合若會設 `FILU_LAST_DIR_FILE`（README 的安裝段或 shell 函式），一起改，並在
CHANGELOG 寫明使用者要更新 shell 設定。改完 `grep -rn 'FILU_[A-Z]' --include=*.go --include=*.md --include=*.tape --include=*.sh .`
除了 CHANGELOG 舊段落是零。


## 已經符合、不用修的（對照 v0.1.21 的改動）

- **D5 選取模式的移動**：yank viewport 的選取有 `h/j/k/l`、`w/b/e`（第七輪 `ec5b405`）、`0/$`、`gg/G`、`u/d`（`detailyank.go`、`selectkeys.go`）。


## 待確認

沒有。

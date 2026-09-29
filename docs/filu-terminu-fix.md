# filu — terminu fix

filu 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.19/principle)（tdp v0.1.19）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與 `docs/dev-remarks.md` 裡描述該行為的段落。有意不修的，改寫成
`dev-remarks.md`「偏離 tdp」的一條並附理由。

> **v0.1.19（2026-09-29，這份清單寫完後才出）**：L5 —— focus 不能只靠顏色分辨（模式會換框色），家族預設雙線；K10 ——
> 子程序還沒準備好時可以不轉送一般的鍵，但 `Ctrl-C` 照樣轉送。本清單的每一條已照 v0.1.19 重新核對過。

盤點日期：2026-09-29。依據 filu `main` 的 `28886d9`（已對齊 v0.1.17，工作區乾淨）。**這一輪只對 v0.1.17 → v0.1.18 的改動**
（`git -C ~/Documents/sideproj/terminu diff v0.1.17 v0.1.18 -- principle/`）：D6 icon 的實際寬度、F1 / D3 finder 的 focus、K11 / D2
模式標示、D2 失焦 panel 的 hint 顏色、D3 下框 hint 放不下時整組捨棄、K10 / K9 PTY。其中 D6 與 D3 的 hint 兩條是 filu 第五輪回饋給
tdp 的。每一條都拿程式碼逐處核對過（照內容，不照行號）；位置寫檔案與函式。icon 寬度是在 scratch 複本裡把 `iconCells` 設成 2、
100 × 40 實際 render 量的（沒有動 filu 的工作樹）。

工作樹裡另有 terminu session 的改動（還沒 commit）：README 兩份、`docs/dev-remarks.md`（兩處）、`.claude/rules/project-rules.md` 的
tdp 連結從 v0.1.17 改成 v0.1.19，只改網址。


## 先看

- **第 1 條（icon 寬度）先做、先 commit。** filu `internal/ui/width.go` 是 D6 的參考實作，其他四個 app 等 filu 補完才照搬。做完在
  terminu 的 family-fix 紀錄寫明「照搬哪些函式」（見第 1 條最後）。
- **清單先 commit，再動程式。** 連同上面那幾處連結一起讀過 diff 再 commit；commit 只加自己改的路徑。
- **每修一處補 model test，並做 mutation**：把修正單獨改回舊行為，確認對應的測試會紅（編譯不過、被上限夾住、fixture 太單純都
  不算抓到；預期值寫死，不用被測的常數或函式去算）。
- **同一個 commit 同步 README 兩份與 `docs/dev-remarks.md`**；CHANGELOG 記在 `[Unreleased]`。
- **建議順序**：第 1 條 → 第 5 條（跟第 1 條都動 `drawPopupBoxPad()`，也動 `keyLegend()`）→ 第 4 條（`keyLegend()` 的失焦配色）→
  第 3 條（`drawPopupBoxPad()` 的上框右側）→ 第 2 條 → 第 6 條。
- **修完拿 v0.1.19 的 rules、defaults、術語全文逐條再對一次。** 修完刪掉這份清單，「已經符合」搬進 `dev-remarks.md`「對照 tdp 時
  確認過的」的 v0.1.18 小節。
- **不 push、不發版**（家族全部穩定才一起發）。
- 把這一輪寫進 `~/Documents/sideproj/terminu/.local/family-fix/filu/README.md`：加一節「第六輪：跟上 v0.1.18」（改了什麼、commit、
  裁定、教訓），並更新那份的「回饋給 tdp 的」（D6、D3 hint 兩條已被 v0.1.18 採納）與「發布」段。


## 已經符合、不用修的（對照 v0.1.17 → v0.1.18 的改動）

- **L5、K10（v0.1.19）**：focus 的 panel 畫雙線 `╔═╗`、失焦圓角（`view.go` 的框線選擇），不只靠顏色；shell PTY 一開就轉送按鍵，`Ctrl-C` 是 shell 的。
- **D6 的探測**：`cmd/filu/main.go` 在 `tea.NewProgram` 之前呼叫 `ui.DetectIconWidth()`（CPR 探測，失敗維持 1，`FILU_ICON_WIDTH` 可覆寫）；
  `filu iconwidth` 印出結果。panel 那一半（清單、麵包屑、tab bar、Marks、Favorites、Tasks、footer、`joinH()` / `joinV()`）都走
  `dispWidth()`；`dimANSI()` 只改 SGR、不量寬度。
- **F1 / D3 finder 打字時**：篩選列亮、cursor 列是 Subtext1 的淡反白（`handColor`），已經是 D3 的樣子；清單態不畫游標。
- **K11 focus 的 panel 保留線型**：filu 的模式在 popup 裡，不適用。
- **D2 失焦 hint**：`[1]` 失焦時不顯示 hint、`[2]` 沒有 hint、zoom 時 panel 一定是 focus 的；只有 `[3]` 要改（第 4 條）。
- **K10 子程序還沒準備好收鍵**：filu 的 shell 同步啟動，`ptyPopup.update()` 在 `ptmx` 建好之前不轉送；條文是「可以」，不用改。出口鍵
  `Alt-Esc` 照樣有效、常駐揭露（`ptyExitHint`）。
- **K9 PTY 裡 `q`、`Ctrl-C` 屬於子程序**：`Update` 的 PTY 分支排在離開流程之前，沒有框疊在 PTY 上時全部送進 shell
  （`TestPtyKeysBelongToShell`）；`Alt-Esc` 的 confirm 疊在 PTY 上時 focus 已經不在 PTY，`q` / `Ctrl-C` 進離開流程，也符合。


## 待確認

沒有。

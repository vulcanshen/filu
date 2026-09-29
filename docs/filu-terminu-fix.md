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


## 2. finder 看不出 focus 在哪一邊 —— F1、D3

**現況**：filu 只有一個 finder 元件 `searchModel`（`search.go`），Search（檔名，fd）、Find（內容，rg）、Goto（home 底下的目錄）、
New tab 的 Search 都是它。`listColumn()` 的 cursor 列：打字時底色 `handColor`（Subtext1 `#bac2de`）、深色字 —— 就是 D3 說的「淡的反白」；
`Tab` 到清單後底色換成 `focusColor`（Blue `#89b4fa`）、不粗體。`inputBar()` 的篩選列兩個階段畫法一樣：Peach 粗體的 `inputGlyph`、
預設色的 query、Overlay0 的計數；只有游標 `█` 在清單態不畫。

**規則**：F1（v0.1.18）—— finder 這種依階段換類別的 popup，**focus 在哪一邊要看得出來**：只有拿鍵的那一邊是亮的。D3 —— 打字時
篩選列亮、清單的 cursor 列是淡的反白；`Tab` 到清單後，篩選列整列用灰色（Overlay0，D2 的暗字）畫，不用 F8 的淡化、也不畫反白與游標；
清單的 cursor 列換成 popup 層色底加深色粗體字（跟 menu 的 cursor 列一樣）。

**怎麼改**：

- 打字時維持現狀。
- 清單態：`inputBar()` 整列（glyph、query、計數）都畫 `dimColor`（Overlay0 `#6c7086`），是單一顏色，不是 `dimANSI()`；游標照舊不畫。
  `listColumn()` 的 cursor 列改成 `popupLayerColor(m.anim.layer)` 底、`baseHex` 字、粗體（跟 `spaceMenu.renderFull()` 的 `cursorStyle`
  同一套）。
- kbu 是這條的出處，但它現在用的是 `dimANSI()` 淡化、也要改；filu 照 D3 的文字做，不要照搬 kbu。
- 文件：dev-remarks「Finder」那段補一句 focus 的畫法；`listColumn()` 裡「turns blue (focusColor)」那段註解跟著改。README 可以不動。
- 測試（truecolor）：打字時篩選列的 glyph 是 Peach、cursor 列底色 `#bac2de`；`Tab` 之後篩選列每一格前景都是 `#6c7086`、沒有反白也沒有
  `█`，cursor 列底色等於這個 popup 的層色、粗體。Goto 模式也跑一次（同一條路徑，但守住）。mutation：篩選列改回原色、cursor 列改回 Blue，
  各自要紅。


## 3. 選取模式沒有標示自己 —— K11、D2

**現況**：照術語「模式」逐個找，filu 只有一個模式：yank viewport 的選取（`detailYank.visual`，`v` 進、`v` / `Esc` 出）。zoom 是版面
切換、finder 打字 / 清單是階段（F1）、`pendingG` 是 chord，都不是模式。選取中 `detailyank.go` `renderFull()` 的框照樣用這一層的層色
（`popupLayerColor(m.anim.layer)`），標題照樣是 `Yank: Preview`，唯一的差別是下框 hint 換成 `y:copy Esc:leave ?:keys`。

**規則**：K11（v0.1.18）—— 模式名一律顯示在模式所在的框的**上框右側**，外框換成模式色；離開模式就恢復。focus 的 panel 在模式裡照樣
是 focus 的線型，只換顏色（filu 的模式在 popup 裡，這半句不適用）。D2 —— 模式色是 Yellow `#f9e2af`（外框與右上角的模式名）。

**怎麼改**：

- 選取中：外框（含標題）畫 Yellow，上框右側放模式名（名字由 filu 定，例：`Selection`，跟 `?` 的標題 `Selection keys` 一致），也是
  Yellow。離開選取就回到層色、拿掉模式名。
- `drawPopupBoxPad()` 目前只有左邊的標題，要能在上框右側放一段（橫線長度用 `dispWidth()` 扣掉兩段）；寬度不夠時先截左邊的標題，
  模式名保留。框的寬度不跟著變（L2）。
- `?` 疊在選取模式上時，底下的 Yellow 框照 F8 淡化（`dimANSI()` 自然做到），不用另外處理。
- 順帶一提（不是這輪的改動）：選取反白現在是 Lavender（`selStyle` 的 `userColor`），D2 的「選取」預設是 Yellow；框變 Yellow 之後兩種顏色
  會同時出現，改完印畫面看一次。
- 文件：dev-remarks「Preview yank viewport」那段補一句；README 兩份「Preview and copy」／「預覽與複製」可以寫「選取中框變黃、右上角寫著
  模式名」；CHANGELOG 記一條。
- 測試（truecolor）：選取中上框在 `╮` 前面是模式名、框線每一格是 `#f9e2af`；離開後框線是層色、沒有模式名；兩種狀態框寬一樣；很窄的
  寬度下模式名還在。mutation：拿掉顏色、拿掉模式名、離開時不恢復，各自要紅。


## 6. yank viewport 在中文字上把游標與選取畫錯位置 —— L4、D6（不是這一輪的改動）

核對寬度時看到的，v0.1.17 以前就這樣。

**現況**：`detailyank.go` 的 `overlayCursorOnStyledLine()` 與 `overlaySelectionOnStyledLine()` 拿到的位置是 rune 的序號（`cursorCol`、
選取的起訖），卻交給 `ansi.Cut()` 當格數用；中文字一個 rune 佔兩格，位置就錯。scratch 實測：`中文abc` 游標停在 `a` 上畫成 `中a文abc`；
選取 `ab` 畫成 `中ababc`（字被重複畫出來）。複製出去的內容是對的（`selectionText()` 用 rune 取），只有畫面錯。

**規則**：L4 —— 每一列剛好等於終端機寬度；D6 —— 量寬度只走一套。

**怎麼改**：`ansi.Cut()` 之前把 rune 序號換算成那一段在字串裡的格數（前面幾個 rune 的 `ansi.StringWidth()` 加總 —— 切的是字串自己的
格數，要跟 `ansi.Cut()` 同一套量法，不是 `dispWidth()`）。測試：中文、中英混排、行尾，各寫死畫出來的純文字；mutation：拿掉換算要紅。
要不要這一輪修，filu 自己排；不修的話寫進 dev-remarks「已知的牆」。


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

# filu — terminu fix

filu 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.17/principle)（tdp v0.1.17）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與 `docs/dev-remarks.md` 裡描述該行為的段落。有意不修的，改寫成
`dev-remarks.md`「偏離 tdp」的一條並附理由。

> **v0.1.17（2026-09-29，這份清單寫完後才出）**：只補了 M5 三點，已照它調整本清單 —— menu 與 key reference **說明欄**裡提到的鍵
> 算句子，加方括號；**README** 內文的鍵用 Markdown code 標（`` `Enter` ``），不加方括號，鍵名與寫法照 M5；**別的工具自己的按鍵**
> （tmux 的 `prefix l`、`C-a x`）照那個工具的寫法。其餘條目照 v0.1.14–v0.1.16。

盤點日期：2026-09-29。依據 filu `main` 的 `f1919c2`（已對齊 v0.1.13，工作區乾淨）。**這一輪只對 v0.1.13 → v0.1.16 的改動**
（`git -C ~/Documents/sideproj/terminu diff v0.1.13 v0.1.16 -- principle/`）：

- v0.1.14：F1 / F8 / K11 的 toast、K10 / D5 的 PTY 出口鍵、術語「模式」與 zoom、M6 key reference 變暗。
- v0.1.15：M5 按鍵寫法全部定案（鍵名、依位置的寫法、hint / footer / key reference 的顏色），D1–D5 的例子跟著改。
- v0.1.16：D5 `Alt-Esc` 一律先 confirm；M6 描述別的 surface 的段落照亮顯示。

每一條都拿程式碼逐處核對過（照內容，不照行號）；位置寫檔案與函式。v0.1.13 以前的條文前四輪已全文對過，這輪沒有重新全文
盤點 —— 修完再拿 v0.1.17 全文對一次（見「先看」）。

工作樹裡另有 terminu session 的改動（還沒 commit）：README 兩份、`docs/dev-remarks.md`（兩處）、`.claude/rules/project-rules.md` 的
tdp 連結從 v0.1.13 改成 v0.1.17，只改網址。


## 先看

- **清單先 commit，再動程式。** 連同上面那幾處連結一起讀過 diff 再 commit；commit 只加自己改的路徑。待確認那題先問 user、
  寫回清單（它只卡第 2 條裡「說明欄的鍵」那一小塊，其他可以先做）。
- **每修一處補 model test，並做 mutation**：把修正單獨改回舊行為，確認對應的測試會紅（編譯不過、被上限夾住、fixture 太單純都
  不算抓到；預期值寫死，不用被測的常數或函式去算）。守舊行為的測試改寫成守新規則（改名、反轉斷言），不要刪。
- **同一個 commit 同步 README 兩份與 `docs/dev-remarks.md`**；CHANGELOG 記在 `[Unreleased]`。`[Unreleased]` 裡還沒發布的
  `Alt+Esc`、`Ctrl+C` 句子可以直接改寫；已發布的版本段不動。
- **建議順序**：第 1 條（PTY 的 confirm，動到路由與繪製順序）→ 第 2 條（按鍵寫法與顏色；PTY 下框與 confirm 的 hint 跟第 1 條同一處，
  先做第 1 條免得改兩次）→ 第 3 條（key reference 變暗；跟第 2 條都改 `helpPopup.renderFull()`，接著做）→ 第 4、5 條（文件）→ 第 6 條。
- **修完拿 v0.1.17 的 rules、defaults、術語全文逐條再對一次**（前四輪每次都在這一步多找到清單沒列的）。修完刪掉這份清單，
  「已經符合」搬進 `dev-remarks.md`「對照 tdp 時確認過的」的 v0.1.16 小節（前幾輪的做法）。
- **不 push、不發版**（家族全部穩定才一起發）。
- 把這一輪寫進 `~/Documents/sideproj/terminu/.local/family-fix/filu/README.md`：加一節「第五輪：跟上 v0.1.14–v0.1.16」（改了什麼、
  commit、裁定、教訓），並更新那份的「偏離 tdp」與「發布」段。


## 1. PTY 的 `Alt-Esc` 直接結束 shell —— K10、D5（已定案）

**現況**：`app.go` `Update()` 的 `tea.KeyMsg` 分支，splash 之後第一站是 `m.pty.isActive()`：`isExitKey()`（`Esc` + Alt）成立就直接
`m.pty.exit()`，`pty_unix.go` 的 `ptyPopup.exit()` 立刻 `Kill()` shell、播關閉動畫、reload 目錄。filu 的 PTY 只跑 `s` 開的 shell
（`buildShellCmd()`；`buildEditorCmd()` 只剩測試在用）。

繪製順序（`view.go` `popupLayers()`）：confirm 與 `help` 在 PTY 之下，`quitMenu`、`quitHelp` 在 PTY 之上。PTY 不在 `stackOrder()`
裡，`assignLayers()` 不算它，它的框固定畫第 1 層色（`renderPopup()` 的 `popupLayerColor(1)`）。

**規則**：K10 —— 家族的 PTY 出口鍵是 `Alt-Esc`（filu 已經是）。D5（v0.1.16）—— **`Alt-Esc` 一律先 confirm**：按了會讓 focus 離開
PTY 或結束子程序時，不論子程序留不留著，都先跳 confirm（`Enter` 離開、`Esc` 回到 PTY）。理由：終端機把 Alt 組合送成「`Esc` 加
那個鍵」，`Alt-Esc` 跟兩次 `Esc` 的 byte 一模一樣；app 忙的時候讀鍵的一端卡住，兩次 `Esc` 就會疊在一起被讀成 `Alt-Esc`（tdp 在
2026-09-29 用 bubbletea v1.3.10 實測：前面有鍵在排隊時，間隔 150ms 的兩次 `Esc` 也會黏在一起）。filu 用的正是 bubbletea v1.3.10：
在 shell 裡開 vim 改檔、連按 `Esc`，shell 會連同沒存的編輯一起被殺掉。F4 —— confirm 疊在 PTY 上，`Esc` 回到 PTY。

**已定案**（user，2026-09-29）：照做（v0.1.16 起 D5 本身就這樣要求）。

**怎麼改**：

- `Alt-Esc` 改成開 confirm（新增 `confirmEndShell`，照其他 confirm 重用 `m.confirm` 與 `confirmAction`）。問句寫出對象與後果
  （F6，例：`End the shell in ~/proj? Anything still running in it stops.`），動詞 `end`（下框 `Enter:end Esc:cancel`，寫法見第 2 條）。
  接受（`Enter` / `y`）才 `m.pty.exit()`；取消（`Esc` / `n`）關 confirm、回到 PTY，shell 照跑。
- **路由**：PTY 分支要讓位給疊在它上面的框。confirm 開著時，鍵照一般 popup 的順序走：toast 的 `Esc` 先收（F3）、`q` / `Ctrl-C`
  進離開流程（K9）、`?` 開 `Confirm keys`（K6）、其餘給 confirm。沒有框疊在 PTY 上時，才照舊把每個鍵（含 `Esc`、`Ctrl-C`、`q`）
  送進 shell。
- **繪製與層色**：confirm 與它的 `?` key reference 都要畫在 PTY 上面（現在兩者都在 PTY 之下，直接重用會被 PTY 蓋住）；confirm 用
  PTY 上一層的層色（D2：第 2 層），PTY 在它底下 dim（F8）。做法由 filu 決定，兩個要避開的坑：
  - 把 PTY 移到 `popupLayers()` 最底下最省事，但開 shell 那一刻，正在關的 Space menu 與 Shell confirm 會畫在 PTY 上、PTY 跟著
    暗一下（F8 的「最上層」看的是 `isActive()`，含關閉中）。改完把畫面印出來看。
  - 若把 PTY 放進 `stackOrder()` 好讓 `assignLayers()` 算到它，`clearStack()` 會直接關掉 PTY 的 animator、不經 `exit()`
    （shell 沒被殺、`stopPending` 沒設）。confirm 接受後本來就會呼叫 `clearStack()`。
- shell 在 confirm 開著時自己結束了（`ptyTickMsg` 看到 `done`），confirm 一起收掉：要結束的東西已經不在（T1）。
- `ptyExitHint` 照樣常駐在 PTY 下框（K10）；寫法與顏色見第 2 條。
- 文件：
  - README 兩份「Open, edit, and everything else」／「開啟、編輯,以及其他」那一條改成「`Alt-Esc` 先問，接受才結束 shell」。
  - dev-remarks「設計決定」PTY 那段改寫：出口鍵先 confirm 與理由（D5）、confirm 疊在 PTY 上、`Esc` 回 PTY、被讀成 `Alt-Esc` 的
    那兩個 `Esc` 不會送進 vim（回到 PTY 後要再按一次）。同一節「一律先 confirm」的清單加上 `Alt-Esc`。
  - dev-remarks「user 裁定的」：保留 2026-09-28 那條，補一條 2026-09-29 的新裁定（先 confirm）。
  - CHANGELOG `[Unreleased]` 的 `Alt+Esc` 那條（Added，還沒發布）直接改寫。
- 參考 kbu：`internal/ui/app.go` 的 `ptyLeaveRequestMsg` 分支（`ConfirmEndShell`，`m.confirm.SetLayer(m.popupDepth() + 1)`）與
  `ptyKillMsg`；紀錄在 terminu `.local/family-fix/kbu/README.md` 的 K10 那一列（`8f530ca`、`7e861dd`）。
- 測試（`pty_unix_test.go`）：
  - `TestPtyAltEscExits` 改寫成守新規則：`Alt-Esc` 開 confirm、`stopPending` 仍是 false、process 還活著；`Enter` 之後 shell 結束、
    popup 關掉；`Esc` 之後 confirm 關、PTY 還在、process 還活著。
  - 另補：confirm 開著時 `View()` 看得到問句，PTY 那幾格是 dim 過的顏色（F8，照 `f8_test.go` 開 truecolor 比對）；confirm 的邊框是
    第 2 層色（D2）；shell 自己結束時 confirm 跟著收。
  - `TestPtyKeysBelongToShell`（`Esc`、`Ctrl-C`、`q` 給 shell）照舊要綠。
  - mutation：`Alt-Esc` 改回直接 `exit()`、confirm 畫回 PTY 底下、層色不算 PTY，各自要紅。


## 2. 按鍵的寫法與顏色 —— M5、D1–D4（已定案）

**規則**（v0.1.15 定案，畫面上所有地方與 README 都一樣）：

- **鍵名**：鍵帽上的名字、大駝峰、不自創縮寫（`Esc` `Tab` `Enter` `Space` `Backspace` `Delete` `Home` `End` `PgUp` `PgDn`，方向鍵
  `↑ ↓ ← →`）；字母照實際大小寫（`q`、`A` 就是 `Shift-A`）；`Ctrl` 後面的字母一律大寫（`Ctrl-C`、`Ctrl-U`）；modifier 用 `-`；
  幾個鍵做同一件事用 `/`（`j/k`、`h/l`）；範圍用 `–`（`1–9`）。
- **依位置**：label（menu 的列、statusbar chip、panel 標題）照括號標記；**句子**（空狀態、toast、錯誤訊息）鍵一律加方括號
  （`Press [A] or [Space]`）；**hint、footer** 寫 `鍵:說明`，冒號前後不空格、項目之間一個空格（`j/k:move Enter:run Esc:close`），
  不再用 ` · `、兩個以上的空白、`=`；**key reference** 兩欄（鍵、說明），鍵欄不加括號、不加冒號。
- **顏色**（D2）：hint 與 footer 的鍵 Blue `#89b4fa`，冒號與說明 Overlay0 `#6c7086`；key reference 的鍵 Blue、說明 Text `#cdd6f4`。
- D1 footer：`Space:menu ?:help Tab/1–N:panels q:quit`；D3 confirm：`Enter:<動詞> Esc:cancel`；D4 menu：`j/k:move Enter:run Esc:close`。

**已定案**（user，2026-09-29，v0.1.15 回答了上一版清單的待確認）：footer 與下框的小寫 `enter`、`space`、`esc`、`tab` 改大駝峰；
toast 的 `(w)` 改 `[w]`；README 兩份照同一套鍵名。

以下是每一處「現在 → 改成」。hint 在框線上前後各留的一格空白照舊，表裡省略。

### 2a. hint 與 footer

popup 的下框（除了 PTY，全部經過 `popup.go` `drawPopupBoxPad()`）：

| 位置 | 現在 | 改成 |
|---|---|---|
| `spacemenu.go` `renderFull()`（Space menu、global operation、sort、Goto、Search chooser、Open in、Open with、quit picker 共用） | `j/k move · Enter run · Esc close` | `j/k:move Enter:run Esc:close` |
| `confirm.go` `renderFull()` | `Enter <verb> · Esc cancel` | `Enter:<verb> Esc:cancel` |
| `inputpopup.go` `hint()` | `Enter <verb> · Esc cancel` | `Enter:<verb> Esc:cancel` |
| `helppopup.go` 常數 `helpHint` | `j/k scroll · ? or Esc close` | `j/k:scroll ?/Esc:close` |
| `metapopup.go` 常數 `metaHint` | `j/k scroll · Esc close` | `j/k:scroll Esc:close` |
| `breadcrumbpopup.go` `renderFull()` | `j/k move   Enter jump   Esc close` | `j/k:move Enter:jump Esc:close` |
| `detailyank.go` `hint()`，選取中 | `y copy · Esc leave · ? keys` | `y:copy Esc:leave ?:keys` |
| `detailyank.go` `hint()`，平常 | `v select · y copy all · Esc close` | `v:select y:copy all Esc:close` |
| `search.go` `hint()`，清單態 | `j/k/u/d · Enter=go · Tab=input · Esc=close` | `j/k/u/d:move Enter:go Tab:query Esc:close` |
| `search.go` `hint()`，打字態 | `↑/↓ · Enter=go · Tab=list · Esc=close` | `↑/↓:move Enter:go Tab:list Esc:close` |
| `pty_unix.go` 常數 `ptyExitHint` | `exit or Alt+Esc to close` | `Alt-Esc:close`（`exit` 是打進 shell 的指令、不是鍵，放不進 `鍵:說明`；README 照舊寫可以打 `exit`） |

panel 下框與 footer（`view.go`，都經過 `keyLegend()` / `keyLegendGap()`）：

| 位置 | 現在 | 改成 |
|---|---|---|
| `listNavHint()`（`[1]`） | `enter into  esc back  jkud move  hl switch tab` | `Enter:into Esc:back j/k/u/d:move h/l:switch tab`（`jkud`、`hl` 是自創縮寫） |
| `marksHint()`（`[3]` Marks） | `p pick   m unmark   Z zip   C clear` | `p:pick m:unmark Z:zip C:clear` |
| `favoritesHint()`（`[3]` Favorites） | `o open in   D remove` | `o:open in D:remove` |
| `footerBar()` | `space menu   ? help   tab/1-3 panels   q quit` | `Space:menu ?:help Tab/1–3:panels q:quit`（`–` 是 en dash） |

顏色：

- `keyLegendGap()`：鍵 `focusColor`（Blue `#89b4fa`）、說明 `dimColor`（Overlay0 `#6c7086`），已經對；改成 `鍵:說明` 時冒號畫說明的
  顏色，項目之間一個空白（`[1]` 的兩格與其他的三格兩種間隔都不要了）。
- popup 的下框：`drawPopupBoxPad()` 用 `tStyle`（這一層的層色、粗體）把整條 hint 畫成同一個顏色 —— 鍵與說明分不開。改成跟
  `keyLegend()` 同一個 helper 產生已上色的 hint（鍵 Blue、冒號與說明 Overlay0），`drawPopupBoxPad()` 原樣放上去；`dispWidth()`
  與 `truncate()` 本來就認得 ANSI（`panelBoxHint()` 已經這樣用）。toast 傳的 `" "` 不受影響。
- PTY 的下框：`renderPopup()` 用 `ts`（第 1 層色、粗體）畫 `ptyExitHint`，同樣改用那個 helper。
- 底下幾層的 hint 由 `dimANSI()` 照常淡化（F8），不用另外處理。

### 2b. key reference

鍵欄（`helpRow.key`）：

| 位置 | 現在 | 改成 |
|---|---|---|
| `keyref.go` `panelKeyRef()` | `1 2 3`、`j k`、`g G`、`u d`、`Ctrl+C` | `1–3`、`j/k`、`gg/G`（`gg` 見第 6 條）、`u/d`、`Ctrl-C` |
| `keyref.go` `menuKeyRef()` | `j k`、`g G` | `j/k`、`g/G` |
| `keyref.go` `quitKeyRef()` | `j k`、`Ctrl+C` | `j/k`、`Ctrl-C`（`1–N` 已經是 en dash） |
| `keyref.go` `confirmKeyRef()` | `Enter y`、`Esc n` | `Enter/y`、`Esc/n` |
| `keyref.go` `breadcrumbKeyRef()`、`metaKeyRef()` | `j k`、`g G` | `j/k`、`g/G` |
| `keyref.go` `finderKeyRef()` | `j k`、`u d`、`g G` | `j/k`、`u/d`、`g/G` |
| `selectkeys.go` `selectKeys`（選取模式的 `?` 與 viewport key reference 的移動列都從這裡來） | `h ←`、`l →`、`j ↓`、`k ↑`、`v Esc` | `h/←`、`l/→`、`j/↓`、`k/↑`、`v/Esc` |

`Tab`、`Enter`、`Esc`、`Space`、`?`、`q`、`0`、`$`、`gg`、`G`、`y`、`v` 與 menu 列帶過來的鍵（`o`、`go`、`/`、`.`、數字）已經對。

顏色：`helpPopup.renderFull()` 的鍵用這一層的層色、粗體（`keyStyle`），說明用 `#7f849c`（Overlay1）—— 改成鍵 Blue `#89b4fa`、
說明 Text `#cdd6f4`。區塊標題（`item operation`、`panel operation`、`keys`）條文沒指定，維持現在的暗字即可。第 3 條變暗的列兩欄
都用 `disabledColor`。

### 2c. 句子

| 位置 | 現在 | 改成 |
|---|---|---|
| `app.go` `Update()`，選取模式按 `Tab` 的 toast | `Esc leaves the selection first` | `[Esc] leaves the selection first` |
| `openin.go` `showInTabs()` 的 toast | `All 5 tabs are in use — close one (w) to open …` | `… close one [w] to open …` |
| `places.go` Favorites 分頁的空狀態 | `(no favorites — press f on a directory)` | `(no favorites — press [f] on a directory)` |
| `goto.go` `setGotoPinnedItems()`，沒有最愛時的那一列 | `Nothing favorited — press f on a directory to favorite it` | `… press [f] on a directory …` |
| `splash.go` 最下面那行 | `Press Esc to close` | `Press [Esc] to close` |
| `config.go` 寫出的設定檔範本註解 | `press O on a file or directory; plain o just opens …` | `press [O] …; plain [o] just opens …`（README 兩份的 yaml 區塊是同一段，一起改；不強制） |

menu 列與 key reference 的**說明欄**裡提到的鍵（`next tab (h/l)`、`same as q, even while typing` 等）見「待確認」。

### 2d. label

- `bracketHotkey()` 產生的 menu 列、panel 標題（`[1]`、`[2] Preview`、`[3]` 的 tab bar）已經是括號標記，不用動。
- `goto.go` `setGotoPinnedItems()` 把鍵寫在 popup 標題裡：`Favorites · f unfavorite` —— 既不是 label 的括號標記，也不是 hint 的
  `鍵:說明`。建議標題只留 `Favorites`，`f:unfavorite` 放進這個 popup 的下框（`j/k:move Enter:run f:unfavorite Esc:close`；
  `spaceMenu` 要能多帶幾組 hint）；它已經在這個 popup 的 `?` 裡（`keyRef()` 的 `extra`），M3 要求的兩處就齊了。

### 2e. README 與 docs

| 位置 | 現在 | 改成 |
|---|---|---|
| `README.md`「Open, edit, and everything else」 | `Alt+Esc` 兩處 | `Alt-Esc`（內容照第 1 條改寫） |
| `README.md`「Key reference」的 code block 第一列 | `j k        u d         gg G        h l (switch this panel's tab)` | `j/k`、`u/d`、`gg/G`、`h/l` |
| `README.md`「Key reference」的表 | `Ctrl+C` | `Ctrl-C` |
| `README-zh_TW.md`「開啟、編輯,以及其他」 | `Alt+Esc` 兩處 | `Alt-Esc`（同上） |
| `README-zh_TW.md`「按鍵一覽」的 code block 第一列 | `j k        u d         gg G        h l(切本面板分頁)` | `j/k`、`u/d`、`gg/G`、`h/l` |
| `README-zh_TW.md`「按鍵一覽」的表 | `Ctrl+C` | `Ctrl-C` |
| `docs/dev-remarks.md`「設計決定」PTY 那段 | `Alt+Esc` 兩處 | 第 1 條改寫時一起改 |
| 同檔「已經符合、不用修的（對照 v0.1.7）」K10 那條 | `Alt+Esc` | `Alt-Esc` |
| 同檔「user 裁定的」PTY 出口鍵那條 | `Alt+Esc` | `Alt-Esc`（裁定內容不改，新裁定見第 1 條） |
| 同檔「popup 共用框」那段 | 沒寫 hint 的寫法 | 補一句：hint 經同一個 helper，`鍵:說明`，鍵 Blue、冒號與說明 Overlay0 |
| `CHANGELOG.md` `[Unreleased]` | `Alt+Esc` 一處（Added）、`Ctrl+C` 三處（Changed） | `Alt-Esc`、`Ctrl-C`；已發布的版本段不動。另記一條 Changed：下框、footer、key reference 的新寫法與顏色 |

README 其他地方的鍵名已經對（`Tab`、`Enter`、`Space`、`Esc`、`↑` / `↓`、`1`–`3`、`gg` / `G`）；「Space menu」表裡「Open 加反引號的
`o`」那種排法是 README 自己的版面，條文只管鍵名。註解不在 M5 範圍：`pty_unix.go` `isExitKey()` 的註解、`pty_unix_test.go` 的註解
與錯誤訊息寫 `Alt+Esc`，第 1 條改寫這幾段時順手用 `-` 即可。

### 2f. 測試

斷言畫面字串的測試要跟著改：

- `confirm_test.go` `TestConfirmRender`（`Enter trash`、`Esc cancel`）
- `inputpopup_test.go` `TestInputPopupRender`（`Enter create`、`Esc cancel`）
- `spacemenu_test.go` `TestSpaceMenuRender`（`Enter run`、`Esc close`）
- `helppopup_test.go` `TestHelpPopupRender`（`? or Esc close`）、`TestHelpPanelDigitsMatchPanels`（預期值用空白串起 `1 2 3`，改成
  `1–3`）
- `view_test.go` `TestListNavHintFocusGated`（`enter into`、`esc back`、`jkud move`、`hl switch tab`）、`TestMarksHint`（`p pick` 等
  四個，以及反向檢查的 `m mark`、`c copy`、`v move` 也要換成新寫法，否則永遠找不到、變成空檢查）
- `k11_test.go` `TestK11TabAnswersWhileSelecting`（`Esc leaves the selection`）、`TestK11HintFollowsStateWidthHolds`（`v select`、
  `? keys`、`Esc leave`）
- `splash_test.go` `TestSplashCreditRendersAboveHint`（`Press Esc to close`）
- `k6_test.go` `TestK6KeyRefScrolls`（`Ctrl+C`）
- `pty_unix_test.go` `TestPtyFrameShowsExitKey`（`Alt+Esc`）
- `f1_test.go` `TestF1FinderTypingEnterPicksTheHighlighted`（`Enter=go`、`Tab=list`）

（`f7_test.go` 找 footer 的 `menu`、`places_test.go` 找 `no favorites`，改完仍然找得到，不用動。）

另補一個 `m5_test.go`（kbu 有同名的前例）：

- 每一個 hint、footer 寫死完整的預期字串（上面兩張表的「改成」），不要只找子字串。
- 全部的 hint、footer、key reference 鍵欄、句子裡沒有 `Alt+`、`Ctrl+`、`Shift+`、` · `、`=`、小寫的 `enter` / `esc` / `space` / `tab`。
- 顏色（truecolor）：hint 與 footer 的鍵是 `#89b4fa`、冒號與說明是 `#6c7086`；key reference 的鍵 `#89b4fa`、說明 `#cdd6f4`。
- mutation：任一處改回舊寫法、`drawPopupBoxPad()` 改回 `tStyle`、key reference 的說明改回 `#7f849c`，各自要紅。


## 3. `?` 的 key reference 不把現在不能按的鍵變暗 —— M6

**現況**：

- **(a) menu 裡變暗的列，到了 key reference 看起來跟能按的一樣。** `keyref.go` `menuRows()` 把 menu 的列轉成 `helpRow`，但 `helpRow`
  （`helppopup.go`）沒有「現在不能按」的欄位，`menuItem.disabled` 在這裡丟掉；`helpPopup.renderFull()` 每一列都用同一種顏色。
  受影響的：
  - `[1]` panel 的 key reference 與 `[1]` Space menu 的 key reference：`l` Switch tab、`w` Close tab（只有一個分頁時）、`t` Tab
    （分頁滿 5 個時）—— `buildSpaceMenu()`。
  - Sort 的 key reference：`r` Reset（這個目錄沒有排序時）—— `setSortColumnItems()`。
  - Open in 的 key reference：`n` New tab（分頁滿了時）—— `openOpenInMenu()`。
- **(b) `[2]` Preview 的 `y` 與 `Enter` 有時不作用，menu 與 key reference 卻照常列。** `openDetailYank()` 在 `preview.body` 是空的
  時候直接 return：cursor 在空目錄上（`treeLines()` 沒有東西）、檔案讀不到（`(unreadable)`）、`[1]` 本身是空目錄（`(no selection)`）。
  `buildSpaceMenu()` 的 `panelDetail` 分支 `Yank` 列照常列出、不變暗；`panelKeyRef()` 的 `y` 與 `Enter`（`enterDesc()`）也一樣。
  menu 這半邊 v0.1.14 之前就不符合 M6，這輪逐 surface 核對 key reference 時看到。

**規則**：M6 —— 對象存在、但現在不能執行：列照樣出現、**變暗**，說明不另寫原因；對象不存在就不列。v0.1.14 起 **`?` 的 key
reference 照同一套**。下框 hint 與 footer 只列現在按得了的鍵也可以，由 app 決定。v0.1.16：key reference 裡另外加標題、說明**別的
surface** 的一段不算這個 surface 的鍵，照亮顯示（filu 沒有這種段落，見「已經符合」）。

**怎麼改**：

- `helpRow` 加 `disabled`；`menuRows()` 把 `it.disabled` 帶過去；`helpPopup.renderFull()` 對 disabled 列的鍵與說明都畫
  `disabledColor`（跟 `spaceMenu.renderFull()` 變暗的列同一個顏色；平常的鍵與說明顏色見第 2 條）。key reference 沒有 cursor，不必
  處理 cursor bar。區塊標題不變暗。
- `[2]`：`Yank` 列在 `preview.body` 空時 `disabled: true`（熱鍵本來就不作用）；`panelKeyRef()` 的 `Enter` 列在 `[2]` 同樣條件下
  變暗（`Enter` 是 core key，照 K6 一定要列）。三種空的情況都算「現在不能」：panel `[2]` 一直在，cursor 換一個項目就有內容。
- 對象不存在的已經不列，不用動：`[1]` 空目錄的 item operation、沒有 mark 時的 Copy / Move here、Marks / Tasks / Favorites 空的時候
  的 item operation 與 Zip / Clear、sort 方向的 `Unset`、Goto → Favorites 沒有最愛時的 `f`；`Favorite`（`f`）對檔案永遠不能，照前例
  不列。
- 下框 hint（app 決定，不強制）：`[1]` 的 `listNavHint()` 在只有一個分頁時照列 `h/l:switch tab`（`h` / `l` 不作用）；Favorites
  分頁的 `favoritesHint()` 在沒有最愛時照列 `o:open in`、`D:remove`（兩個都不作用）；Marks 分頁的 `marksHint()` 則是 bucket 空時整條
  留白。條文給的兩種做法（照 key reference 變暗，或只列按得了的）現在這兩處都不是。建議跟 `marksHint()` 一致，只列按得了的
  （`favoritesHint(hasItems)`、單一分頁時拿掉 `h/l`）。footer 的四個鍵永遠按得了，不用動。
- 文件：dev-remarks「`?` key reference」那段補一句「menu 裡變暗的列，key reference 也變暗」；README 兩份 `?` 那一列可以順手寫
  「現在不能按的鍵變暗」（不強制）。
- 測試（新增 `m6_test.go`，或併進 `k6_test.go`）：開 truecolor（`lipgloss.SetColorProfile(termenv.TrueColor)`，`t.Cleanup` 還原）。
  一個分頁時 `[1]` key reference 的 `l`、`w` 列是 `disabledColor`、`t` 不是；五個分頁時 `t` 是；Sort 沒有排序時 `r`；Open in 分頁
  滿時 `n`；`[2]` 沒有內容時 Space menu 的 `Yank` 是 disabled，key reference 的 `y` 與 `Enter` 變暗、有內容時不變暗；區塊標題不變暗。
  mutation：拿掉 `menuRows()` 帶 `disabled` 那一行、拿掉 `renderFull()` 的變暗、拿掉 `[2]` 的條件，各自要紅。


## 4. zoom 還寫在「偏離 tdp」 —— 術語「模式」（已定案）

**現況**：`docs/dev-remarks.md`「偏離 tdp」只有一條：「Zoom 不是 `Esc` 會退出的模式（K4）」。程式碼的行為：`z` 展開、再按一次
還原（`toggleZoom()`），focus 移到別的 panel 也會還原（`setFocus()`），zoom 中 `Esc` 仍是回上一層目錄。

**規則**：v0.1.14 的術語「模式」—— 版面的切換（例：zoom 把一個 panel 放大到全畫面）**不是模式**：沒有鍵換意思，`Esc` 不必退出
它，由它自己的鍵還原。

**已定案**（user，2026-09-29）：照 v0.1.14 這已經不是偏離，移到「設計決定」。

**怎麼改**（只動文件，程式碼不用改）：

- 把那一條從「偏離 tdp」移到「設計決定」，改寫成依據術語：zoom 是版面的切換、不是模式（tdp v0.1.14 起的術語「模式」），由 `z`
  還原；`Esc` 維持「回上一層目錄」，理由照舊（在同一個畫面裡兼兩種意義，P4）。
- 「偏離 tdp」剩下空的：寫「目前沒有」（標題留著，D7 的固定章節）。
- 「已知的牆與未做」的 tdp 那一行：「除了下一節的偏離都符合」改掉並補上 v0.1.14–v0.1.16；「列進 `docs/filu-terminu-fix.md`（目前
  沒有這個檔）」在清單刪掉時再確認一次。
- terminu `.local/family-fix/filu/README.md` 的「偏離 tdp」段一併更新（見「先看」）。


## 5. dev-remarks 與註解說 toast「不握鍵盤」 —— F1、F8（文件對齊）

**現況**：行為已經符合（見「已經符合」），用語停在 v0.1.13：`docs/dev-remarks.md`「最上層以外全部 dim」那段寫「toast 不握鍵盤、
不算一層」；`view.go` `popupLayers()` 的註解寫 `The toast is not a layer (it holds no keys, tdp F8)`。

**規則**：v0.1.14 F1 —— toast 除了 `Esc`，按鍵都穿過它；F8 —— toast 不觸發 dim：除了 `Esc` 它不收鍵，不是一層。

**怎麼改**：兩處改成「除了 `Esc` 不收鍵」。只是用語，行為不變，不記 CHANGELOG。


## 6. key reference 跟實際按鍵不符的兩處 —— K6、M4（不是這一輪的改動）

核對第 2、3 條的 key reference 時看到的，v0.1.13 以前就這樣：

- **`g` 其實是 `gg`。** `panelKeyRef()` 寫 `g G`（top / bottom），但 panel 上單按 `g` 只是等下一個鍵（`Update` 的 `pendingG`），要
  `gg` 才到頂（README 寫的就是 `gg`）。改成 `gg/G`（寫法見第 2 條）。menu、finder、breadcrumb、metadata 框的 `g` 是單鍵，寫 `g/G`
  沒錯。
- **`Shift-Tab` 哪裡都沒寫。** panel 上 `Shift-Tab` 反向換 focus（`Update` 的 `case "shift+tab"`），但 key reference、Space menu、
  README 都沒有它，是只能靠事先知道的鍵（M4：panel 的 key reference 列出這個 panel 能按的鍵）。K2 說反向切換是熱鍵、做不做由 app
  決定；建議留著，在 `panelKeyRef()` 的 `Tab` 下面補一列 `Shift-Tab`，README 兩份 `Tab` 那一列可以順手提。

測試：`[1]` 的 key reference 有 `gg/G`、沒有單獨的 `g`、有 `Shift-Tab`。


## 已經符合、不用修的（對照 v0.1.13 → v0.1.16 的改動）

- **F1 toast 除了 `Esc` 不收鍵**：`Update` 的 toast 分支只認 `esc`（`m.toast.owns() && msg.String() == "esc"`），其他鍵照常往下路由
  （`TestF3ToastLetsOtherKeysThrough`：toast 開著時 `q` 照樣開 quit picker）。
- **F8 toast 不觸發 dim**：toast 不在 `popupLayers()` / `stackOrder()` 裡，`View` 在 dim 與合成之後才畫它。
- **K11 模式裡回應 `Tab` 的 toast，第一個 `Esc` 先收它**：選取模式的 `Tab` 開 toast（`TestK11TabAnswersWhileSelecting`）；toast 的
  `Esc` 分支在 `detailYank` 路由之前，所以第一個 `Esc` 收 toast、選取還在，第二個才離開選取。沒有測試直接釘住這個順序，可以順手
  補一個（mutation：把 toast 分支移到 `detailYank` 之後）。
- **K10 家族的出口鍵 `Alt-Esc`**：`isExitKey()` 就是 `Esc` + Alt；`ptyExitHint` 在 PTY 開著時常駐下框（`TestPtyFrameShowsExitKey`）。
- **D5 其他 Alt 組合的出口鍵**：filu 的 PTY 只有 `Alt-Esc` 一個 app 鍵，不適用。
- **PTY 開著時的 toast**：背景任務完成時寫 `state.yaml` 失敗之類的 toast 可能疊在 PTY 上；那時 `Esc` 給 shell（PTY 分支在 toast
  分支之前），toast 等時間到自己收。terminal 類「按鍵都給子程序，只有出口鍵屬於 app」（F1、K10）優先，這是對的。第 1 條調路由時
  保持這個順序：只有 PTY 上疊了框時，toast 的 `Esc` 才先收。
- **術語「模式」**：zoom 的行為本來就是條文說的樣子（第 4 條只動文件）；filu 唯一的模式是 yank viewport 的選取，不受影響。
- **M5 的 label**：`[]` 只出現在 label —— menu 的列（`bracketHotkey()`）與 panel 標題；標記方式 v0.1.15 沒變。key reference 說明裡的
  `[1]`（`enterDesc()`）是 panel 的名字，不是熱鍵標記。
- **M5 已經對的部分**：key reference 的 `Tab`、`Enter`、`Esc`、`Space`、`?`、`q`；quit picker 的 `1–N`（en dash）；panel 下框與 footer
  的鍵 Blue、說明 Overlay0（`keyLegendGap()`）；README 的 `Tab`、`Enter`、`Space`、`Esc`、`↑` / `↓`、`1`–`3`。
- **M6 menu 的部分**：`[1]` 的 Switch tab / Tab / Close tab、Open in 的 New tab、Sort 的 Reset 在 menu 裡已經變暗、cursor 可停、
  `Enter` 與熱鍵不作用（`spaceMenu.update()`）。第 3 條只補 key reference 與 `[2]` 的 Yank。
- **M6 描述別的 surface 的段落（v0.1.16）**：filu 的 key reference 沒有這種段落 —— 區塊標題 `item operation`、`panel operation`、
  `keys` 都是這個 surface 自己的鍵；viewport 的 `v` 那一列只在說明裡提到選取模式另有 `?`，不是一段。沒有東西要照亮。


## 待確認

沒有。上一題（menu 與 key reference 說明欄裡的鍵算不算句子）user 2026-09-29 裁定**算**，v0.1.17 寫進 M5：`next tab [h]/[l]`、
`Marks / Tasks / Favorites [h]/[l]`、`same as [q], even while typing`、`open the scrollable view (same as [y])`、
`start selecting (then [?] lists its keys)`，併進第 2 條（2c 句子）一起改。

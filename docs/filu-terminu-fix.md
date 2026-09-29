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

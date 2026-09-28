# filu — terminu fix

filu 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.12/principle)（tdp v0.1.12）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與 `docs/dev-remarks.md` 裡描述該行為的段落。有意不修的，改寫成
`dev-remarks.md`「偏離 tdp」的一條並附理由。

盤點日期：2026-09-28（對照 tdp v0.1.11）。以 `main` 的 `f0bb6b9` 為準，逐條拿程式碼核對過；位置寫檔案與函式，不寫行號。

v0.1.11 動到的是 K3、F6、F7、F8 與 D2。filu 大致已經符合：F8 / D2 的 dim 本來就是 tdp 的參考實作，K3 的內容區 panel 與
F6 的 picker 都是 filu 先做、tdp 再寫進條文的。**只有一條要改**：finder 載入中沒有輪轉的 loading icon（F7）。


## 先看

- **dim 不用動。** tdp D2 的參考實作就是 filu 的 `internal/ui/dim.go`，下方「已經符合」逐項對過 D2。其他 app 照抄的做法是
  「一個共用 helper、在合成最上層之前過一次整張畫面」，不是每個 popup 各自畫暗的版本 —— filu 已經是這樣，改其他東西時別拆掉。
- **驗證要把 `View()` 印出來看。** 條文讀再多遍也看不出畫面長怎樣；loading icon 要在 finder 開著、`fd` 還沒送完時印一次
  `View()`，確認 icon 在標題後面、框的寬高沒變、每一列仍剛好是終端機寬（L4），`fd` 送完再印一次確認 icon 消失。
  動到 dim 相關的畫面時，也印一次開著 popup 的 `View()`，確認 `[1]` 的 powerline 膠囊背景還在（變暗、沒消失）。
- **每修一處補一個 model test，逐處 mutation**：把修正單獨改回舊行為，確認對應的測試會紅。預期值寫死成條文算出來的東西，
  不要用被測的函式量自己（filu 第二輪三次同源自比的教訓）。
- **守舊行為的測試要改寫，不是刪掉**：改名並反轉斷言，名稱寫出新規則。
- **描述 loading 的文件一起改**：`dev-remarks.md`「運作方式」的 Finder 段、「對照 tdp 時確認過的」的「F7 finder 的高度」；
  README 若提到 finder 的載入狀態也要同步。
- **不發版。** 家族全部 app 與 tdp 都穩定之前不發 release，也不動 tdp 以外的版號；commit 照常記在 CHANGELOG `[Unreleased]`。


## 1. finder 載入中沒有輪轉的 loading icon —— F7

**現況**（`internal/ui/search.go`）：

- Search / Find / Goto 三種 finder 都是 `searchModel`。`open()` 與 `rescan()`（`/` 錨定到絕對路徑時重掃）把 `loading` 設成
  `true`，`fd` 的串流由 `streamFilesCmd` 在背景送 `fileBatchMsg`，`onStreamBatch()` 收到 `done` 才把 `loading` 設回 `false`。
  依內容搜尋時，每次打字經 `grepDebounce` 後跑 `rg`，期間 `searching` 為 `true`。
- 載入的揭露只有清單裡的靜態暗字：`listColumn()` 在 `loading` 時整個清單只畫一列 ` (indexing…)`，`searching` 且沒有結果時畫
  ` (searching…)`；輸入列右側的筆數會跟著長。
- 標題固定是 `renderFull()` 裡的 ` Search` / ` Find` / ` Goto`，**標題後面沒有任何 loading icon**，也沒有東西在轉。
- 高度：`geometry()` 固定取畫面高的 90%，開框後不伸縮（不在 loading 期間變高，這點沒問題）。

**規則**：F7 —— 打開時內容還不確定（串流、載入中）的 popup，loading 要揭露：**popup 標題後面放一個輪轉的 loading icon**
（webu 載入網址時的那個 icon）；loading 結束，icon 消失。finder 開框時結果還在串流，正是這種 popup。

**怎麼改**：

- `loading || searching` 時，finder 左框（清單框）的標題後面接一個輪轉 icon；兩者都結束時 icon 消失。`rescan()` 重新進入
  `loading` 時 icon 再出現。右邊預覽框的標題是檔名，不放 icon。
- icon 用 webu 的那一組（`webu/internal/ui/theme.go` 的 `spinnerFrames`：Material Design 圓形切片 `U+F0A9E`–`U+F0AA5`，
  一格方形、兩欄寬、原地轉）。filu Tasks 用的 braille 點是一欄寬，webu 試過會讓旁邊的字跳一格；code point 用裝好的 Nerd Font
  cmap 核對，不憑記憶。
- 動畫要有 tick 驅動：現有的 `spinnerTickMsg`（`app.go`）只在 `anyRunning()` 時續跑，要讓它在 finder loading 時也續跑，或像
  webu 一樣照時鐘取 frame、另給 finder 一個 tick。tick 停止的條件要跟 `loading || searching` 同一個來源。
- 標題變長不會撐寬框（`drawPopupBoxPad` 以 `innerW` 畫、上框的橫線跟著縮），但要測一次 icon 出現與消失時框寬、每列寬度都不變（L2、L4）。
- 清單裡的 ` (indexing…)` / ` (searching…)` 要不要留，由 filu 決定（見「待確認」第 1 題：載入中要不要直接顯示已串流到的結果）。
- 測試：開 finder、送一批未 `done` 的 `fileBatchMsg`，`View()` 的 finder 標題後面有 icon 的某一格；送 `done` 後沒有；`rg` 進行中
  （`searching`）也有；`rescan()` 之後再出現。mutation：拿掉標題的 icon、讓 icon 常駐、讓 tick 只看 tasks，各自要紅。


## 已經符合、不用修的（對照 v0.1.11）

- **F8、D2 dim**（`internal/ui/dim.go`、`view.go` 的 `View()`）：這就是 D2 寫的參考實作，逐項對過：
  - `dimKeep = 0.45`、`dimBase = #1e1e2e`，`dimRGB()` 算 `c × 0.45 + base × 0.55`（四捨五入）。
  - `dimSGR()` 前景（`38;2`、`38;5`、`30–37`、`90–97`）與背景（`48;2`、`48;5`、`40–47`、`100–107`）**都**淡化；16 色用 xterm
    調色盤、256 色用 `xterm256()` 先換成 RGB 再算。
  - 沒有指定前景的文字給 `dimText = dim(#cdd6f4)`：每一列開頭、每個 reset（`0` 與空參數）與 `39` 之後都補上。
  - bold、reverse、游標移動、非 SGR 的 CSI、文字本身不動；不剝色、不丟背景、不把前景統一成一個 dim 色。
  - 每一層都經過它：`View()` 依 `popupLayers()` 由下往上合成，輪到最上層（最後一個 `isActive()`）之前，把已合成的整張畫面
    （base panel、底下所有 popup、PTY）過**一次** `dimANSI()`，再疊上最上層。底下 popup 的邊框因此是自己層色（`assignLayers()`）
    的 dim 版本。toast 畫在 dim 之後、不觸發 dim。
  - 盤點時把 `View()` 印出來核對過：開 Space menu 後，`[1]` 膠囊的背景 `#89b4fa` 變成 `rgb(78,97,138)`、分頁列的 crust 背景
    `#11111b` 變成 `rgb(24,24,37)`，形狀與版面不變；Space menu 框外所有帶背景的格子，背景都等於原色的 dim 值，沒有一格被丟掉。
    現有 `f8_test.go` 守前景與下層邊框層色；背景目前沒有測試守著（修第 1 條時可以順手補一個）。
- **F7 其他 popup 的 loading**：只有 finder 的內容是開框後才來的。Space menu、global operation popup、sort / Goto / Open with /
  Open in / Search chooser / quit 各 picker、confirm、input、breadcrumb、metadata（`fileFacts()` 開框時同步 `stat`）、`[2]` 的
  yank viewport（開框時拿現成的 `preview.body`）、key reference，開框時內容都已確定。
- **F7 Tasks 的進度、`[2]` preview 的載入**：都不是 popup。copy / move / zip 的進度畫在 `[3]` Tasks 分頁，執行中那一列已有輪轉的
  braille spinner（`tasks.go` 的 `spinnerFrames`）；`[2]` 與 finder 預覽的 `loadPreview()` 是同步讀取（有 `previewCap` 上限），沒有
  loading 狀態。
- **F7 開著時列數會變的 popup**：
  - finder：打字篩選與 `fd` / `rg` 串流改變的是清單內容，框高固定（`geometry()`），列數變化在框內捲動。原因是 loading 與使用者
    打字，兩者都在允許範圍。
  - Goto → Favorites（`gotoFavMenu`）裡按 `f` 取消收藏：少一列，原因是使用者自己的操作（允許）；filu 的做法是框高不變、補空白列
    （`spaceMenu.openRows`），見「待確認」第 2 題。
  - sort 的 `Reset`：一開始就在、沒排序時變暗（M6），開著時不長出列。
  - 沒有被背景事件改列數的 popup：Space menu 與各 picker 的列在開框時建好（`setItems()` 只在開框處呼叫），Tasks 完成、目錄
    watch 觸發都不會改到開著的 popup。
- **K3 內容區 panel**：`[2]` preview 是沒有項目的內容區，`Enter`（跟 `y` 同一個分支，`app.go` 的 `[2]` 按鍵處理）開可捲動的 yank
  viewport（`openDetailYank()`）；preview 沒有內容（`body` 為空）時不作用。`[3]` 的 Tasks 不是內容區：它有 cursor（`taskCursor`），
  `Enter` 對那一筆任務做事（把當前分頁帶到它的目的地），屬於 K3 的「focus 項目」；Marks、Favorites 同樣是有 cursor 的清單。
- **F6 picker 算確認**：`o` 先 confirm（`confirmOpen`）；`O` 的 Open with picker 選任何一列（含 `Default`）直接執行
  （`runOpenWith()`），不另跳 confirm。這正是 v0.1.11 F6 舉的例子，裁定與理由已寫在 `dev-remarks.md`「設計決定」。
- **F1 多步驟一步一 popup**（v0.1.9，再核一次）：sort 欄位 → 方向（`sortDirMenu`）、Goto → Favorites（`gotoFavMenu`）、
  Search chooser → finder、`t` → Goto 流程，每一步都是疊在上一步上的 popup（F4）；其他流程（Delete、Unfavorite、Clear marks 的
  confirm，Rename、Add、Zip 的 input）只有一步。
- **其餘條目**：v0.1.11 沒有改動，第二輪已對 v0.1.10 全文核對（見 `dev-remarks.md`「對照 tdp 時確認過的」），這次重讀全文沒有
  找到新的違反。


## tdp v0.1.12 定案（2026-09-28，回答四個 app 在 v0.1.11 盤點時的共同問題）

修的時候以這裡為準；本檔的條目與「待確認」照下面改讀。

- **F7 loading icon 一定要放**：popup 在 loading 時，標題後面一定放輪轉的 loading icon，**跟高度會不會變無關**。
  loading 指**整個 popup** 的內容還沒到；若只是**某一個項目**本身是持續進來的資料流，loading 的是那個項目，怎麼揭露由 app 決定。
- **F7 使用者操作造成的高度變化是「允許」不是「要求」**：app 可以維持原高；原則是揭露的資訊要正確。
- **D2 淡化絕不讓顏色變亮**：每個通道取原值與淡化值較小的那個；比 base 還暗的顏色（例：`#000000`）維持原色。
- **D2 / D6 truecolor**：家族要求 truecolor terminal，dim 一律輸出 24-bit，不必照色彩深度降階；README 的需求段跟 Nerd Font
  並列寫上「需要 truecolor terminal」（D6，家族預設）。
- **D3 loading icon 規格**：Nerd Font `nf-md-circle_slice_1`–`_8`（U+F0A9E–U+F0AA5）八格；一格 90ms；由時鐘決定哪一格
  （`frames[(now / 90ms) % 8]`），tick 只在有東西 loading 時續排；寬一格；顏色跟旁邊的字，放在 popup 標題後面時用該層層色（bold）。

- **本檔**：
  - 待確認第 2 題（Favorites 取消收藏後維持原高、補空白）照 F7 定案：「允許」不是「要求」，現況符合，不用改。
  - 第 1 條的 icon 照 D3 規格（webu 的圓形切片八格、90ms、由時鐘決定）。
  - 新增：README 兩份的需求段補「需要 truecolor terminal」。
  - 待確認第 1 題（finder 載入中要不要邊串流邊顯示，程式碼與 dev-remarks 對不上）仍是 filu 自己的決定。
  - **新增:`[3]` Tasks 執行中的轉圈也換成 D3 icon**。上面「已經符合」把它算成「單一項目的資料流、由 app 決定」,
    user 裁定(2026-09-28)仍改成跟 finder 同一個 D3 icon(circle slice、90ms、時鐘驅動),全 app 只有一種轉圈;
    braille 的 `spinnerFrames` 與 100ms 的 `spinnerTick` 退場。顏色跟旁邊的字同色。

## 已裁定

1. **finder 載入中邊串流邊顯示**(user 2026-09-28):`loading` 時清單直接列出已收到的結果，可以濾、可以選;
   是否還在載入交給標題後的 loading icon(第 1 條)揭露，跟 `dev-remarks.md` Finder 段「首批近乎立即、載入中就能濾」一致。
   一筆都還沒收到時才顯示 ` (indexing…)`。併進第 1 條一起修。

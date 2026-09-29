# filu 開發者備忘

開發 filu 時要提醒自己、以及與 AI 協作時記下的決策。filu 遵循
[terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.23/principle)（tdp）；
使用者要知道的在 README,這裡記的是「它為什麼長這樣、內部怎麼做」。

---

## 運作方式

### cd-on-quit 為什麼需要一行 shell 設定

這是 OS 限制、不是偷懶:一個 process 只能改**自己**的工作目錄,沒有任何 syscall 能改
**父** process(你的 shell)的 cwd。filu 是 shell 的子程序,自己 `cd` 影響不到 shell;
唯一能原生改 shell cwd 的是 shell **內建指令**,而 filu 是外部 binary。

所以採兩段式 handshake:`filu shell` 印出一個 shell function,它透過
`FILU__LAST_DIR_FILE` 給 filu 一個暫存檔;filu 離開時把選定目錄寫進去,function 再讀檔
`cd`。這也是為什麼要用 `filu` 而不是 `./filu` 啟動 —— wrapper 攔截的是指令名 `filu`,
帶路徑的呼叫會繞過它。

### 各功能的實作備註

- **3 面板比例** — 上排 list 與 preview 以 2:1 共用(資訊豐富的 list 值得更寬);
  `[3]` 在下方橫跨滿寬。寬度 `< 72` 時放棄 grid、只畫 list(`view.go normalMiddle`),
  `z` Zoom 是任何面板的逃生艙。頂部沒有任何 content row,`midH = height − 1`,只扣 footer。
- **檔案清單欄位** — 狀態 glyph、`Modified`、`Owner`(user:group)、`Perms`(eza 配色
  `r` 黃 / `w` 紅 / `x` 綠)、`Size`(eza color-scale,越大越暖;目錄顯示 `-`,絕不遞迴
  加總)、icon + 檔名。欄位標題兼排序指示;面板變窄時依 owner → size → modified →
  perms 的順序收合,檔名永遠最後才收。欄位在 `listModel.reload()` 一次算好,render
  每 frame 零 I/O。
- **每目錄排序** — `S` 可疊多層 chain,每個目錄各自記住,存進 `state.yaml`。
- **麵包屑** — lavender 純文字放在 `[1]` 第一列,下方一條低調分隔線。過長時前段縮成
  字首(`~/Documents/x` → `~/D/x`),再不夠中間縮 `…`,永不折行。
- **隨寬逐階縮字只有一份** — `header.go fitPathSegments` 是泛型的四階縮法(全名 →
  前段逐段縮成首字元 → 中間 `…` → 末段硬截),breadcrumb 量渲染寬、Goto picker 的
  Favorites 清單量 plain 字串寬,共用同一支。**別再各寫一份**,縮法會漂移。
- **Marks bucket** — 複製會保留 bucket(可連續落地到多個目錄);搬移會更新 bucket 內
  路徑讓它保持有效。`p` pick 出來的子集優先於整個 bucket。
- **Zip** — 打包「Copy 會落地的那些」;檔名預設:單項用自己的名字、同目錄多項用該
  目錄名、跨目錄用時間戳。壓縮檔寫進暫存目錄(不動任何工作目錄),並成為 bucket 裡
  **唯一**的 pick,之後走既有的 `c` / `v` 落地路徑。每個 pick 以自己的名字放進壓縮檔
  (目錄保留內部結構),同名的 pick 兩個都留。輸入值過 `zipFileName`(取 basename +
  補 `.zip`),打 `../../etc/passwd` 也只會變成 `passwd.zip`。
- **Tasks** — 同磁碟搬移是瞬間 `rename`;跨磁碟 / 複製才顯示進度。log 是帶時間戳的
  人話(`2026-07-28 14:32:07  Copied report.pdf → proj`)。中斷的任務存進
  `state.yaml`、下次啟動還原成 pending。`action` 是自由字串,加一種新任務不必動白名單。
  執行中那一列轉的是家族的 loading icon(D3,見下一條)。
- **Finder** — filu 自畫的分割 picker(清單 + 預覽),不是 fzf binary(fzf-in-PTY 試過
  失敗)。每種模式都串流列檔:用 `fd` 的走訪序、不排序,首批近乎立即、載入中就能濾;
  沒有 `fd` 時退回純 Go walk。以 `/` 或 `~/` 開頭的 query 會錨定到該絕對路徑,對整條
  路徑 fuzzy、深度限錨點下數層。Goto 一律掃隱藏目錄,噪音靠 `ignore_dirs` 黑名單擋。
  走訪還沒結束或 `rg` 還在跑時(`isLoading()`),清單照樣列出已收到的結果，標題後面轉
  loading icon(tdp F7);一筆都還沒到才寫 `(indexing…)`。icon 是 tdp D3 的規格
  (`loading.go`:circle slice 八格、90ms 一格、由時鐘 `loadingNow()` 決定哪一格),跟 Tasks
  共用一個 `loadingTickMsg`,只在 `anyLoading()` 時續排。popup 的上下框用 `dispWidth()` 量
  標題與 hint,在 CJK icon 字型上 icon 佔兩格時框線跟著縮。
  focus 在哪一邊看得出來(tdp F1、D3):打字時篩選列亮(Peach 的 glyph、游標 `█`),清單的
  cursor 列是淡的反白(`handColor`);`Tab` 進清單後，篩選列整列單一灰色(Overlay0,不是
  `dimANSI()` 的淡化)、不畫游標，cursor 列換成這一層的層色底、深色粗體字，跟 menu 的 cursor
  列一樣。
- **Preview** — 讀 magic bytes 判型別:目錄 → 內層 tree、壓縮包 → 內容清單、圖片 →
  base64 `data:` URI、SVG → 高亮 XML、文字 → Chroma(catppuccin-mocha)高亮 + 行號、
  二進位 → hex + ASCII、PDF → 抽出的文字 + 頁數。
- **Preview yank viewport** — `[2]` 的 `y` 開一個覆蓋 preview 的 viewport(`detailyank.go`):
  vim cursor + `v` 字元級選取,行號 gutter 不進剪貼簿;preview 為了塞進面板寬折斷的
  續行(`previewModel.cont`)複製時**不補**換行,否則貼出來的 base64 / 長 URL 會斷掉。
  移動照 vim:`h/j/k/l`、`0/$`、`gg/G`、`u/d`,以及以字為單位的 `w/b/e`(`wordmotion.go`):
  字分三類(空白、字母數字與 `_`、其他標點),跨行時行尾算空白、空行自成一個字(`w`、`b` 停在
  它上面),但折斷的續行不算行尾 —— 被面板寬切成兩段的字仍是一個字。選取內外都能用。
  選取是 tdp K11 的「模式」:鍵表只有一張(`selectKeys`),選取中 `?` 的 key reference 與
  viewport 本身 key reference 的移動列都由它產生。模式裡沒有 Space menu、也沒有可執行的
  按鍵清單，模式的鍵直接按;選取中 `Space` 不作用(tdp v0.1.10 拿掉了 v0.1.2 起那份只用方向鍵
  移動的按鍵清單 `modeList`)。選取中 `Tab` 暫停但回一個 toast;選取外 `Space` 也不作用
  (K5)。下框 hint 分選取內外兩種，框寬跟著螢幕、不跟 hint 變(L2)。選取中框標示自己是模式
  (tdp K11、D2、D3):外框與標題換成模式色 Yellow(`modeColor`),上框右側的模式名夾在兩個
  接頭之間、像框上嵌了一個標籤:`╭─ notes.txt ────┤Selection├─╮`(popup 是單線框，接頭用
  `┤` `├`,跟框同色不加粗，名字加粗;`selectModeName`,`?` 的標題 `Selection keys` 用同一個);
  框太窄時先截標題，模式名與接頭留著(`drawPopupBoxMode()`)。離開選取就回到層色。選取的反白也
  是 Yellow(`selectColor`,D2 的「選取」;以前是 Lavender,跟黃框不搭，Lavender 回到只當使用者
  足跡，P4)。游標與選取的位置是 rune 序號(複製用 rune 取);畫的時候交給 `ansi.Cut()` 之前先
  換算成那一段在字串裡的格數(`width.go` 的 `cellsBefore()`,跟 `ansi.Cut()` 同一套量法):中文字
  一個 rune 佔兩格，不換算就會畫在錯的字上、還把字重複畫出來。
- **Yank** — 走 OSC 52,所以能穿 tmux / SSH。
- **刪除** — 移到 OS 垃圾桶:macOS 用 `osascript`(不需 cgo)、Linux 寫 XDG
  `.trashinfo`。
- **分頁標記** — 第一個分頁掛啟動 glyph(它永遠開在啟動目錄,是 cd-on-quit picker 的
  固定參照),其餘各掛一隻動物(cat / dog / paw / egg)。路徑交給麵包屑列,分頁列只標
  位置與哪個 active。上限五個：到上限時 Space menu 的 `Tab` 列變暗、`t` 不作用(tdp M6),
  分頁列已畫出 5 個分頁，看得出原因。
- **icon 與配色** — 型別 glyph 取自 eza 完整 icon 表(~760 個);顏色來自烘進 binary 的
  `vivid generate catppuccin-mocha` `LS_COLORS` palette,依 eza 的優先序解析(目錄 →
  symlink → executable → 最長 suffix → 副檔名)。執行時不讀 `LS_COLORS`,每個安裝都是
  同一套配色。
- **即時刷新** — 清單分頁用 fsnotify 監看自己的目錄,外部變動時 reload 並保留游標;連續
  事件會 debounce。
- **session 持久化** — 多開的分頁(dir + cursor)、marks bucket、favorites、tasks、每
  目錄排序存進 `state.yaml`;第一個分頁永遠開在啟動目錄,啟動時永遠 focus 在清單。
- **config 與 state 分開** — `config.yaml` 是使用者手改的檔,`state.yaml` 每次離開自動
  重寫,兩者刻意分檔。`FILU__CONFIG` / `FILU__STATE` env var 各自指定一個**目錄**,
  `config.yaml` / `state.yaml` 照原檔名放在裡面(測試與 demo 錄製用它隔離，demo 兩個都指向
  `.local/demos/filu-home`)。filu 讀的變數一律照 tdp D6 的 `FILU__<名字>`(app 名後兩個底線):
  `__CONFIG`、`__STATE`、`__LAST_DIR_FILE`、`__ICON_WIDTH`、`__REPAINT`;2026-09-29 從單底線
  改名、不留舊名(user 裁定)。另讀家族共用的 `TERMINU__ICON_WIDTH`(見下一條)。
- **CJK Nerd Font 寬度** — 有些 CJK Nerd Font(如 Maple Mono NF CN)把 file-type icon
  畫成 2 格。filu 啟動時決定 icon 佔幾格(`DetectIconWidth()`),順序照 tdp D6:
  `FILU__ICON_WIDTH`(手動覆寫)→ `TERMINU__ICON_WIDTH`(家族 app 在自己的 PTY 裡設給子程序的;
  在別的 app 的 PTY 裡，CPR 是外層的終端模擬器回答，它把 icon 當一格，量不到真的)→ CPR 探測
  實際格寬，只收 1、2。反過來，`[s]hell` 的 PTY 開子程序時也把自己用的格數設成
  `TERMINU__ICON_WIDTH`(`ptyPopup.start()`,取代繼承來的同名值),巢狀幾層都傳得下去。**每一個量寬度的地方**
  都走 `width.go` 的顯示寬度層(tdp D6,filu 是參考實作):量寬 `dispWidth()`、截斷
  `dispClip()` / `truncate()` / `truncPathLeft()`、補齊 `padDisp()`、並排 `joinH()` /
  `joinV()`、置中 `centerDisp()`,疊 popup 用 `compositeDisp()`(`overlay.Composite` 的
  顯示寬度版:左段 `dispClip()`、右段 `dispCutLeft()`,被框邊切成兩半的 icon 補一格空白;框比
  畫面寬或高 —— 調整終端機大小的那一格還是舊尺寸 —— 起點取 0、切到畫面邊界，不 panic,tdp D6
  v0.1.21。以前 `clampSpan()` 會給出負的起點,`strings.Repeat` 拿到負數就 panic)。
  截斷從 w 格往回找，不假設 icon 只在行首(疊 popup 時切點右邊常有 icon)。splash 的
  像素在兩格 icon 下只畫 glyph、不再加空白。powerline caps `U+E0A0–E0D7` 刻意排除(它們
  單寬);不能靠終端的 East-Asian-Width 全域旋鈕解，那會連帶改動其他字元的寬度。
  `d6_test.go` 在 icon 佔 1、2 格下把每一種 popup 各開一次，量單獨的框與疊上去的整個
  畫面每一列(finder 量單一個框：`joinH()` 會把錯位補平);`width_test.go` 量 panel。
- **控制字元清洗** — 檔名可能含控制字元(macOS 的 `Icon\r`)。`safeName`(`list.go`)在
  顯示時剝掉控制字元(也擋 ANSI injection),檔案操作仍用真實名;套在所有 name-render 點。
- **Space menu 是熱鍵的殼** — `buildSpaceMenu` 依 focus 組 item / panel 兩區
  (`groupedMenu`),選一列就把那個熱鍵送給 focus 的 panel(`dispatchFocusKey`)。
  新增一個動作,要同步加進對應 focus 的 Space menu。
- **`gg` / `go` chord** — 單一 `AppModel.pendingG` 掛在主 switch 的 chokepoint(所有 popup
  return **之後**、只管主面板):`gg` 落既有 `case "g"`、`go` 呼 `handleListKey("go")`。
- **popup 共用框** — 全部走 `drawPopupBox`(title 嵌上框、hint 嵌下框、內容上下各一列
  padding);yank viewport 與 finder 用 `drawPopupBoxPad(pad=false)` 貼齊邊框。內容列
  (finder 的結果、input 的輸入列、Open in、quit picker、viewport 的目錄樹都有 icon)由
  `padDisp()` 補齊或裁到框寬，寬度一律走 `width.go`(見「CJK Nerd Font 寬度」)。
  hint 與 panel 下框、footer 同一個 helper(`keyLegendFit()`):`鍵:說明`、項目之間一個空格，
  鍵 Blue、冒號與說明 Overlay0(tdp M5、D2)。每個框用自己的寬度去 fit,放不下的項目從尾端
  整組不放、不截在中間(tdp D3、D1;finder 並排時的清單框在 96–116 欄會用到);`drawPopupBoxPad`
  原樣放上去，不再截、也不用層色重畫。panel 的下框收「鍵、說明」的組，由 `panelBoxHint()` fit;
  panel 沒有 focus 時(`[3]` 在 focus 停在 `[1]` 時照樣列 Marks / Favorites 的鍵)改用
  `legendFit()` 的暗配色：鍵 Overlay0、冒號與說明 Surface2 —— Blue 只給拿鍵的地方(tdp D2)。
  key reference 的鍵 Blue、說明 Text,區塊標題維持暗字。
- **popup 一律同寬** — 每個 popup 的外框都是 `min(terminal 寬 − 2, 120)`(`popupInnerWidth()`
  給扣掉左右邊框的內寬),不看內容:說明太長就截、訊息與值在框內折行(tdp F7,取代 D4 的
  「key reference 依最長說明算寬」)。finder 的兩個框加中間一欄間隔合起來是這個寬度。
  PTY 例外:外框貼滿畫面(F7 的 terminal 類)。
  `[2]` 的 viewport 高度依內容(`contentRows()`),上限是畫面扣上下留白，超過才捲動。
  menu 的高度在開框時定好(`spaceMenu.openRows`):開著時列變少補空白列、變多在框裡捲動，
  框不跟著伸縮(F7)。sort 欄位框的 Reset 因此一律在，沒有排序時變暗(M6),加了排序框也不變高。
  位置都是水平、垂直置中;toast 例外，貼在畫面下方，下框離底兩列、不蓋 footer(F7,
  跟 webu、locku 同一個位置)。
- **popup 疊層** — `stackOrder()` 是整疊由下往上的唯一順序:Space menu 在最底、它開出的
  picker / confirm / input / breadcrumb / yank viewport 在上、finder 在開它的 chooser 或
  Goto picker 之上、key reference 與 quit picker 最上。多步驟的流程每一步是自己的 popup
  (tdp F1):sort 的方向(`sortDirMenu`)疊在欄位(`sortMenu`)上、Goto 的 Favorites 清單
  (`gotoFavMenu`)疊在 Goto picker 上，不在同一個框裡換內容;`Esc` 回上一步，選定方向後
  方向框關掉、欄位框留著顯示新的排序鏈。`View` 照這個順序畫(`assignLayers()`
  依深度給層色),`Update` 反過來由上往下路由按鍵，所以最上面那個框收鍵、`Esc` 只關它。
  Space menu 的列若開出了框(`boxOverSpaceMenu()`),menu 留在底下;完成動作時(confirm
  接受、input 送出、breadcrumb 跳轉、open-in / open-with、finder 選定)`clearStack()` 整疊
  一起收掉(tdp F4、T1)。正在關閉的框不收鍵(`owns()`),下一個 `Esc` 直接關底下那層(F3)。
- **最上層以外全部 dim** — 有 popup 開著時，`View` 在合成最上層那個 popup 之前，先把整張已經
  畫好的畫面(base 與底下的 popup)過一次 `dimANSI()`:每個 SGR 前景 / 背景色往 base
  `#1e1e2e` 混(`dimKeep` = 0.45),每個通道取原值與混後較小的那個(比 base 暗的顏色不會被
  「淡化」成變亮，tdp D2),沒設顏色的文字補上 dim 過的預設色。這份是 tdp D2 引用的參考實作。所以底下 popup 的邊框
  是自己層色的 dim 版本、還看得出第幾層，串流內容與警示色一起 dim(tdp F8,T2 的例外)。
  toast 除了 `Esc` 不收鍵、不算一層(tdp F1、F8),畫在 dim 之後也不觸發 dim。popup 的合成
  順序在 `popupLayers()`,跟 `stackOrder()` 同序，PTY 墊在最底下。
- **`?` key reference** — 唯讀、可捲動、沒有游標(`helpPopup`)。按鍵路由在 quit 之後、所有
  popup 之前攔 `?`(輸入態除外，那裡 `?` 是字元),`keyRef()` 照疊層由上往下找最前面的
  surface:popup 各給自己的鍵;沒有 popup 時是 focus panel,由 `buildSpaceMenu()` 的 item /
  panel 列產生(`menuRows()`,跳過沒有鍵的 `Global operation`)再接 core key,所以跟 Space
  menu 不會對不上(tdp K6、M4)。quit picker 的 `?` 另開 `quitHelp`,疊在 picker 上(D3)。
  menu 裡變暗的列(`menuItem.disabled`),到了 key reference 也變暗(`helpRow.disabled`,鍵與
  說明都畫 `disabledColor`,區塊標題不變暗);`[2]` 沒有內容可開(空目錄、讀不到的檔、`[1]`
  是空的)時,Space menu 的 Yank 與 key reference 的 `y`、`Enter` 一起變暗(tdp M6,v0.1.14)。
  下框 hint 則只列現在按得了的鍵(條文讓 app 選):`[1]` 只有一個分頁時不列 `h/l`,Marks 與
  Favorites 分頁空的時候整條留白。

### 程式碼目錄

```
filu/
├── cmd/filu/           進入點:filu [path] / shell / version / iconwidth
├── internal/
│   ├── ui/             3 面板、popup、finder、preview、marks / tasks / favorites、PTY、splash
│   └── version/        版本字串(goreleaser 以 ldflags 注入,本機 build 是 dev)
└── docs/               dev-remarks.md、icon.svg、social-preview.png、demo-basics.gif
```

## 設計決定

- **`Enter` 在目錄上是進入，在檔案上是開資訊框，從不交給外部程式。** 開檔仍是 `[o]pen`
  (OS 預設 app,先 confirm)/ `[O]pen with`(挑 app)的職責：「進入」與「交給外部程式」
  是兩件不同代價的事(tdp P4),`Enter` 不兼後者。以前檔案列上的 `Enter` 什麼都不做，
  2026-09-28 裁定不符合 tdp K3(對 focus 項目做最直觀的動作),改成開唯讀的資訊框
  (`metaPopup`,`fileFacts()`):完整路徑(symlink 多一列目標)、類型(跟 preview 同一套
  判法)、大小(人話 + bytes)、Modified / Accessed / Created(Linux 是 Changed)、`rwx` +
  八進位、owner:group。值過長就折行、全部揭露，框寬照 F7、太高就捲動，讀不到的欄位
  在框裡寫原因(F5)。
  其他 panel 的 `Enter`(同日裁定):`[2]` 開可捲動的檢視(同 `y`);`[3]` Marks / Favorites
  在 `[1]` 找已經開著那個目錄的分頁、沒有就開新分頁(`showInTabs()`,Marks 游標停在該檔;
  分頁滿了跳 toast,不擠掉別的分頁);Tasks 把當前分頁帶到任務的目的地。完成後 focus 回 `[1]`。
- **quit 是 picker、不是 confirm。** 「離開時 shell 要 `cd` 去哪」是個選擇、不是一次確認
  (`quit.go quitMenu`):列出啟動目錄 + 各分頁的當前目錄,去重;有任務在跑時頂端插
  一條紅字 warning header。這是 tdp K9 的「離開流程由 app 決定」。`q` 與 `Ctrl-C` 都打開
  它(`Ctrl-C` 在輸入態也有效，`q` 在輸入態是字母),在 picker 上再按 `Ctrl-C` 立即離開。
  picker 疊在當下整疊 popup 的最上面、不關底下的框，所以 `Esc` 回到原本的框;按鍵路由
  在 splash、PTY、toast 的 `Esc` 之後第一個處理它，繪製時也只有 toast 畫在它上面。
- **PTY 的出口鍵是 `Alt-Esc`,按了先 confirm。** PTY 裡每個鍵都屬於 shell(包括 `Esc`、
  `Ctrl-C`、`q`),只有 `Alt-Esc` 被 filu 攔下(`isExitKey()`),常駐寫在 PTY 下框
  (`ptyExitHint`),是家族的出口鍵(tdp K10、D5)。filu 的 `[s]hell` 是用完就走的子 shell,
  沒有「離開後再接回」的 session,所以出口鍵等同打 `exit`:confirm 接受後殺掉 shell
  (`ptyPopup.exit()`)、播關閉動畫、reload 該目錄，回到 panel(2026-09-28 user 裁定)。
  先 confirm 是因為終端機把 Alt 組合送成「`Esc` 加那個鍵」,`Alt-Esc` 跟兩次 `Esc` 的 byte
  一樣;app 忙的時候兩次 `Esc` 會黏成 `Alt-Esc`,在 shell 裡的 vim 連按 `Esc` 就會連同沒存
  的編輯把 shell 殺掉(tdp D5,2026-09-29 user 裁定)。confirm 疊在 PTY 上、PTY 在它底下
  dim;`Esc` 回到 PTY,shell 照跑 —— 被讀成 `Alt-Esc` 的那兩個 `Esc` 不會送進 vim,回到
  PTY 後要再按一次。confirm 開著時鍵照一般 popup 的順序走(toast 的 `Esc`、`q` / `Ctrl-C`
  的離開流程、`?` 的 `Confirm keys`);沒有框疊在 PTY 上時(`boxOverPty()`),每個鍵照舊送進
  shell,toast 開著也一樣。shell 在 confirm 開著時自己結束,confirm 與它的 `?` 一起收掉。
  繪製上 PTY 在整疊 popup 的最底下(`popupLayers()` 第一個，`assignLayers()` 把它算成第
  1 層);開 shell 時 Space menu 與 Shell confirm 直接收掉、不播關閉動畫(`dropStack()`),
  免得收合中的框畫在 PTY 上、讓它跟著暗一下(F8)。
- **會改變磁碟或把控制權交出去的動作一律先 confirm**:`D` Delete(list)、`D`
  Unfavorite(Favorites)、`o` Open、`s` Shell、`Alt-Esc` 結束 shell(見上一條)、`C`
  Clear(Marks)。`Open` 要問,是因為交給外部 app 之後 filu 就管不到了;`Clear` 要問,
  是因為 bucket 是慢慢累積的、一鍵歸零沒有 undo。`m` mark / `p` pick / `f` favorite 是可逆的一鍵 toggle,不 confirm。
  `O` Open with 的 picker 不再另跳 confirm,連選 `Default`(跟 `o` 同一個結果)也一樣:
  在 picker 裡挑一個 app 本身就是一次明確的選擇，等於確認過了;`o` 是一鍵直達，才需要
  confirm 擋誤觸。所以「交給外部 app」每次都經過一次確認，符合 tdp F6(2026-09-28 user 裁定)。
- **input popup 在 `Enter` 當下驗證，不過就不送出。** Rename / Add / Zip 開框時掛上
  `nameCheck()`:空白、Rename 名稱含 `/` 或撞到現有名稱(改回原名放行)、Add 是 `.` / `..`
  或已存在，都留在框裡、在輸入列下方預留的錯誤列用紅字說原因，打字就清掉(tdp K3)。Rename 以前會
  `os.Rename` 直接蓋掉同名檔，現在在送出前就擋下。框寬照 F7 固定(`popupInnerWidth()`),
  錯誤列在開框時就留好、平常空白(掛了 `check` 的 input 才留，也就是送出可能失敗的那種),
  原因太長就截;值太長從左邊截。框的寬高都不跟著浮動(tdp F7、L2)。
  驗證過了仍可能在寫入時失敗(權限等),那個錯誤走 toast(tdp F5)。
- **Zip 打到 temp,再走既有的落地路徑。** 輸出位置固定是 `os.MkdirTemp("", "filu-zip-")`,
  因為使用情境是「打包完再搬去某個 `[1]` 的目錄」,輸出位置不等於目的地。打完的 zip
  成為唯一 pick,接既有 `c` / `v`,不另造「送到哪裡」的機制;打包範圍就是
  `landItems()`,與 Copy / Move here 同語意。
- **`/` 是一個 chooser、不是兩個熱鍵。** 早期 `/`=名稱、`f`=內容各佔一鍵;收攏成
  `/` → {filename, content} 之後,`f` 才空出來給 Favorite。
- **分頁標籤是一個動物 glyph、不是目錄名。** basename 長度不定,會讓 tab bar 寬度跟內容
  綁動(tdp L2);單一 glyph 是固定寬的位置標記。「這個 tab 在哪」由面板內第一列的
  breadcrumb 承載。第一格固定是 rocket(= `iconCWD`,quit picker 用的同一顆)。
- **`maxTabs = 5` 恆定**:讓 tab bar 寬度與 zoom 分欄數有上界。zoom 時有 tab 的面板依
  實際 tab 數攤成等寬並排欄(`splitN(w, len(m.tabs))`),每欄各自顯示自己的 breadcrumb。
- **面板數是成本。** 早期是 5-panel + header 麵包屑 + top status bar;v0.2.0 收斂成 3-panel
  (Places 併進 Favorites tab 與 Goto picker、Meta 併進 list 的多欄、Carries 與 Tasks 併成
  `[3]`),v0.2.8 再拆光頂部兩列。能收進既有面板的就不開新面板。
- **breadcrumb 是純文字、不是 chip**:正上方的 border 就是一排 tab chip,兩排 chip 疊在
  一起會打架。
- **mark 欄是單格、狀態合併不並排**(`list.go markCell`):「在 bucket」與「是 favorite」
  同時成立時用第三個合併 glyph `markFavGlyph` 佔同一格,不並排(並排會讓欄寬跟狀態
  綁動)。list 的 `m`(在 bucket)與 `[3]` Marks 的 `p`(在 land 子集)用兩個不同 glyph
  (`f0b14` / `f05d`),兩個狀態不共用。
- **`[2]` Preview 失焦不變暗**:它是邊看別的面板邊讀的參考視角。
- **Zoom 是版面的切換，不是模式。** `z` 把 focus 的面板展開佔滿全畫面，再按一次 `z` 才還原
  (`toggleZoom()`;focus 移到別的面板也會還原，`setFocus()`)。zoom 中沒有鍵換意思,`Esc`
  仍是「回上一層目錄」:使用者在 zoom 的多欄裡照樣瀏覽,`Esc` 若拿去退出 zoom,就在同一個
  畫面裡兼了兩種意義(P4)。tdp v0.1.14 的術語「模式」把版面的切換排除在外 ——`Esc` 不必
  退出它，由它自己的鍵還原。以前記在「偏離 tdp」(K4),v0.1.14 起不再是偏離(2026-09-29
  user 裁定)。
- **平台:只支援 macOS / Linux。** `GOOS=windows` **刻意編譯失敗**:平台分岔操作
  (metadata / hidden / roots / trash / open)走 platform interface,unix 實作用 build tag,
  沒有 Windows 實作。Windows 使用者走 WSL。

  不做原生 Windows 的理由(2026-07-27 拍板):platform 插槽只是冰山一角,還有沒插槽的
  結構性坑 —— `[s]hell` 內嵌 PTY(`creack/pty` 是 unix-only)、路徑 / `~` / 分隔符散在
  多處 render、cd-on-quit 的 POSIX shell wrapper、`rwx` / owner 權限顯示在 Windows 沒有
  對應模型。Git Bash 也不算:它是 POSIX 外殼、底下仍是原生 Windows PE。

## 已否決，不要重提

接 fzf binary(fzf-in-PTY:彩色 rg + 每鍵 reload + preview 把 vt10x 畫爆、root 改 Home
掃整棵卡死)、Goto 用 mtime 排序(mtime 追的是「OS / 工具碰過」不是「你想跳過去」)、
Zip 輸出到 active tab 的目錄(污染一個可能只是中途站的目錄)、Zip 輸出到 picks 的共同父
目錄(跨目錄時退化成 `/` 或 `~`)、`config.yaml` 的 `zip_dir` 旋鈕(需求還沒出現就先做
設定)、`/` 與 `f` 各佔一個搜尋熱鍵、5-panel layout 與頂部的 header / status bar、
powerline 漸層 header(實心三角的斜邊就是兩段背景的色界,低對比的漸層切不出三角;
亮→暗漸層要用 WCAG 對比度翻文字色)、分頁用羅馬數字 `Ⅰ`–`Ⅲ` 標記(EA-ambiguous,
CJK 字型畫 2 格)、分頁標籤用目錄名、`gt` 當 Goto chord(vim 的 go-to-tab)、原生 Windows
與 Git Bash。

## 已知的牆與未做

- **平台**:macOS 與 Linux(WSL 可),見「設計決定」。
- **目錄大小**:size 欄對目錄畫 `-`,不遞迴加總(會 walk 整棵子樹、卡)。
- **`[s]hell` 裡的 icon**:PTY 的內容是 vt10x 的格子，子程序(例：`eza --icons`)自己認定
  icon 佔一格;在 icon 畫成兩格的字型上，那一列會超出 shell 的框。子程序的排版 filu 改不了。
- **未做**:
  - Mouse(沒有 wire)。
  - 每列錯誤 `!` 前綴:broken symlink / 無權限 / 上次操作失敗(目前僅面板層 error note)。
  - zoxide 式磁碟快取索引(給 Goto 真 recency,只在串流不夠時做)。
  - chmod / extract、真圖(kitty / sixel)、sort filter、續傳。
- **tdp**:2026-09-28 對照 v0.1.7 全文修完，同日再跟上 v0.1.8–v0.1.10(F1 六類、F7 尺寸、
  F8 dim、K11 模式沒有按鍵清單)與 v0.1.11–v0.1.12(F7 loading icon、D2 淡化不變亮、D6
  truecolor)、v0.1.13(F1 放寬 menu 與 confirm,filu 不必改);2026-09-29 跟上 v0.1.14–v0.1.17
  (PTY 出口鍵先 confirm、按鍵的寫法與顏色、key reference 變暗、zoom 不算模式)與 v0.1.18–
  v0.1.19(icon 寬度全走顯示寬度、finder 的 focus、模式標示自己、失焦 hint 變暗、hint 整組捨棄)、
  v0.1.20(模式名夾在框線接頭之間)、v0.1.21(環境變數 `FILU__<名字>`、疊 popup 不 panic、
  選取模式照 vim 移動)、v0.1.22(`TERMINU__ICON_WIDTH`、又寬又高的框也切),都符合(沒有偏離，
  見下一節)。之後發現沒寫理由的違反，列進
  `docs/filu-terminu-fix.md`(目前沒有這個檔)。

## 偏離 tdp

目前沒有。(zoom 不讓 `Esc` 退出，以前記在這裡;tdp v0.1.14 起 zoom 不算模式，見「設計決定」。)

## 對照 tdp 時確認過的

2026-09-28 到 09-29 八次修 `filu-terminu-fix.md`(v0.1.7、v0.1.10、v0.1.12、v0.1.17、v0.1.19、v0.1.20、
v0.1.21、v0.1.22 各一輪，清單都已刪;v0.1.13 只改文件)時留下的：下次對照不必重查的，以及當時由 user
逐題裁定的。

**已經符合、不用修的**(對照 v0.1.7)

- **K2、K8(多行文字寫入狀態的 `Tab` 是縮排)**:filu 沒有多行文字輸入。input popup 與
  finder 的輸入列都是單行;能寫多行的只有 `[s]hell` 的 PTY,按鍵屬於子程序(K10)。
- **K2(單一輸入框有灰字提議時 `Tab` 接受提議)**:filu 沒有灰字提議。Rename 預填原名、Zip
  預填 `suggestZipName()`,兩者都是可以直接編輯的**值**,不是灰字;Add 是空的。input popup
  裡 `Tab` 不作用;finder 輸入列同樣沒有提議,`Tab` 在輸入列與結果清單之間切換。
  finder 打字時是 input(tdp F1):`Enter` 就是送出，直接選反白的那一筆(預設第一筆);方向鍵
  在候選之間移動,`j` / `k` 仍是字元(K8);要用 `j` / `k` 就先 `Tab` 進清單(2026-09-28 user
  裁定，以前打字時的 `Enter` 跟 `Tab` 一樣只是把 focus 交給清單)。
- **K10(至少一個出口鍵)**:filu 的 PTY 只有一格 shell,只需要出口鍵 `Alt-Esc`。
- **M3 與 P3「同一個動作在兩區」**:沒有熱鍵同時出現在兩個區。全域動作只有離開;切分頁、
  Goto、Search、Shell、Sort 都作用在 `[1]`,是 `[1]` 的 panel operation。
- **K5 在其他 popup**:只有真正的 Space menu 讓 `Space` 關閉(`spaceToggle`);input popup 與
  finder 輸入態的 `Space` 是字元(K8)。
- **M2 vs M6(`Favorite` 只在目錄列出現)**:檔案**永遠**不能收藏，不是「現在不能」,是動作
  對這個項目不成立，照 M2 不列;M6 的變暗只給狀態一變就能做的動作(例如分頁已滿的 `Tab`)。
- **F6(`O` 的 picker 不 confirm,`o` 會)**:見「設計決定」的 confirm 那條。
- **S3、S4**:splash 在按鍵路由第一站;`V` 只在主 switch,popup、輸入態、PTY 都叫不出來。

**已經符合、不用修的**(對照 v0.1.10)

- **F7 terminal 類的尺寸**:「terminal 寬 − 2 × 高 − 2」指內容區(user 2026-09-28 裁定)。
  shell popup 的外框貼滿畫面，內容區正好是 W−2 × H−2。
- **F7 finder 的高度**:固定是畫面的 90%,不依內容。結果是開框之後才串流進來，開框當下
  不知道會有幾筆,「打開時定好」只能依畫面定;開框後不再伸縮，loading 中也不變(v0.1.11
  允許 loading 時變，不要求)。

**已經符合、不用修的**(對照 v0.1.12)

- **F8 / D2 的 dim**:`dim.go` 就是 D2 引用的參考實作 —— 前景、背景都淡化，16 / 256 色先換 RGB,
  一律輸出 24-bit,沒設前景的文字補 `dim(#cdd6f4)`,其他 SGR 不動;v0.1.12 的「絕不變亮」這輪補上。
- **F7 其他 popup 的 loading**:只有 finder 的內容是開框後才來的;其他 popup 開框時內容都已確定
  (metadata 開框同步 `stat`、viewport 拿現成的 preview)。
- **F7 開著時改變高度**:loading 與使用者操作兩種情況都是「可以」不是「必須」。menu 開框定高
  (`spaceMenu.openRows`,Favorites 取消收藏後補空白列),finder 依畫面定高，都符合。
- **K3 內容區的 `Enter`**:`[2]` preview 的 `Enter` 開可捲動的檢視，正是條文的例子;Tasks、Marks、
  Favorites 有 cursor,是 focus 項目。
- **F6 picker 選定算確認**:`O` 的 picker 選了 app 不再 confirm,條文現在明文允許。

**已經符合、不用修的**(對照 v0.1.13)

- **F1 menu 的 `Enter` 可以打開那一列的全文、confirm 可以帶回答前要看的內容**:兩條都是「可以」。filu 的 menu 每一列
  `Enter` 都是執行，沒有「只有 cursor、沒別的動作」的清單;confirm 只有一句問句(Delete、Unfavorite、Clear marks、
  Open、Shell),沒有要先讀的明細。沒有東西要改。

**已經符合、不用修的**(對照 v0.1.14–v0.1.17)

- **F1、F8 toast 除了 `Esc` 不收鍵、不觸發 dim**:`Update` 的 toast 分支只認 `esc`,其他鍵照常
  往下路由(`TestF3ToastLetsOtherKeysThrough`);toast 不在 `popupLayers()` / `stackOrder()`,
  `View` 在 dim 與合成之後才畫它。
- **K11 模式裡回應 `Tab` 的 toast,第一個 `Esc` 先收它**:toast 的 `Esc` 分支在 `detailYank`
  路由之前，第一個 `Esc` 收 toast、選取還在，第二個才離開選取(`TestK11TabAnswersWhileSelecting`)。
- **PTY 開著時的 toast**:沒有框疊在 PTY 上時，`Esc` 給 shell,toast 等時間到自己收 ——
  terminal 類「按鍵都給子程序，只有出口鍵屬於 app」(F1、K10)優先;PTY 上疊了 confirm 時，
  toast 的 `Esc` 才先收(`TestPtyToastEscOrder`)。
- **D5 其他 Alt 組合的出口鍵**:filu 的 PTY 只有 `Alt-Esc` 一個 app 鍵，不適用。
- **術語「模式」**:filu 唯一的模式是 yank viewport 的選取;zoom 見「設計決定」。
- **M5 的 label**:`[]` 只出現在 label —— menu 的列(`bracketHotkey()`)與 panel 標題。key
  reference 說明裡的 `[1]`(`enterDesc()`)是 panel 的名字，不是熱鍵標記。
- **M6 描述別的 surface 的段落(v0.1.16)**:filu 的 key reference 沒有這種段落 —— 區塊標題
  `item operation`、`panel operation`、`keys` 都是這個 surface 自己的鍵。

**已經符合、不用修的**(對照 v0.1.18–v0.1.19)

- **L5 focus 不只靠顏色**:focus 的 panel 畫雙線 `╔═╗`、失焦圓角(`panelBoxHint()` 的框線
  選擇),兩者同寬。
- **D6 的探測**:`cmd/filu/main.go` 在 `tea.NewProgram` 之前呼叫 `ui.DetectIconWidth()`(CPR
  探測，失敗維持 1,`FILU__ICON_WIDTH` 可覆寫),`filu iconwidth` 印出結果;`dimANSI()` 只改
  SGR、不量寬度。
- **K11 focus 的 panel 保留線型**:filu 的模式在 popup 裡，不適用。
- **D2 失焦 hint 的其他地方**:`[1]` 失焦時不顯示 hint、`[2]` 沒有 hint、zoom 時 panel 一定是
  focus 的;要改的只有 `[3]`。
- **K10 子程序還沒準備好收鍵**:shell 同步啟動,`ptyPopup.update()` 在 `ptmx` 建好之前不轉送;
  條文是「可以」。一開就轉送,`Ctrl-C` 是 shell 的，出口鍵 `Alt-Esc` 照樣有效、常駐揭露。
- **K9 PTY 裡 `q`、`Ctrl-C` 屬於子程序**:沒有框疊在 PTY 上時全部送進 shell
  (`TestPtyKeysBelongToShell`);`Alt-Esc` 的 confirm 疊在上面時 focus 已經不在 PTY,`q` /
  `Ctrl-C` 進離開流程。

**已經符合、不用修的**(對照 v0.1.20)

- **D6 的新寫法**:filu 就是參考實作。CPR 量的是游標實際前進幾格(`DetectIconWidth()`),
  `FILU__ICON_WIDTH` 覆寫、`filu iconwidth` 查看;Windows 的預設不適用(filu 沒有 Windows 版)。
  驗收的 grep(`lipgloss.Width` 等)在 `internal/ui` 除了 `width.go` 找不到 —— 這輪把 viewport 的
  `cellsBefore()` 也搬進 `width.go`。
- **D3 模式名一個詞、先截標題**:`Selection` 一個詞;`drawPopupBoxMode()` 窄時先截標題。
- **D3 膠囊跟著換色**:filu 的模式在 popup 裡，popup 沒有 `[N] label` 膠囊，不適用。

**已經符合、不用修的**(對照 v0.1.21)

- **D5 選取文字的模式照 vim 移動**:yank viewport 的選取有 `h/j/k/l`、`w/b/e`、`0/$`、`gg/G`、
  `u/d`(`w/b/e` 是 user 實機後要求、filu 先做再回饋給 tdp 的，v0.1.21 採納)。
- **D6 的 `<APP>__DATA` / `__CACHE`**:filu 沒有另外的資料或快取目錄，不適用;給別的程式讀的
  變數 filu 沒有。

**已經符合、不用修的**(對照 v0.1.22)

- **D6 又寬又高的框也切**:`compositeDisp()` 上一輪(`b2436f3`)就拿掉了「又寬又高就原樣回傳」,
  `TestD6CompositeDispOversized` 就是 v0.1.22 指的參考測試。
- **D6 環境變數過不了 ssh**:filu 沒有遠端 session,不適用。

**user 裁定的**(2026-09-28、2026-09-29)

- 檔案列上 `Enter` 不做事 → 不符合 K3,改成開 metadata popup(`metaPopup`)。`[2]`、`[3]`
  的 `Enter` 見「設計決定」的 `Enter` 那條。
- `Favorite` 只在目錄上列出 → 維持(見上)。
- breadcrumb popup 的 `b` 兼關閉 → 拿掉，只留 `Esc`(其他熱鍵開的 popup 都只認 `Esc`)。
- `[1]` 的 Space menu 標題 → `[1] <cursor 項目名>`(空目錄 `[1] CWD`):照 D4 的 `[N]`,又保留
  item operation 作用在哪個檔案。
- PTY 出口鍵 → `Alt-Esc`,按下直接結束 shell。
- (2026-09-29)PTY 出口鍵先 confirm → 照 tdp D5(v0.1.16):`Enter` 結束 shell、`Esc` 回到
  PTY。上一條的「直接」不再成立，結束仍是殺掉 shell。
- `O` 的 picker 要不要 confirm → 不要(見「設計決定」)。
- `[3]` Tasks 執行中的轉圈 → 換成 D3 的 loading icon,全 app 只有一種轉圈(原本是 braille 點)。
- finder 載入中 → 邊串流邊列出結果(以前要等走訪結束才顯示清單)。
- (2026-09-29)下框與 footer 的小寫 `enter`、`space`、`esc`、`tab` → 大駝峰;toast 的 `(w)`
  → `[w]`;README 兩份照同一套鍵名(tdp M5,v0.1.15)。
- (2026-09-29)menu 與 key reference 說明欄裡提到的鍵算不算句子 → 算，加方括號
  (`next tab [h]/[l]`);tdp v0.1.17 照這個裁定寫進 M5。
- (2026-09-29)zoom 不讓 `Esc` 退出 → 照 v0.1.14 的術語「模式」已經不是偏離，移到「設計決定」。
- (2026-09-29)viewport 選取的反白 → 從 Lavender 改成 Yellow(照 D2),跟選取模式的黃框一致
  (實機看過 Lavender 反白配黃框不搭)。

## 設計文件導讀

filu 沒有另外的設計文件;功能與決定的原始清單在 `.forge/meta/IDEA.md`(本機、不進版控)。
tdp 本身在 [terminu](https://github.com/vulcanshen/terminu/tree/v0.1.23/principle)。

| 檔案 | 內容 |
|---|---|
| [`icon.svg`](icon.svg) | 圖示;`V` 的 splash 照它畫 |
| [`social-preview.png`](social-preview.png) | GitHub 的 social preview |

以 Go 與 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 打造。取法對象:cd-on-quit 對標
[superfile](https://github.com/yorukot/superfile) 的 `cd_on_quit`,finder 取法
[LazyVim](https://github.com/LazyVim/LazyVim) 的 search。

## 建置與開發

```bash
git clone https://github.com/vulcanshen/filu.git
cd filu
CGO_ENABLED=0 go build -o filu ./cmd/filu   # 或:make build
./filu
```

- 需要 **Go 1.26+**。
- 一律 `CGO_ENABLED=0`,產出靜態 binary(同 kbu)。
- `Makefile` 收攏常用工作 —— `make build`(→ `./filu`)、`make install`(→
  `$GOPATH/bin`)/ `make uninstall`、`make package`(當前平台的 `.tar.gz` 到
  `dist/`)、`make check`(fmt + vet + test)。跑 `make` 列出全部。
- 本機 build 的版本號永遠是 `dev`;release 版號由 goreleaser 以 `-X` 注入
  `internal/version.Version`。
- 除了主程式,binary 還有幾個子指令:`filu version`、`filu shell`(印出
  cd-on-quit 的 shell wrapper)、`filu iconwidth`(印出偵測到的 icon 格寬,除錯用)。
- `FILU__REPAINT=1`:每次導覽強制整頁重畫,給把 Nerd Font glyph 畫得比游標前進更寬、
  留下殘影的終端機用。

測試是 table-driven + programmatic model test(送 msg、斷言 state / render),不靠真終端。
幾個踩過的坑:

- **新加 popup 一定要把它的 `handleTick` 接進 `Update` 的 `AnimTickMsg` batch**,否則一開
  就 hang 在動畫第一格。直接把 `anim.state` 設成 open 的測試抓不到 —— 要用真的 tick 驅動。
- **在 panel body 上方加 / 減裝飾列,必定要同步 cursor 的 row 預算。** 拆頂部列時
  `listRows()` 還停在舊的 `-3`、`midHeight()` 還在扣不存在的兩列,cursor 到不了最後幾列、
  zoom 少了三分之一。現在 `listBody` 與 `listRows()` 共用 `listChromeRows`,`midHeight()` 是
  `height - 1`。
- **回歸測試要對「真正畫出來的東西」斷言,不要自己重算一次高度。**
  `TestListRowsMatchesRender` / `TestDetailRowsMatchesRender` 走 `View()` 傳的同一個高度、
  數畫面上真正出現幾列,四種尺寸(含 zoom)。

### demo gif

README 只放一個 gif,`docs/demo-basics.gif`(owner 的 GitHub profile 也引用它),用 VHS 錄。
tape 照家族慣例放在 `.local/demos/`(gitignore,不進版控):先 `make build`,再
`vhs .local/demos/demo-basics.tape`。

## 發布

push 一個 `v*` tag,GitHub Actions(`.github/workflows/release.yml`)在 ubuntu 與 macOS 跑
`go test -race`,過了 goreleaser 打包 4 個 unix target(linux / darwin × amd64 / arm64)與
checksums、更新 vulcanshen/homebrew-tap 的 formula;release notes 是 `CHANGELOG.md` 對應的
那一節。CHANGELOG 只記 binary 行為的變動;`install.sh`、`uninstall.sh`、打包、CI 與文件的
改動不記。

- **goreleaser 的 `--release-notes` 不可靠**:workflow 最後一步用 `gh release edit` 把 body
  強制換成 CHANGELOG 那一節。
- **Homebrew 發 formula、不發 cask**:cask 的產物預設被 Homebrew 蓋上
  `com.apple.quarantine`,未簽章的 binary 會被 Gatekeeper 擋;formula 的 binary 不會。
- **安裝腳本**:`install.sh` 下載 release binary 後,會把缺少的 `ripgrep` / `fd` 一併裝進
  同一個目錄 —— 從各工具自己的 GitHub release 抓、免 sudo,binary 名就是 `rg` / `fd`
  (不是 Debian 的 `fdfind`)。該平台沒有預編 binary 時(如 Intel macOS 的 `fd`)才退回印
  安裝提示。Homebrew formula 則直接宣告 `ripgrep` + `fd` 依賴。

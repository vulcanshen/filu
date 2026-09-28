# filu 開發者備忘

開發 filu 時要提醒自己、以及與 AI 協作時記下的決策。filu 遵循
[terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.10/principle)（tdp）；
使用者要知道的在 README,這裡記的是「它為什麼長這樣、內部怎麼做」。

---

## 運作方式

### cd-on-quit 為什麼需要一行 shell 設定

這是 OS 限制、不是偷懶:一個 process 只能改**自己**的工作目錄,沒有任何 syscall 能改
**父** process(你的 shell)的 cwd。filu 是 shell 的子程序,自己 `cd` 影響不到 shell;
唯一能原生改 shell cwd 的是 shell **內建指令**,而 filu 是外部 binary。

所以採兩段式 handshake:`filu shell` 印出一個 shell function,它透過
`FILU_LAST_DIR_FILE` 給 filu 一個暫存檔;filu 離開時把選定目錄寫進去,function 再讀檔
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
- **Finder** — filu 自畫的分割 picker(清單 + 預覽),不是 fzf binary(fzf-in-PTY 試過
  失敗)。每種模式都串流列檔:用 `fd` 的走訪序、不排序,首批近乎立即、載入中就能濾;
  沒有 `fd` 時退回純 Go walk。以 `/` 或 `~/` 開頭的 query 會錨定到該絕對路徑,對整條
  路徑 fuzzy、深度限錨點下數層。Goto 一律掃隱藏目錄,噪音靠 `ignore_dirs` 黑名單擋。
- **Preview** — 讀 magic bytes 判型別:目錄 → 內層 tree、壓縮包 → 內容清單、圖片 →
  base64 `data:` URI、SVG → 高亮 XML、文字 → Chroma(catppuccin-mocha)高亮 + 行號、
  二進位 → hex + ASCII、PDF → 抽出的文字 + 頁數。
- **Preview yank viewport** — `[2]` 的 `y` 開一個覆蓋 preview 的 viewport(`detailyank.go`):
  vim cursor + `v` 字元級選取,行號 gutter 不進剪貼簿;preview 為了塞進面板寬折斷的
  續行(`previewModel.cont`)複製時**不補**換行,否則貼出來的 base64 / 長 URL 會斷掉。
  選取是 tdp K11 的「模式」:鍵表只有一張(`selectKeys`),選取中 `?` 的 key reference 與
  viewport 本身 key reference 的移動列都由它產生。模式裡沒有 Space menu、也沒有可執行的
  按鍵清單，模式的鍵直接按;選取中 `Space` 不作用(tdp v0.1.10 拿掉了 v0.1.2 起那份只用方向鍵
  移動的按鍵清單 `modeList`)。選取中 `Tab` 暫停但回一個 toast;選取外 `Space` 也不作用
  (K5)。下框 hint 分選取內外兩種，框寬跟著螢幕、不跟 hint 變(L2)。
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
  重寫,兩者刻意分檔。`FILU_CONFIG` / `FILU_STATE` env var 可各自覆蓋單一檔案(測試與
  demo 錄製用它隔離)。
- **CJK Nerd Font 寬度** — 有些 CJK Nerd Font(如 Maple Mono NF CN)把 file-type icon
  畫成 2 格。filu 啟動時用 CPR 偵測實際格寬,並透過自訂的 display-width 層排版,讓面板
  框線不會破。powerline caps `U+E0A0–E0D7` 刻意排除(它們單寬);不能靠終端的
  East-Asian-Width 全域旋鈕解,那會連帶改動其他字元的寬度。
- **控制字元清洗** — 檔名可能含控制字元(macOS 的 `Icon\r`)。`safeName`(`list.go`)在
  顯示時剝掉控制字元(也擋 ANSI injection),檔案操作仍用真實名;套在所有 name-render 點。
- **Space menu 是熱鍵的殼** — `buildSpaceMenu` 依 focus 組 item / panel 兩區
  (`groupedMenu`),選一列就把那個熱鍵送給 focus 的 panel(`dispatchFocusKey`)。
  新增一個動作,要同步加進對應 focus 的 Space menu。
- **`gg` / `go` chord** — 單一 `AppModel.pendingG` 掛在主 switch 的 chokepoint(所有 popup
  return **之後**、只管主面板):`gg` 落既有 `case "g"`、`go` 呼 `handleListKey("go")`。
- **popup 共用框** — 全部走 `drawPopupBox`(title 嵌上框、hint 嵌下框、內容上下各一列
  padding);yank viewport 與 finder 用 `drawPopupBoxPad(pad=false)` 貼齊邊框。popup
  內容列刻意不放 glyph(`lipgloss.Width` 會低估 ambiguous / PUA 寬度),glyph 只擺在框線上。
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
  `#1e1e2e` 混(`dimKeep` = 0.45),沒設顏色的文字補上 dim 過的預設色。所以底下 popup 的邊框
  是自己層色的 dim 版本、還看得出第幾層，串流內容與警示色一起 dim(tdp F8,T2 的例外)。
  toast 不握鍵盤、不算一層，畫在 dim 之後也不觸發 dim。popup 的合成順序在
  `popupLayers()`,跟 `stackOrder()` 同序。
- **`?` key reference** — 唯讀、可捲動、沒有游標(`helpPopup`)。按鍵路由在 quit 之後、所有
  popup 之前攔 `?`(輸入態除外，那裡 `?` 是字元),`keyRef()` 照疊層由上往下找最前面的
  surface:popup 各給自己的鍵;沒有 popup 時是 focus panel,由 `buildSpaceMenu()` 的 item /
  panel 列產生(`menuRows()`,跳過沒有鍵的 `Global operation`)再接 core key,所以跟 Space
  menu 不會對不上(tdp K6、M4)。quit picker 的 `?` 另開 `quitHelp`,疊在 picker 上(D3)。

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
- **PTY 的出口鍵是 `Alt+Esc`,按下直接結束 shell。** PTY 裡每個鍵都屬於 shell(包括 `Esc`、
  `Ctrl-C`、`q`),只有 `Alt+Esc` 被 filu 攔下(`isExitKey()`、`ptyPopup.exit()`),常駐寫在
  PTY 下框(`ptyExitHint`)。跟 sshu 同一個鍵(tdp K10 的例子)。filu 的 `[s]hell` 是
  用完就走的子 shell,沒有「離開後再接回」的 session,所以出口鍵等同打 `exit`:殺掉 shell、
  播關閉動畫、reload 該目錄，回到 panel(2026-09-28 user 裁定)。
- **會改變磁碟或把控制權交出去的動作一律先 confirm**:`D` Delete(list)、`D`
  Unfavorite(Favorites)、`o` Open、`s` Shell、`C` Clear(Marks)。`Open` 要問,是因為
  交給外部 app 之後 filu 就管不到了;`Clear` 要問,是因為 bucket 是慢慢累積的、一鍵
  歸零沒有 undo。`m` mark / `p` pick / `f` favorite 是可逆的一鍵 toggle,不 confirm。
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
- **未做**:
  - Mouse(沒有 wire)。
  - 每列錯誤 `!` 前綴:broken symlink / 無權限 / 上次操作失敗(目前僅面板層 error note)。
  - zoxide 式磁碟快取索引(給 Goto 真 recency,只在串流不夠時做)。
  - chmod / extract、真圖(kitty / sixel)、sort filter、續傳。
- **tdp**:2026-09-28 對照 v0.1.7 全文修完，同日再跟上 v0.1.8–v0.1.10(F1 六類、F7 尺寸、
  F8 dim、K11 模式沒有按鍵清單),除了下一節的偏離都符合。之後發現沒寫理由的違反，列進
  `docs/filu-terminu-fix.md`(目前沒有這個檔)。

## 偏離 tdp

- **Zoom 不是 `Esc` 會退出的模式(K4)。** `z` 把 focus 的面板展開佔滿全畫面,再按一次
  `z` 才還原;zoom 中 `Esc` 仍是「回上一層目錄」。zoom 是版面、不是任務模式 —— 使用者
  在 zoom 的多欄裡照樣瀏覽,`Esc` 若拿去退出 zoom,就在同一個畫面裡兼了兩種意義(P4)。

## 對照 tdp 時確認過的

2026-09-28 兩輪修 `filu-terminu-fix.md`(v0.1.7 一輪、v0.1.10 一輪，清單都已刪)時留下的：
下次對照不必重查的，以及當時由 user 逐題裁定的。

**已經符合、不用修的**(對照 v0.1.7)

- **K2、K8(多行文字寫入狀態的 `Tab` 是縮排)**:filu 沒有多行文字輸入。input popup 與
  finder 的輸入列都是單行;能寫多行的只有 `[s]hell` 的 PTY,按鍵屬於子程序(K10)。
- **K2(單一輸入框有灰字提議時 `Tab` 接受提議)**:filu 沒有灰字提議。Rename 預填原名、Zip
  預填 `suggestZipName()`,兩者都是可以直接編輯的**值**,不是灰字;Add 是空的。input popup
  裡 `Tab` 不作用;finder 輸入列同樣沒有提議,`Tab` 在輸入列與結果清單之間切換。
  finder 打字時是 input(tdp F1):`Enter` 就是送出，直接選反白的那一筆(預設第一筆);方向鍵
  在候選之間移動,`j` / `k` 仍是字元(K8);要用 `j` / `k` 就先 `Tab` 進清單(2026-09-28 user
  裁定，以前打字時的 `Enter` 跟 `Tab` 一樣只是把 focus 交給清單)。
- **K10(至少一個出口鍵)**:filu 的 PTY 只有一格 shell,只需要出口鍵 `Alt+Esc`。
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
  不知道會有幾筆,「打開時定好」只能依畫面定;開框後不再伸縮。

**user 裁定的**(2026-09-28)

- 檔案列上 `Enter` 不做事 → 不符合 K3,改成開 metadata popup(`metaPopup`)。`[2]`、`[3]`
  的 `Enter` 見「設計決定」的 `Enter` 那條。
- `Favorite` 只在目錄上列出 → 維持(見上)。
- breadcrumb popup 的 `b` 兼關閉 → 拿掉，只留 `Esc`(其他熱鍵開的 popup 都只認 `Esc`)。
- `[1]` 的 Space menu 標題 → `[1] <cursor 項目名>`(空目錄 `[1] CWD`):照 D4 的 `[N]`,又保留
  item operation 作用在哪個檔案。
- PTY 出口鍵 → `Alt+Esc`,按下直接結束 shell。
- `O` 的 picker 要不要 confirm → 不要(見「設計決定」)。

## 設計文件導讀

filu 沒有另外的設計文件;功能與決定的原始清單在 `.forge/meta/IDEA.md`(本機、不進版控)。
tdp 本身在 [terminu](https://github.com/vulcanshen/terminu/tree/v0.1.10/principle)。

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
- `FILU_REPAINT=1`:每次導覽強制整頁重畫,給把 Nerd Font glyph 畫得比游標前進更寬、
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

# filu 開發者備忘

開發 filu 時要提醒自己、以及與 AI 協作時記下的決策。filu 遵循
[terminu design principle](https://github.com/vulcanshen/terminu)（tdp，目前跟到 v0.8.0），畫面建在家族共用的
[terminu-go](https://github.com/vulcanshen/terminu-go)（v0.1.13）上；使用者要知道的在 README，這裡記的是「它為什麼長這樣、內部怎麼做」。

---

## 運作方式

### terminu-go 做什麼、filu 做什麼

畫面的規定（core key、Space menu、key reference、panel 的框與 hint、footer、popup 的疊法與開關動畫、淡化、confirm、input、
toast、finder、viewer、PTY、splash、icon 與 powerline 符號的寬度探測與量法）都是 terminu-go 的（tdp E8）。filu 只登記：

- **一個畫面、三個 panel**（`filu.go` 的 `build`）：`[1]` 與 `[2]` 在上面 2:1 並排、佔三分之二高，`[3]` 橫跨下面（`layout`）；
  寬度 `< 72` 時 lib 只畫 focus 的那一個（`Screen.NarrowBelow`）。
- **每個 panel 的動作**（`terminu.Action`：鍵、名稱、說明、範圍、hint、能不能按、做什麼）：`actions.go` 的 `filesActions`、
  `previewpanel.go` 的 `previewActions`、`drawer.go` 的 `drawerActions`；global 的在 `globalActions`。Space menu、key reference、
  熱鍵與下框 hint 都由這一份產生，不會對不上。
- **panel 的內容**：`[1]` 是每個 tab 一份 `terminu.List`，外面包一層 `filesBody` 畫路徑列（`files.go`）；`[2]` 是 `terminu.Pager`
  外面包一層 `previewBody`，放 filu 自己的預覽（`previewpanel.go`）；`[3]` 是三份 `terminu.List`，`OnTab` 換 `Body`（`drawer.go`）。
- **背景的工作**：複製、搬移、打包（`tasks.go`、`zip.go`）、fd 與 ripgrep（`finders.go`）、目錄監看（`watch.go`）都跑在 goroutine，
  經 channel 送回 `App.OnMsg`（`filu.go` 的 `onMsg`）。

`filesBody` 嵌入目前那個 tab 的 `*terminu.List`：lib 每個訊息之後呼叫 body 的 `moved()`（沒公開的 method，嵌入才拿得到），清單的
`OnMove` 因此照樣會跑，`[2]` 跟著 cursor 換（components/layout/panel 的「明細」；`[1]` 的 `Details` 是 `[2]`）。換 tab 時把嵌入的
List 換成那個 tab 的。

### cd-on-quit 為什麼需要一行 shell 設定

這是 OS 限制、不是偷懶：一個 process 只能改**自己**的工作目錄，沒有任何 syscall 能改**父** process（你的 shell）的 cwd。filu 是
shell 的子程序，自己 `cd` 影響不到 shell；唯一能原生改 shell cwd 的是 shell **內建指令**，而 filu 是外部 binary。

所以採兩段式 handshake：`filu shell` 印出一個 shell function，它透過 `FILU__LAST_DIR_FILE` 給 filu 一個暫存檔；filu 結束時
（`App.shutdown`）把目錄寫進去，function 再讀檔 `cd`。目錄是 `[Q]uit to` 選的那一個，沒選就是結束時所在 tab 的目錄。這也是為什麼
要用 `filu` 而不是 `./filu` 啟動 —— wrapper 攔截的是指令名 `filu`，帶路徑的呼叫會繞過它。

### 各功能的實作備註

- **檔案清單的欄位** —— `Modified`、`Owner`（user:group）、`Perms`（eza 配色：`r` 黃、`w` 紅、`x` 綠）、`Size`（eza 的
  color-scale，越大越暖；目錄顯示 `-`，絕不遞迴加總）、icon + 檔名，前面是 lib 的標記欄。panel 變窄時依 owner → size → modified
  → perms 的順序整欄收掉，檔名永遠留著（`fitCols`）；畫的時候欄位組合變了才重算列（`drawTab`）。
- **排序** —— filu 自己排（`sortItems`）：目錄一律在前，再照這個目錄的排序鏈（`sortByDir`，每個目錄各自記住、存進 `state.yaml`），
  最後依名稱。lib 的 `List.SortBy` 會把目錄打散，所以不用它；表頭的排序標示照 lib 的寫法自己寫進欄名（順位 `(2)`、U+F160 升冪、
  U+F161 降冪，`fileColumns`）。`TODO(lib FL6)`。
- **路徑列** —— `[1]` 的第一列：Lavender 純文字、斜線暗（`header.go` 的 `crumbRow`），下面直接接表頭、不畫線（panel 的表頭與
  打字列下面也不畫）；太長時前段縮成字首（`~/Documents/x` → `~/D/x`），再不夠中間縮 `…`，永不折行（`fitPathSegments`）。目錄是
  最愛時後面加一顆星。
- **先問的動作** —— Open、Delete、Shell、Clear、Unfavorite 用 `Action.Ask`：lib 把 confirm 疊在開它的 Space menu 上（`Esc` 回到
  menu），接受後才整疊收掉（tdp F4、T1）。問句要寫 cursor 那一項的名字，`Ask` 卻是固定的指標：lib 執行動作時先問 `Disabled`、
  再讀 `Ask`，所以 `Disabled` 順手把問句填好（`actions.go` 的 `asking`，`TODO(lib FL15)`）。
- **錯誤** —— `config.yaml` 讀不懂、`state.yaml` 讀不到或讀不懂、目錄監看起不來，啟動後立刻說（一個用 toast、好幾個用 error popup）。
  `state.yaml` 壞掉時不覆寫它（`keepState`）：寫回去就是把裡面的 tabs、marks、最愛、排序無聲丟掉。`[2]` 讀不到的檔是 panel 的錯誤
  （`Panel.Fail`：Red 的事實加點名按鍵的提示）。動作失敗開 error popup，標題寫「哪件事失敗」（`Rename failed`）。
- **marks** —— 一個跨目錄、跨 tab 的 bucket（`marks.go` 的 `marksModel`），存在 `state.yaml`。清單上的勾是 lib 的標記欄，換目錄、
  bucket 變動時用 `syncMarks` 照 bucket 重勾（`TODO(lib FL4)`：List 只能動 cursor 那一列）。`[3]` Marks 清單自己的標記就是 pick：
  有 pick 時只有 pick 的會落地（`landItems`）。複製保留 bucket；搬移把搬走的移出 bucket。
- **Zip** —— 打包「會落地的那些」到 `os.MkdirTemp` 的暫存目錄，成為 bucket 裡唯一的 pick，再走既有的 `c` / `v`。每個 pick 以自己的
  名字放進壓縮檔（目錄保留內部結構），同名的兩個都留。名字過 `zipFileName`（取 basename + 補 `.zip`）。
- **Tasks** —— 同磁碟搬移是瞬間 `rename`；跨磁碟、複製才逐項回報。log 是帶時間的白話（`taskText`）；中斷的任務存進 `state.yaml`、
  下次啟動標成 interrupted。執行中那一列用一個固定的 glyph 加 `3/10`（暫定，見下面）。
- **Finder** —— lib 的 `terminu.Finder`：檔名搜尋與 Goto 用 `Files: true`（lib 的檔案預覽與四列資訊），結果由 `fd` 串流進 `Items`
  （`streamFinder`，走訪序、不排序；沒有 `fd` 時退回 Go 的 walk），到 `finder_cap` 為止。Goto 一律掃隱藏目錄，噪音靠 `ignore_dirs`
  擋。比對照 components/input/finder（包含、不分大小寫、每個詞都要出現、順序不變），是 lib 做的。內容搜尋先用 text input 問要找
  什麼，再把 `rg` 找到的檔（一個檔一列、記下第一個符合的行）串流進 finder，預覽從那一行的上面幾行開始（`hitPreview`，照
  file-picker 的預覽：`hide_secrets_in_preview`、控制字元畫成 Red 的名字、讀不到寫灰字 `(unreadable)`）。
  `TODO(lib FL2)`：finder 不告訴 app 打了什麼，所以內容搜尋不能邊打邊找，以前 `/`、`~/` 開頭的路徑搜尋也拿掉了。
  `TODO(lib FL16)`：lib 的檔案預覽不能從某一行開始，所以內容搜尋的預覽是 filu 自己畫的，下面少了檔案的四列資訊。
- **預覽** —— 讀 magic bytes 判型別：目錄 → 三層的樹、壓縮包 → 內容清單、圖片 → base64 `data:` URI、PDF → 抽出的文字與頁數、
  文字 → Chroma（catppuccin-mocha）上色 + 行號、二進位 → hex + ASCII（`preview.go`）。Tab 畫成 `input_tab_width` 格。行依寬度產生
  （圖片的 data URI 照寬度折），所以在 `previewBody.Draw` 時才做。`TODO(lib FL3)`：lib 的檔案預覽只有 finder 用得到。
- **檢視與複製** —— `[2]` 的 `Enter` 用 lib 的 `Viewer` 打開預覽。文字檔交出去的是檔案原本的文字（最多 4 MB）：Tab、控制字元、
  CRLF、太長的行都由 lib 畫（`Lang` 是檔名，跟預覽一樣上色），複製出來是原樣；`[2]` 的 sanitize 只給 panel 用。選取模式是 lib 的
  （`v` 兩次、vim 的移動、`y` 複製）；`y` 在選取以外複製全部。圖片的 data URI 合回一行再交出去，lib 自己折行，複製出來不會斷。
  剪貼簿走 OSC 52（`clipboard.go`），穿得過 tmux 與 SSH。
- **檔案資訊** —— 檔案上的 `Enter` 用 `Viewer` 顯示 `fileFacts`（`facts.go`）：完整路徑（symlink 多一列目標）、類型（跟預覽同一套判法）、
  大小（人話 + bytes）、Modified / Accessed / Created（Linux 是 Changed）、`rwx` + 八進位、owner:group；讀不到的欄位寫原因。
- **刪除** —— 移到 OS 垃圾桶：macOS `~/.Trash`、Linux `~/.local/share/Trash/files`（`trash.go`）。
- **tab 的標記** —— 第一個 tab 是啟動的 glyph（它永遠開在啟動的目錄），其餘各一隻動物（`look.go` 的 `tabMarks`）。標記是 panel 的
  tab 名稱（lib 的膠囊串），固定寬，不跟目錄名綁動（tdp L2）。上限五個（`maxTabs`）：到上限時 New tab 停用。
- **zoom** —— `z` 切換 lib 的 `ForceNarrow`：只畫 focus 的 panel、佔滿畫面。`[1]` 在 zoom 時把每個 tab 畫成等寬並排的欄（`drawColumns`，
  欄間一條 Overlay0 的 `│`），`h|l` 換到的那一欄是亮的，其他的淡化（tdp P6），見「設計決定」。zoom 是版面的切換，
  不是模式：沒有鍵換意思，`Esc` 照樣回上一層目錄。
- **icon 與配色** —— 檔案類型的 glyph 取自 eza 完整的 icon 表（`ezadata.go`）；顏色來自烘進 binary 的
  `vivid generate catppuccin-mocha` `LS_COLORS`，依 eza 的優先序解析（目錄 → symlink → executable → 最長 suffix → 副檔名，`theme.go`）。
  執行時不讀 `LS_COLORS`。
- **即時刷新** —— fsnotify 監看每個 tab 的目錄，外部變動時 debounce 後重讀、cursor 留在同一項（`watch.go`、`session.go`）。
- **session** —— 第一個以外的 tab（目錄與 cursor）、marks、最愛、tasks、每個目錄的排序存進 `state.yaml`（`persist.go`、`session.go`）；
  第一個 tab 永遠開在啟動的目錄，啟動時 focus 在 `[1]`。
- **設定與 state 分開** —— `config.yaml` 是使用者手改的，`state.yaml` 每次變動都重寫。位置照 tdp E3：`$XDG_CONFIG_HOME/filu`，
  否則 `~/.config/filu`；`FILU__CONFIG`、`FILU__STATE` 各指定一個目錄。v0.4.0 以前 macOS 用 `~/Library/Application Support/filu`：
  第一次啟動時整個目錄搬到新位置、Info toast 說從哪搬到哪；搬不動就照舊用舊的、開 error popup 寫出兩個路徑（`moveOldSettings`）。
  最愛與 task 紀錄放 `~/.filu/data.yaml`（`FILU__DATA`），以前在 `state.yaml` 的第一次啟動時搬過去（`loadState`）。`config.yaml` 除了 filu 自己的
  `finder_cap`、`ignore_dirs`、`open_with`，還收家族共用的 `input_tab_width`、`hide_secrets_in_preview`，交給 lib。
- **檔名裡的控制字元** —— 檔名可能含控制字元（macOS 的 `Icon\r`）。`safeName`（`list.go`）在顯示時剝掉，檔案操作仍用真實名。
  使用者打的值由 lib 的 input 處理：控制字元畫成紅色的名字、送出時擋下。
- **名稱的檢查** —— Rename、Add、Zip 的 `Check`（`actions.go` 的 `renameProblem`、`addProblem`）：空的、撞名、Rename 含 `/`、Add 在
  目前目錄外面（`filepath.Clean` 後以 `/` 開頭、是 `..` 或以 `../` 開頭）、Zip 取不出檔名，都留在框裡、寫在錯誤列。寫入時才失敗的
  （權限等）開 error popup。

### 程式碼目錄

```
filu/
├── cmd/filu/           進入點：filu [path] / shell / iconwidth / version / help
├── internal/
│   ├── ui/             建在 terminu-go 上的畫面，以及檔案、預覽、搜尋、marks、tasks、最愛、session、設定
│   └── version/        版本（tdp project 的範本；goreleaser 以 ldflags 注入，本機 build 是 dev）
└── docs/               dev-remarks.md、icon.svg、social-preview.png、demo-basics.gif
```

## 設計決定

- **`Enter` 在目錄上是進入，在檔案上是看資訊，從不交給外部程式。** 開檔是 `[o]pen`（預設 app，先 confirm）與 `[O]pen with`
  （選 app）的事：「進入」與「交給外部程式」代價不同（tdp P4）。2026-09-28 user 裁定。
- **其他 panel 的 `Enter`**：`[2]` 打開檢視；`[3]` Marks、Favorites 在 `[1]` 找已經開著那個目錄的 tab、沒有就開新的（Marks 的
  cursor 停在那個檔；tab 滿了跳 toast，不擠掉別的 tab）；Tasks 讓 `[1]` 到任務落地的地方。focus 留在 `[3]`（tdp K3：只有明細可以
  用 `Enter` 換 panel），toast 說在 `[1]` 的哪裡。
- **會改變磁碟、或把控制權交出去的動作先 confirm**：`D` Delete、`D` Unfavorite、`o` Open、`s` Shell、shell 裡的 `Alt-Esc`、`C` Clear。
  `m` mark、`p` pick、`f` favorite 是可逆的開關，不問。`O` 的選單裡挑一個 app 就算確認過（tdp F6 允許），連 `Default` 也不另外問。
- **Zip 打到 temp，再走既有的落地路徑。** 使用情境是「打包完再搬去某個目錄」，輸出位置不等於目的地；打包範圍就是 `landItems()`。
- **`/` 是一個選單、不是兩個熱鍵。** `/` → {File names, Contents}；`f` 因此空出來給 Favorite。
- **tab 的標籤是一個 glyph、不是目錄名。** 目錄名長度不定，會讓膠囊串跟內容綁動（tdp L2）；目錄由路徑列說。
- **`maxTabs = 5`**：讓 tab 的膠囊串與 zoom 的欄數有上界。
- **面板數是成本。** v0.2.0 從 5 個收成 3 個，能收進既有 panel 的就不開新的。
- **平台：只支援 macOS / Linux。** 平台分岔的操作（metadata、trash、open）用 build tag，沒有 Windows 實作；`[s]hell` 的 PTY、路徑、
  cd-on-quit 的 POSIX shell function、`rwx` 權限都沒有 Windows 對應。Windows 使用者走 WSL（2026-07-27 拍板）。

- **zoom 時換 panel 照樣放大**（user 2026-10-19 定案）：zoom 用 lib 的 `ForceNarrow`，`Tab` 換到下一個 panel 時，下一個也是放大的，
  `z` 才還原。放大是使用者要的版面，換 panel 不代表不要了；跟窄寬時一次只看一個 panel 的走法一樣。
- **zoom 時 `[1]` 的 tab 並排**（user 2026-10-19 定案：tdp 的 zoom 給預設，怎麼放大由 app 決定）：filu 在 zoom 時把每個 tab 畫成一欄
  （`files.go` 的 `drawColumns`）：zoom 本來就是要更多空間，在好幾個目錄之間比對、搬檔時，一次看到全部的 tab 比一頁一頁切好用；`h|l`
  照樣換 tab，有 focus 的那一欄亮、其他的淡化（tdp P6），欄間的線用 Overlay0（跟 menu 的分隔線一樣）。
- **長時間工作的進度寫在 Tasks 的列裡**（user 2026-10-19 定案）：filu 是單畫面、沒有 statusbar，進度是 app 的內容（`3/10`，執行中的 glyph
  Green）；開著 shell 時看不到，接受。
- **設定、狀態、資料分開**（tdp E3）：用「刪掉會怎樣」分。tab、marks、每個目錄的排序刪了只是回到預設，是狀態（`state.yaml`，在設定目錄）；
  最愛與 task 紀錄刪了找不回來、會想備份，是資料（`~/.filu/data.yaml`）。

### 暫定（自己定的，等 user 確認）

- **`[3]` 的欄位**：Marks、Favorites 是 `NAME` 與 `IN`（所在目錄，從前面截）；Tasks 是 `WHEN` 與 `TASK`。理由：名稱與所在目錄分兩欄，
  名稱不會被長路徑擠掉，`IN` 從前面截、留下最近的幾層；Tasks 照時間讀，所以時間放前面。

### 定案了、等 terminu-go 下一版的（user 2026-10-19）

- **`q` 回到選目錄的 menu**（FL1）：現在是 confirm 加 global 的 `[Q]uit to`。
- **內容搜尋邊打邊找，`/`、`~/` 開頭的路徑搜尋回來**（FL2）。
- **Zip 的名字是灰字的建議值**（FL9，tdp S6）：不碰就按 `Enter` 用它；現在寫在說明列。
- **執行中的 task 轉家族的 loading icon**（FL11）：現在是固定的 nf-md-timer_sand。
- **檔案資訊的 viewer 不畫行號**（FL12，tdp S66），照樣能選取複製。
- **排序選了方向只收方向那一步、欄位 menu 留著**（FL13，tdp S4 的 T1 表：選一個值寫回 picker 保留 source）；現在整疊收起。

## 已否決，不要重提

接 fzf binary（fzf-in-PTY 把畫面畫爆、root 改 Home 掃整棵卡死）、Goto 用 mtime 排序（mtime 追的是「OS / 工具碰過」不是「你想跳過去」）、
Zip 輸出到 active tab 的目錄（污染一個可能只是中途站的目錄）、Zip 輸出到 picks 的共同父目錄（跨目錄時退化成 `/` 或 `~`）、
`config.yaml` 的 `zip_dir` 旋鈕、`/` 與 `f` 各佔一個搜尋熱鍵、5-panel layout 與頂部的 header / status bar、powerline 漸層 header、
tab 用羅馬數字 `Ⅰ`–`Ⅲ` 標記（EA-ambiguous，CJK 字型畫 2 格）、tab 標籤用目錄名、`gt` 當 Goto 的組合鍵（vim 的 go-to-tab）、原生 Windows
與 Git Bash。filu 自己的 popup、寬度層與 icon 探測（改用 terminu-go，tdp E8）。

## 已知的牆與未做

- **平台**：macOS 與 Linux（WSL 可），見「設計決定」。
- **目錄大小**：size 欄對目錄畫 `-`，不遞迴加總。
- **`[s]hell` 裡的 icon**：shell 裡的程式照 lib 設給它的 `TERMINU__ICON_WIDTH` 排版；不讀這個變數的程式（例：`eza --icons`）自己認定
  icon 佔一格，在 icon 畫成兩格的字型上那一列會歪。
- **等 lib 或規範的**（程式裡的 `TODO(lib FLn)`、`TODO(tdp FLn)`，已回報 terminu）：FL1 離開流程不能是選單；FL2 finder 拿不到打的字
  （內容搜尋先問字、沒有路徑搜尋）；FL3 panel 用的檔案預覽；FL4 照 ID 設標記；FL6 排序時目錄在前、
  整欄收掉；FL7 key reference 沒列 `Shift-Tab`；FL8 finder 在很小的畫面上當掉（測試跳過那一格）；FL9、FL11、FL12、FL13 見「設計決定」；
  FL14 menu 的標題不能照原樣放名稱（`Open with`、`Open in` 只寫動作）；FL15 confirm 的問句不能跟著 cursor 那一項；FL16 內容搜尋的
  預覽不能用 lib 的檔案預覽。
- **未做**：Mouse；每列的錯誤標示（broken symlink、沒有權限）；zoxide 式的目錄索引；chmod、解壓縮、真的圖（kitty / sixel）、續傳。
- **拿掉的**：`FILU__REPAINT`（每次導覽整頁重畫，給把 glyph 畫得比游標前進更寬的終端機用）：畫到終端機是 lib 的事。

## 偏離 tdp

目前沒有。zoom 時 tab 並排以前寫在這裡，tdp v0.8.0 讓 app 決定怎麼放大，移到「設計決定」。`[1]` 的路徑列、`[3]` 各清單的欄位是 panel
的內容；排序的標示照 components/layout/panel 的寫法（`TODO(lib FL6)`，等 lib 的群與收欄）。

## 設計文件導讀

filu 沒有另外的設計文件；功能與決定的原始清單在 `.forge/meta/IDEA.md`（本機、不進版控，內容停在 v0.2.0 以前）。
tdp 在 [terminu](https://github.com/vulcanshen/terminu)（`principle/`、`components/`、`project/`、`checklist/`），lib 在
[terminu-go](https://github.com/vulcanshen/terminu-go)。

| 檔案 | 內容 |
|---|---|
| [`filu-terminu-fix.md`](filu-terminu-fix.md) | 還沒符合 tdp、在等 lib 或規範的地方（tdp E7；全部修完就刪掉） |
| [`icon.svg`](icon.svg) | 家族 mark 的正本；`V` 的 splash 照它畫（程式裡的副本是 `internal/ui/icon.svg`，`TestIconMatchesDocs` 守住兩份一樣） |
| [`social-preview.png`](social-preview.png) | GitHub 的 social preview |

以 Go、[Bubble Tea](https://github.com/charmbracelet/bubbletea) 與 terminu-go 打造。取法對象：cd-on-quit 對標
[superfile](https://github.com/yorukot/superfile) 的 `cd_on_quit`，finder 取法 [LazyVim](https://github.com/LazyVim/LazyVim) 的 search。

## 建置與開發

```bash
git clone https://github.com/vulcanshen/filu.git
cd filu
make build        # → ./filu（CGO_ENABLED=0 靜態；版本是 dev）
./filu
```

- 需要 **Go 1.26+**。一律 `CGO_ENABLED=0`。
- `Makefile` 只設 `BINARY`、`TAPES` 再 include 家族共用的 `family.mk`（tdp project 的範本，照原樣複製、不在這裡改）。常用的：
  `make build`、`make install` / `make uninstall`、`make package`、`make check`（commit 前跑：fmt-check、vet、`go test -race`；有
  `go.work` 時加 `check-lib`）。跑 `make` 列出全部；filu 自己的 target 放在最後一段 `##@ filu 自己的`（目前沒有）。
- **改 lib 時**：`go.mod` 寫發布的 terminu-go 版本（不用 `replace`）；本機用 gitignore 掉的 `go.work`、`go.work.sum`（`go work init . ../terminu-go`）接上
  lib 的工作目錄。`make check` 的 `check-lib` 關掉 `go.work`、照 `go.mod` 的版本編譯與測試：過不了就是用到了 lib 還沒發版的東西。
- 本機 build 的版本號是 `dev`；release 版號由 goreleaser 以 `-X` 注入 `internal/version.Version`。
- binary 的子指令：`filu version`（也收 `--version`、`-v`）、`filu help`（也收 `--help`、`-h`）、`filu shell`（印出 cd-on-quit 的 shell
  function）、`filu iconwidth`（印出 lib 探測到的 icon 與 powerline 符號格數，以及從哪裡來的）。
- 頂層另有（tdp project 的目錄圖上沒有的）：`CLAUDE.md`（gitignore，給 AI 協作的指引）、`.claude/`（進版控：專案的 Claude Code
  規則、agent 與指令）、`.forge/`（gitignore：forge 的 GOAL.md、PROGRESS.md、IDEA.md、REDESIGN-LIST-AND-NAV.md，內容停在 v0.2.0
  前後）、`.checkpoints/`（自帶 `.gitignore`：AI 協作的 session 紀錄）。

測試是 programmatic 的（不靠真終端）：`harness_test.go` 照 terminu-go 的 app 的做法，把按鍵送進 `App.Update`、跑完它發出的
command（動畫、背景的 channel）、讀 `Frame()`。session、設定、垃圾桶、剪貼簿都指到測試自己的暫存目錄。

- **tdp L4 與 E5**：`screen_test.go` 在幾種尺寸、「icon 一格 / 兩格、powerline 符號一格 / 兩格」下，每一類 popup 各開一次，量每一列；
  再把畫面縮到比 popup 小（比畫面寬、比畫面高、兩邊都大）。popup 的開關動畫是真的 32ms 一格，所以這組測試要幾十秒。
- **lib 的檢查**：`terminu.App` 在登記時就拒絕佔用 core key 或移動字母的熱鍵（popup 裡也一樣，例：menu 不能用 `d`）—— 新加的
  menu 要在測試裡打開過一次，才知道沒踩到。

### demo gif

README 只放一個 gif，`docs/demo-basics.gif`（owner 的 GitHub profile 也引用它），用 VHS 錄。tape 照家族慣例放在 `.local/demos/`
（gitignore，不進版控）：`make gif`（先 build，再跑 `TAPES`）。錄之前先在 tmux 照 tape 的按鍵走一遍 —— Space menu 的列數一變，
tape 裡 `j|k` 的次數就錯。

## 發布

push 一個 `v*` tag，GitHub Actions（`.github/workflows/release.yml`）在 ubuntu 與 macOS 跑 `go test -race`，過了 goreleaser 打包 4 個 unix
target（linux / darwin × amd64 / arm64）與 checksums、更新 vulcanshen/homebrew-tap 的 formula；release notes 是 `CHANGELOG.md` 對應的
那一節（標題 `## vX.Y.Z — tdp vA.B.C`，改動先寫在 `## Unreleased`；v0.4.0 以前的標題照舊）。CHANGELOG 只記 binary 行為的變動；
`install.sh`、`uninstall.sh`、打包、CI 與文件的改動不記。`release.yml`、`.goreleaser.yaml`、`install.sh`、`uninstall.sh`、`family.mk`、
`.gitignore`、`internal/version/version.go` 是 tdp project 的範本，只改範本允許的地方（`app's own` 的段落、`app:` 的行）；把 `filu` 換回
`<app>` 跟同一版 tdp 的範本 diff，只該差那些地方。

- **CI 只在 tag 上跑**：打 tag 前先在乾淨的環境跑過測試 —— 只放 `go` 的 `PATH`（不要 `rg`、`fd`），以及 docker 的 `golang` 映像
  `go test -race ./...`。v0.4.0 第一次打 tag 時，一個測試因為 runner 沒有 `rg` 而失敗。
- **goreleaser 的 `--release-notes` 不可靠**：workflow 最後一步用 `gh release edit` 把 body 換成 CHANGELOG 那一節。
- **Homebrew 發 formula、不發 cask**：cask 的產物會被蓋上 `com.apple.quarantine`，未簽章的 binary 會被 Gatekeeper 擋；formula 不會。
  formula 宣告 `ripgrep` 與 `fd` 依賴。
- **安裝腳本**：`install.sh` 下載 release binary 後，在範本留給 app 的段落裡把缺少的 `ripgrep` / `fd` 一併裝進同一個目錄 —— 從各工具
  自己的 GitHub release 抓、免 sudo，binary 名就是 `rg` / `fd`（不是 Debian 的 `fdfind`）。該平台沒有預編 binary 時才退回印安裝提示。
  `uninstall.sh` 在 macOS 也問要不要刪舊的 `~/Library/Application Support/filu`。

# filu 開發者備忘

開發 filu 時要提醒自己、以及與 AI 協作時記下的決策。filu 遵循
[terminu design principle](https://github.com/vulcanshen/terminu)（tdp，目前跟到 v0.11.0），畫面建在家族共用的
[terminu-go](https://github.com/vulcanshen/terminu-go)（v0.6.0）上；使用者要知道的在 README，這裡記的是「它為什麼長這樣、內部怎麼做」。

---

## 運作方式

### terminu-go 做什麼、filu 做什麼

畫面的規定（core key、Space menu、key reference、panel 的框與 hint、footer、popup 的疊法與開關動畫、淡化、confirm、input、
toast、finder、viewer、PTY、splash、icon 與 powerline 符號的寬度探測與量法）都是 terminu-go 的（tdp E8）。filu 只登記：

- **一個畫面、三個 panel**（`filu.go` 的 `build`）：`[1]` 與 `[2]` 在上面 2:1 並排、佔三分之二高，`[3]` 橫跨下面（`layout`）；
  到最低寬度 60 欄也照樣並排（tdp L1、screen.5：`[1]` 40 欄、`[2]` 20 欄），所以不設 `Screen.NarrowBelow`。
- **每個 panel 的動作**（`terminu.Action`：鍵、名稱、說明、範圍、hint、能不能按、做什麼）：`actions.go` 的 `filesActions`、
  `previewpanel.go` 的 `previewActions`、`drawer.go` 的 `drawerActions`。Space menu、key reference、熱鍵與下框 hint 都由這一份產生，
  不會對不上。global 的只有 lib 自己的離開那一列；離開流程是 `drawer.go` 的 `syncQuit`。
- **panel 的內容**：`[1]` 是每個 tab 一份 `terminu.List`，外面包一層 `filesBody`（zoom 時並排、讀不到目錄時畫錯誤，`files.go`）；
  `[2]` 是 `terminu.Pager` 外面包一層 `previewBody`，放 filu 自己的預覽（`previewpanel.go`）；`[3]` 是三份 `terminu.List`，`OnTab`
  換 `Body`，Tasks 外面包一層 `tasksBody` 讓執行中的 icon 轉（`drawer.go`）。
- **背景的工作**：複製、搬移、打包（`tasks.go`、`zip.go`）、fd 與 ripgrep（`finders.go`）、目錄監看（`watch.go`）都跑在 goroutine，
  經 channel 送回 `App.OnMsg`（`filu.go` 的 `onMsg`）。

`filesBody` 嵌入目前那個 tab 的 `*terminu.List`：lib 每個訊息之後呼叫 body 的 `moved()`（沒公開的 method，嵌入才拿得到），清單的
`OnMove` 因此照樣會跑，`[2]` 跟著 cursor 換（components/layout/panel 的「明細」；`[1]` 的 `Details` 是 `[2]`）。換 tab 時把嵌入的
List 換成那個 tab 的。`tasksBody` 也是同一個做法。

### cd-on-quit 為什麼需要一行 shell 設定

這是 OS 限制、不是偷懶：一個 process 只能改**自己**的工作目錄，沒有任何 syscall 能改**父** process（你的 shell）的 cwd。filu 是
shell 的子程序，自己 `cd` 影響不到 shell；唯一能原生改 shell cwd 的是 shell **內建指令**，而 filu 是外部 binary。

所以採兩段式 handshake：一個 shell function 透過 `FILU__LAST_DIR_FILE` 給 filu 一個暫存檔；filu 結束時
（`App.shutdown`）把目錄寫進去，function 再讀檔 `cd`。目錄是 `q` 的 menu 選的那一個（`quitMenu`，lib 的 `App.QuitFlow`），用
`Ctrl-C` 兩次直接離開、或只有一個 tab 時 `q` 直接離開，是當時所在 tab 的目錄。有這個變數又有好幾個 tab 時 `q` 才是選目錄的 menu，
沒有變數就只是一個 confirm（`syncQuit`）。menu 與 confirm 都是第一步就問，所以 popup 開著時按 `q`（tdp K9-6）lib 不會再多問一次
`Quit filu?`；只有一個 tab、本來會直接離開時，popup 開著就由 lib 問 `Quit filu?`。這也是為什麼
要用 `filu` 而不是 `./filu` 啟動 —— wrapper 攔截的是指令名 `filu`，帶路徑的呼叫會繞過它。

**function 怎麼裝**（user 2026-10-24 定，`cmd/filu/shell.go`）：`filu shell install` 照 `$SHELL` 在 rc 檔（zsh 的
`$ZDOTDIR/.zshrc` 或 `~/.zshrc`；bash 的 `~/.bashrc`，macOS 是 `~/.bash_profile`，那裡的終端機開 login shell）寫一段有頭尾標記的
區塊，裡面一行：filu 在 PATH 上時 `eval "$(command filu shell init zsh)"`。function 每次開 shell 由當下的 filu 印，升級就跟上 ——
以前叫使用者把 `eval "$(filu shell)"` 放進 rc，有人照抄 function 本身，v0.4.0 改變數名（`FILU_` → `FILU__`）時抄的那份就悄悄失效
（user 自己的就是）。再跑一次是「已經裝了」，區塊內容不同就換新；寫回時跟著 symlink（dotfiles repo），保留權限。區塊外看起來是舊做法
的行（`filu shell`、`LAST_DIR_FILE`、`filu()`）只指出行號、不動。`filu shell uninstall` 從 zsh、bash 可能的三個 rc 檔拿掉區塊；
`uninstall.sh` 在刪 binary 前呼叫它（Homebrew 的 uninstall 沒有 hook，README 叫使用者先跑）。`filu shell` 本身不再印 function：
在終端機印說明；被舊的 `eval "$(filu shell)"` 呼叫（stdout 不是終端機）時印一行只往 stderr 說「改用 filu shell install」的 shell
code，開 shell 不會冒出 `command not found`。`init` 不寫進 usage：只給區塊裡那一行用。

### 各功能的實作備註

- **檔案清單的欄位** —— `Modified`、`Owner`（user:group）、`Perms`（eza 配色：`r` 黃、`w` 紅、`x` 綠）、`Size`（eza 的
  color-scale，越大越暖；目錄顯示 `-`，絕不遞迴加總）、icon + 檔名，前面是 lib 的標記欄。欄寬照 tdp S13（v0.8.2）：檔名是可以截短的欄
  （`Column.Shrink`），吃其他欄留下的寬度；其他欄照最寬的值、不縮。放不下時檔名先縮到 12 格（`MinWidth`），還放不下 lib 才依
  `Column.Drop` 整欄收掉：owner → size → modified → perms，檔名永遠留著（`files.go` 的 `fileColumns`）。一個很長的檔名只會被截短，
  不會把其他欄擠掉。最後一欄是最愛的星（Yellow，沒有表頭，也不收）：以前接在檔名後面，檔名一截短星就跟著不見。`[3]` 的名稱、路徑、
  task 的白話也標成可以截短。
- **排序** —— lib 排（`List.SortBy`）：每一列的 `Row.SortGroup` 讓目錄在前、排序只在群裡；`Row.Sort` 給排序用的值（大小是 bytes、
  時間是固定寬的 UTC 字串、名稱是原名）。filu 只交出照名稱排好的列（`sortItems`，不分大小寫），沒有排序時 lib 照這個順序；排序鏈
  每個目錄各自記住、存進 `state.yaml`（`sortByDir`）。表頭的順位與方向 glyph 是 lib 畫的。
- **位置列** —— `[1]` 的第一列是 lib 的 `List.Location`：只寫你在哪裡，整列 Lavender、太長從前面截、空目錄也留著（tdp v0.9.1，FL25）。
  你在的目錄是不是最愛，看上一層的清單或 `[3]` Favorites。
- **先問的動作** —— Open、Delete、Shell、Clear、Unfavorite、有工作在跑時的離開用 `Action.AskFor`：執行時才做問句，寫 cursor 那一項的
  名字；從 Space menu 跑時 lib 把 confirm 疊在 menu 上，`Esc` 回到 menu（tdp F4、T1）。
- **menu 做完之後留不留**（tdp T1）—— lib 預設：開了東西的保留開它的 menu、什麼都沒開的整疊收掉。filu 另外做的：Shell 用
  `Action.Clear`（開 shell 清掉 source）；finder 選了東西就進去，`Done` 先 `App.CloseAll`（drill-down 清掉 source，`revealFound`）；
  排序選了方向用 `Stay` 加 `App.CloseTop(1)` 只關方向那一步，欄位 menu 的說明是 `Action.Live`，回去就看得到新的排序。
- **讀不了自己的檔** —— `config.yaml`、`state.yaml`、`data.yaml`、`open_with.yaml`、`ignore_dirs.yaml` 讀不到或讀不懂，啟動時一個
  一個問（tdp E3「讀不了時」，lib 的 `App.MoveAside`；`filu.go` 的 `unread`、`askUnread`）：`Enter` 改名成 `.bak`（已有就
  `.bak.1`……），像第一次打開一樣開始（tdp v0.9.2）—— 第一次打開就寫的照樣寫：設定與兩份清單各寫一份新的範本（清單的範本是預設值：
  `open_with` 空的、`ignore_dirs` 是預設的那一串）；`state.yaml`、`data.yaml` 第一次打開不寫，移開後也等第一次存檔才有：session
  重讀（`reopenSession`，`data.yaml` 也壞了就接著問），不馬上存；`Esc` 離開，`Run` 回傳路徑與原因，`main` 印到 stderr。問之前不寫回去
  （`keepState`）：寫回去就是把裡面的 tabs、marks、最愛、排序無聲丟掉。filu 沒有快取。
- **錯誤** —— 目錄監看起不來、搬舊設定失敗，啟動後立刻說（一個用 toast、好幾個用 error popup）。好幾件事一件一列給 lib
  （`Error(title, rows...)`、`Toast(kind, rows...)`；`moveRows`、`startupErrs`），原因與該怎麼做也各一列：filu 自己的字不放 `\n`，畫面上
  紅色的 `\n` 一定是外面來的（tdp v0.9.15，components「控制字元」）。外面來的錯誤有好幾行（yaml 的型別錯誤：一行標題、一行一個錯）時，
  問移開的 confirm 只放說原因的那一行，整段接成一行記進 App Log（`App.why`）。`[1]` 讀不到的目錄、`[2]` 讀不到的檔是 panel
  的錯誤（Red 的事實加點名按鍵的提示），細節記進 App Log。Rename、Add、Zip 寫入時失敗用 input 的 `Commit`：input 留著、打的字還在，
  error popup 疊在上面；其他動作失敗開 error popup，標題寫「哪件事失敗」與對象（`App.ErrorAbout`：`DELETE FAILED: notes.md`）。
- **App Log** —— lib 的（`!`，footer 的 `!:log`）：每個 toast 與 error popup 自己進去。filu 另外記（`App.Log`）：使用者執行的每個動作
  與結果（tdp note.7：menu、熱鍵或 `Enter` 執行的，只看不改的也算 —— 進目錄與去到哪裡、Info 與預覽的 viewer、mark 與 pick、排序、
  隱藏檔、開關 tab、zoom、改名、新增、開檔、開始複製／搬移／打包、清掉 marks、刪掉 Tasks 的一列、開 shell 與 shell 結束；去某個目錄
  一律走 `goTo`）。會出 toast 的（Yank、Delete、Favorite）由 toast 記，不另外記；排序每選一步就是一次結果，每步記一筆。不記的：
  `Esc` 回上一層（不是 menu 裡的動作，`up` 直接 `cd`）、`h|l` 換 tab（跟換 focus 一樣）、移 cursor、Space menu、`?`、`!`。背景做的事（搬舊設定的每一項、第一次寫的設定範本），
  讀不到外面的東西的細節（目錄、預覽，同一件事只記一次：`tab.logged`、`previewBody.logged`）。
- **marks** —— 一個跨目錄、跨 tab 的 bucket（`marks.go` 的 `marksModel`），存在 `state.yaml`。清單上的勾是 lib 的標記欄，換目錄、
  bucket 變動時照 bucket 重勾（`List.SetMarked`）。`[3]` Marks 清單自己的標記就是 pick：有 pick 時只有 pick 的會落地（`landItems`）。
  複製保留 bucket；搬移把搬走的移出 bucket。
- **Zip** —— 打包「會落地的那些」到 `os.MkdirTemp` 的暫存目錄（input 的 `Commit` 建，建不了 input 留著），成為 bucket 裡唯一的 pick，
  再走既有的 `c` / `v`。名字的建議值是 `TextInput.Suggest`（`suggestZipName`：一項用它的名字、同一個目錄的用目錄名、其他用時間）。
  每個 pick 以自己的名字放進壓縮檔（目錄保留內部結構），同名的兩個都留。名字過 `zipFileName`（取 basename + 補 `.zip`）。
- **Tasks** —— 同磁碟搬移是瞬間 `rename`；跨磁碟、複製才逐項回報。log 是帶時間的白話（`taskText`）；中斷的任務存進 `data.yaml`、
  下次啟動標成 interrupted。執行中那一列是 lib 的 loading icon（`LoadingFrame`）加 `3/10`：`tasksBody` 畫的時候重做這幾列，
  `App.Spinning` 在 Tasks 開著、有工作在跑時讓 lib 繼續送畫面。做完是 Success toast，失敗是 Error toast：字是那一列的字、不帶那一列
  的 glyph（toast 的標題已經說了結果；App Log 那一筆也才是大寫開頭，tdp v0.9.2 note-78）。只開一個 toast：每個 toast 都記進
  App Log，先 Success 再 Error 就會多一筆 OK。以前還會卡住：lib 的動畫只有一條 tick（`App.animate`），第二個 toast 拿不到 tick、第一個
  的 command 又被丟掉，之後的 popup 全部停在一條線、不收按鍵；terminu-go v0.4.4 起 lib 自己送 tick 與 toast 的計時，不會再卡。
- **Finder** —— lib 的 `terminu.Finder`，三個都用 `Files: true`（lib 的檔案預覽與四列資訊）與 `Finder.Search`（filu 自己搜、自己篩）：
  - 檔名搜尋與 Goto（`pathFinder`）：`fd` 把 tab 的目錄（Goto 是家目錄）下的東西串流進來（`streamFinder`，走訪序、不排序；沒有
    `fd` 時退回 Go 的 walk），到 `finder_cap` 為止，用 `terminu.Matches` 照打的字篩（包含、不分大小寫、每個詞都要出現、順序不變，
    跟 lib 自己篩的一樣）。打的字以 `/` 或 `~/` 開頭就是路徑：從它寫到的最深的那個目錄往下三層重新列（`absAnchor`；以 `/` 結尾只列
    那一層），剩下的字拿來篩，列出來的寫完整路徑 —— 可以出 tab 的目錄、出家目錄。Goto 一律掃隱藏目錄，噪音靠 `ignore_dirs` 擋。
  - 內容搜尋（`contentSearch`）：邊打邊找，每次停 150ms 才跑 `rg`（舊的那次取消），一個檔一列、`FinderItem.Line` 是第一個符合的行，
    lib 的預覽從那一行附近開始；什麼都沒打時列出全部，跟檔名搜尋一樣。
  - 選了就進去：`[1]` 到那個目錄、cursor 停在那一項，開它的 menu 一起收掉。
- **預覽** —— 讀 magic bytes 判型別：目錄 → 三層的樹、壓縮包 → 內容清單、圖片 → base64 `data:` URI、PDF → 抽出的文字與頁數、
  文字 → Chroma（catppuccin-mocha）上色 + 行號、二進位 → hex + ASCII（`preview.go`）。文字照檔案原本的樣子交給 lib 的 `Pager`
  （`textLines` 只拿掉 CRLF 的 `\r`）：`Pager` 是「拿來讀的內容」，Tab 畫成 `input_tab_width` 格、檔案自己的顏色碼上色、一行裡的
  `\r` 只留最後的樣子、其他控制字元畫成紅色的名字（components/README「控制字元」）。樹與壓縮檔清單裡的檔名用 `terminu.Escape`
  畫成名字（不然名字裡的顏色碼會上色）。行依寬度產生（圖片的 data URI 照寬度折），所以在 `previewBody.Draw` 時才做。不用 lib 的
  `FilePreview`，見「設計決定」。
- **檢視與複製** —— `[2]` 的 `Enter` 用 lib 的 `Viewer` 打開預覽。文字檔交出去的是檔案原本的文字（最多 4 MB）：Tab、控制字元、
  CRLF、太長的行都由 lib 畫（`Lang` 是檔名，跟預覽一樣上色），複製出來是原樣。選取模式是 lib 的
  （`v` 兩次、vim 的移動、`y` 複製）；`y` 在選取以外複製全部。圖片的 data URI 合回一行再交出去，lib 自己折行，複製出來不會斷。
  剪貼簿走 OSC 52（`clipboard.go`），穿得過 tmux 與 SSH。
- **檔案資訊** —— 檔案上的 `Enter` 用 `Viewer` 顯示 `fileFacts`（`facts.go`）：完整路徑（symlink 多一列目標）、類型（跟預覽同一套判法）、
  大小（人話 + bytes）、Modified / Accessed / Created（Linux 是 Changed）、`rwx` + 八進位、owner:group；讀不到的欄位寫原因。
- **刪除** —— 移到 OS 垃圾桶：macOS `~/.Trash`、Linux `~/.local/share/Trash/files`（`trash.go`）。
- **tab 的標記** —— 第一個 tab 是啟動的 glyph（它永遠開在啟動的目錄），其餘各一隻動物（`look.go` 的 `tabMarks`）。標記是 panel 的
  tab 名稱（lib 的膠囊串），固定寬，不跟目錄名綁動（tdp L2）。上限五個（`maxTabs`）：到上限時 New tab 停用。
- **zoom** —— `z` 切換 `App.zoomed`，版面由 `App.layout` 照它與 focus 的 panel 給（lib 每畫一次就問一次 `Screen.Layout`）：`[2]`、
  `[3]` 佔滿畫面，`[1]` 佔滿上面三分之二、`[3]` 留在下面；不畫的 panel 給空的 `Rect`（lib 不畫小於 2×2 的 panel）。不用 lib 的
  `ForceNarrow`：它一律只畫 focus 的 panel。`[1]` 在 zoom 時把每個 tab 畫成等寬並排的欄（`drawColumns`，
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
  第一次啟動時裡面的東西一項一項搬到新位置、Info toast 說從哪搬到哪（`moveOldSettings`，tdp E3）：新位置沒有的搬過去；新位置已經有的
  兩邊都不動，開 error popup 寫出兩個路徑讓使用者自己合；搬空的舊目錄刪掉。有一項搬不動就把搬過去的搬回來、照舊用舊的，error popup
  寫出兩個路徑。最愛與 task 紀錄放 `~/.filu/data.yaml`（`FILU__DATA`），以前在 `state.yaml` 的第一次啟動時搬過去（`loadState`）；
  `data.yaml` 已經有了的話，`state.yaml` 裡的留著、每次存檔照樣寫回去（`leftInState`），一樣開 error popup。`config.yaml` 只收單一的
  選項：filu 自己的 `finder_cap`，與家族共用的 `input_tab_width`、`hide_secrets_in_preview`（交給 lib）。不再用的檔（tdp v0.9.3
  E3-23，lib 的 `App.NoLongerUsed`）現在沒有：filu 自己的檔一直是 `config.yaml`、`state.yaml`、`data.yaml`、`open_with.yaml`、
  `ignore_dirs.yaml` 這五個，都還在用；舊的 macOS 目錄是換位置，照上面的搬家做。哪天不再用其中一個，就在啟動時叫 `NoLongerUsed`。
- **使用者寫的清單** —— `open_with`、`ignore_dirs` 是資料（tdp v0.9.1 E3，FL26）：各一個檔，放在 `data.yaml` 旁邊，檔名就是以前的 key
  （`lists.go` 的 `userLists`）。沒有檔時寫一份帶註解的預設值（`loadLists`），之後只讀不寫。以前的版本放在 `config.yaml`：啟動時
  `moveLists` 用 yaml.v3 的 `Node` 讀 `config.yaml`，一項一項搬 —— 那個 key 的值連同它上面的註解寫成新檔，從 `config.yaml` 拿掉，再把
  filu 範本裡講那一項的段落（`# open_with:` 到下一個 `#` 行）從剩下的註解拿掉，其餘原樣寫回；排在第一個的 key 上面掛的是整份檔的開頭
  註解，留在原檔。新檔已經在的那一項留在 `config.yaml`、filu 用新檔，開 error popup 寫兩個路徑；`config.yaml` 寫不回去就把寫出的新檔
  刪掉、照舊從 `config.yaml` 讀（所以 `fileConfig` 還認得這兩個 key），error popup 寫兩個路徑。Info toast 說從哪搬到哪，App Log 每項一行。
- **檔名裡的控制字元** —— 檔名可能含控制字元（macOS 的 `Icon\r`）。filu 原樣交給 lib：清單的格子、位置列、menu、標題、toast、
  confirm、finder 的列是「名字與短字」，lib 把每個控制字元畫成紅色的名字（`\r`、`\n`、`\x1b`），不丟、不換成空白（components/README
  「控制字元」）。filu 要上色的格子給 `Row.Styled`（`[]terminu.Span`，字照樣留在 `Cells` 讓篩選與排序用；`look.go` 的 `colorPerm`、
  `colorSize`、`colorOwner`），不再把顏色碼塞進字裡。拿來讀的內容裡（`[2]`、檔案資訊的 viewer）的名字用 `terminu.Escape`。以前的
  `safeName` 把它們丟掉，已經拿掉。使用者打的值由 lib 的 input 處理：控制字元畫成紅色的名字、送出時擋下。
- **Shell** —— 互動 terminal（`Terminal.Interactive`）：`PgUp` 進歷史模式、`Alt-v` 進選取模式，選取的複製走 filu 的剪貼簿
  （`copyToClipboardCmd`，跟 viewer 一樣）。
- **名稱的檢查** —— Rename、Add、Zip 的 `Check`（`actions.go` 的 `renameProblem`、`addProblem`）：空的、撞名、Rename 含 `/`、Add 在
  目前目錄外面（`filepath.Clean` 後以 `/` 開頭、是 `..` 或以 `../` 開頭）、Zip 取不出檔名，都留在框裡、寫在錯誤列。寫入時才失敗的
  （權限等）在 `Commit` 裡失敗：error popup 疊在框上，關掉回到框、打的字還在。

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
  cursor 停在那個檔；tab 滿了跳 toast，不擠掉別的 tab）；Tasks 讓 `[1]` 到任務落地的地方。focus 跟著到 `[1]`（`App.Focus`），
  Favorites 的 `o` 選了 tab 也是：使用者按它就是要去那裡，預期 focus 會換（tdp K3 v0.8.0：focus 只在使用者預期時換；S19）。
- **會改變磁碟、或把控制權交出去的動作先 confirm**：`D` Delete、`D` Unfavorite、`o` Open、`s` Shell、shell 裡的 `Alt-Esc`、`C` Clear。
  `m` mark、`p` pick、`f` favorite 是可逆的開關，不問。`O` 的選單裡挑一個 app 就算確認過（tdp F6 允許），連 `Default` 也不另外問。
- **Zip 打到 temp，再走既有的落地路徑。** 使用情境是「打包完再搬去某個目錄」，輸出位置不等於目的地；打包範圍就是 `landItems()`。
- **`/` 是一個選單、不是兩個熱鍵。** `/` → {File names, Contents}；`f` 因此空出來給 Favorite。
- **tab 的標籤是一個 glyph、不是目錄名。** 目錄名長度不定，會讓膠囊串跟內容綁動（tdp L2）；目錄由路徑列說。
- **`maxTabs = 5`**：讓 tab 的膠囊串與 zoom 的欄數有上界。
- **面板數是成本。** v0.2.0 從 5 個收成 3 個，能收進既有 panel 的就不開新的。
- **平台：只支援 macOS / Linux。** 平台分岔的操作（metadata、trash、open）用 build tag，沒有 Windows 實作；`[s]hell` 的 PTY、路徑、
  cd-on-quit 的 POSIX shell function、`rwx` 權限都沒有 Windows 對應。Windows 使用者走 WSL（2026-07-27 拍板）。

- **zoom 時換 panel 照樣放大**（user 2026-10-19 定案）：`Tab` 換到下一個 panel 時，下一個也是放大的，
  `z` 才還原。放大是使用者要的版面，換 panel 不代表不要了。
- **zoom `[1]` 時 `[3]` 留著**（user 2026-10-24 定：tdp screen「Zoom」讓 app 決定怎麼放大）：`[1]` 佔掉 `[2]` 的位置、`[3]` 照舊在下面，
  marks、tasks、最愛一直看得到；`[2]`、`[3]` 照 tdp 的預設佔滿畫面。
- **zoom 時 `[1]` 的 tab 並排**（user 2026-10-19 定案：tdp 的 zoom 給預設，怎麼放大由 app 決定）：filu 在 zoom 時把每個 tab 畫成一欄
  （`files.go` 的 `drawColumns`）：zoom 本來就是要更多空間，在好幾個目錄之間比對、搬檔時，一次看到全部的 tab 比一頁一頁切好用；`h|l`
  照樣換 tab，有 focus 的那一欄亮、其他的淡化（tdp P6），欄間的線用 Overlay0（跟 menu 的分隔線一樣）。
- **長時間工作的進度寫在 Tasks 的列裡**（user 2026-10-19 定案）：filu 是單畫面、沒有 statusbar，進度是 app 的內容（`3/10`，執行中的 glyph
  Green）；開著 shell 時看不到，接受。
- **設定、狀態、資料分開**（tdp E3）：用「刪掉會怎樣」分。tab、marks、每個目錄的排序刪了只是回到預設，是狀態（`state.yaml`，在設定目錄）；
  最愛與 task 紀錄刪了找不回來、會想備份，是資料（`~/.filu/data.yaml`）；使用者寫的 `open_with`、`ignore_dirs` 也是，就算內容是 filu
  怎麼運作（tdp v0.9.1，user 定，FL26）。設定只剩 filu 自己一個一個的選項。
- **最愛的星在 `[1]` 清單的最後一欄**（user 2026-10-20 定，FL25）：位置列只寫你在哪裡（tdp v0.9.1）；星放在哪是 filu 自己的設計。放在
  名稱後面時，名稱一截短星就不見；一欄自己的、不收，窄的時候也看得到。
- **離開**（user 2026-10-19 定案，FL1）：用 `filu shell install` 裝的 function 跑時（有 `FILU__LAST_DIR_FILE`），`q` 是選 shell 停在哪裡的 menu：
  啟動的目錄、再來每個 tab 的（重複的只列一次），選了就離開；有工作在跑時選了先 confirm（`Esc` 回到 menu）。沒有這個變數時選了也
  沒有差別，所以只是一個 confirm（有工作在跑時多一句 Peach 的警告）。**只有一個 tab 時 `q` 不問、直接離開**（user 2026-10-24），
  shell 停在那個 tab 的目錄；有工作在跑時照樣先問（離開會中斷它，救不回來）。
- **檔案資訊的 viewer 不畫行號**（user 2026-10-19 定案，tdp S66）：是一張事實的表，不是檔案的內容；照樣能選取複製。
- **排序選了方向只關方向那一步**（tdp T1：選一個值寫回 picker，保留 source）：欄位 menu 留著、寫出新的排序，可以接著疊下一欄；
  `Esc` 收起。
- **最低尺寸 60×24**（tdp L1：app 可以調低）：80×24 是大多數終端機打開時的大小，在 filu 的版面裡每件事都做得完（上面兩個 panel
  16 列、`[3]` 8 列）；寬度到 60 欄也一樣：三個 panel 照樣並排，finder、menu、confirm 的列都放得下。比它小就是 lib 的
  「只畫名稱與要多大」，`screen_test.go` 量到 1×1。
- **`[2]` 用 filu 自己的預覽、不用 lib 的 `FilePreview`**：`FilePreview` 不收鍵、不捲，`[2]` 要能捲、`Enter` 打開檢視、`y` 複製，
  圖片的 data URI 也要照寬度折，所以 `[2]` 是 `Pager` 加 filu 的預覽。
- **Tasks 窄的時候先截任務、再收 `WHEN`**（`Column.Shrink`、`Drop`）：任務的白話先截到最少寬度，還放不下時間整欄走；白話比時間重要。

### 暫定（自己定的，等 user 確認）

- **`[3]` 的欄位**：Marks、Favorites 是 `NAME` 與 `IN`（所在目錄，從前面截）；Tasks 是 `WHEN` 與 `TASK`。理由：名稱與所在目錄分兩欄，
  名稱不會被長路徑擠掉，`IN` 從前面截、留下最近的幾層；Tasks 照時間讀，所以時間放前面。
- **`[3]` Favorites 的星照舊在名稱前面**（FL25 讓 filu 自己判斷）：那個清單每一列都是最愛，星是那一列的 icon，跟 Marks 分頁在名稱前面
  畫檔案類型的 icon 同一個位置；`[1]` 的星是「一堆項目裡哪幾個加了星」的旗標，才放到最後一欄。
- **兩份清單一份一個檔、檔名照以前的 key**（FL26 的做法）：`open_with.yaml`、`ignore_dirs.yaml` 各自是使用者手寫的清單，彼此無關；
  `data.yaml` 是 filu 每次存檔都重寫的，手寫的東西放進去，註解會被洗掉、filu 開著時改的也會被蓋掉。
- **讀不了的 `ignore_dirs.yaml` 移開後，新的一份是預設的那一串、不是空的**：tdp E3 說資料從空的開始，但空的 `ignore_dirs` 是「什麼都
  不略過」，Goto 會掃進 `node_modules`、`.git`；新的一份跟第一次啟動寫的一樣，比較像「一份新的」。

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
- **未做**：Mouse；每列的錯誤標示（broken symlink、沒有權限）；zoxide 式的目錄索引；chmod、解壓縮、真的圖（kitty / sixel）、續傳。
- **拿掉的**：`FILU__REPAINT`（每次導覽整頁重畫，給把 glyph 畫得比游標前進更寬的終端機用）：畫到終端機是 lib 的事。

## 偏離 tdp

目前沒有；還沒符合 tdp 的地方也沒有（`docs/filu-terminu-fix.md` 已經刪掉，最後一件 FL27 —— hint 的 `h|l:tab`，user 2026-10-24
要的，tdp v0.9.14 定成全家族由 lib 加 —— 在 terminu-go v0.4.17 做好）。
zoom 時 tab 並排以前寫在這裡，tdp v0.8.0 讓 app 決定怎麼放大，移到「設計決定」。`[1]` 清單最後一欄的星、`[3]` 各清單的欄位是 panel 的內容。

## 設計文件導讀

filu 沒有另外的設計文件；功能與決定的原始清單在 `.forge/meta/IDEA.md`（本機、不進版控，內容停在 v0.2.0 以前）。
tdp 在 [terminu](https://github.com/vulcanshen/terminu)（`principle/`、`components/`、`project/`、`checklist/`），lib 在
[terminu-go](https://github.com/vulcanshen/terminu-go)。

| 檔案 | 內容 |
|---|---|
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
  `make build`、`make dev`（build 完直接跑 `./filu`，讀真正的設定與資料）、`make install` / `make uninstall`、`make package`、
  `make check`（commit 前跑：fmt-check、vet、`go test -race`；有 `go.work` 時加 `check-lib`）。跑 `make` 列出全部；filu 自己的 target 放在最後一段 `##@ filu 自己的`（目前沒有）。
- **改 lib 時**：`go.mod` 寫發布的 terminu-go 版本（不用 `replace`）；本機用 gitignore 掉的 `go.work`、`go.work.sum`（`go work init . ../terminu-go`）接上
  lib 的工作目錄。`make check` 的 `check-lib` 關掉 `go.work`、照 `go.mod` 的版本編譯與測試：過不了就是用到了 lib 還沒發版的東西。
- 本機 build 的版本號是 `dev`；release 版號由 goreleaser 以 `-X` 注入 `internal/version.Version`。
- binary 的子指令：`filu version`（也收 `--version`、`-v`）、`filu help`（也收 `--help`、`-h`）、`filu shell install` / `uninstall`
  （cd-on-quit，見「運作方式」）、`filu iconwidth`（印出 lib 探測到的 icon 與 powerline 符號格數，以及從哪裡來的）。
- 頂層另有（tdp project 的目錄圖上沒有的）：`CLAUDE.md`（gitignore，給 AI 協作的指引）、`.claude/`（進版控：專案的 Claude Code
  規則、agent 與指令）、`.forge/`（gitignore：forge 的 GOAL.md、PROGRESS.md、IDEA.md、REDESIGN-LIST-AND-NAV.md，內容停在 v0.2.0
  前後）、`.checkpoints/`（自帶 `.gitignore`：AI 協作的 session 紀錄）。

測試是 programmatic 的（不靠真終端）：`harness_test.go` 照 terminu-go 的 app 的做法，把按鍵送進 `App.Update`、跑完它發出的
command（動畫、背景的 channel）、讀 `Frame()`。session、設定、垃圾桶、剪貼簿都指到測試自己的暫存目錄。

- **tdp L4、E5 與 L1**：`screen_test.go` 在幾種尺寸（到最低尺寸 60×24 為止，再小一點）、「icon 一格 / 兩格、powerline 符號一格 / 兩格」
  下，每一類 popup 各開一次，量每一列；再把畫面縮到比 popup 小（最低尺寸、比畫面寬、比畫面高、兩邊都大，再到最低尺寸以下、1×1）。
  最低尺寸以下只畫名稱與要多大、鍵沒有作用、`Ctrl-C` 兩次才離開，拉大後畫面跟原本一樣（`TestL1UnderTheLeastSize`）。popup 的開關動畫
  是真的 32ms 一格，所以這組測試要幾十秒。
- **背景的 channel**：harness 執行讀 channel 的 command（`waitFind` 之類）最多等 150ms；等不到時那個讀還在，之後讀到的訊息放進
  `harness.late` 再交給 filu，不會掉。搜尋是邊打邊找的，少了這一步，打字後的第一批結果會被上一次留下來的讀拿走。
- **背景讀的預覽**：lib 的 finder 預覽在畫的時候記下要讀什麼、下一次 update 才開始讀（tdp file-picker 的「在背景讀」）。harness 的
  `has`、`hasNot` 先 `settle`（畫一次、送一次 update，三回），像終端機接下來的幾格畫面，讀完的預覽才會在畫面上。
- **tdptest**（terminu-go 的）：`tdp_test.go` 的 `TestFirstLaunch` 用 `tdptest.Check` 在很多種尺寸與 icon 寬度下跑第一次啟動（每次一個新的
  沙盒當 HOME），每個 panel 開 Space menu、global operation、App Log 與 `?`，再縮到最低尺寸以下又拉回來；每一格畫面都驗 tdp L4、有沒有
  控制字元送到終端機、有沒有顏色碼被畫成字。`go test -short` 只跑最低尺寸與 80×24。`TestFirstLaunchOnTrickyNames` 是同一個 Check，
  家目錄裡放 `tdptest.Tricky` 命名的檔與目錄（`Tricky: true`：外面來的名字本來就可能含顏色碼，畫成名字是對的）；`TestTrickyNames` 用
  `tdptest.Start` 讓 cursor 一個一個走過那些名字、看資訊、mark、搜尋，那是 Check 不會走到的地方。
- **顏色碼塞錯地方**：塞進格子的顏色碼不會送到終端機（lib 把它畫成名字）；`tdptest.Check` 抓得到被畫成字的顏色碼，`screen_test.go`
  的 `checkWidths` 也在 harness 開的每一類 popup 上看畫面有沒有 `\x1b` 這幾個字（fixture 的檔名沒有 ESC，出現了就是 filu 自己的顏色碼
  放錯地方）。
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

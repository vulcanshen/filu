# filu — terminu fix

filu 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.6/principle)（tdp v0.1.6）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與 `dev-remarks.md` 裡描述該行為的段落。有意不修的，改寫成
`dev-remarks.md`「偏離 tdp」的一條並附理由。

盤點日期：2026-09-27（對照 tdp v0.1.6，逐條讀 `rules-zh_TW.md` 全文重新核對，不只看 CHANGELOG）。行號以當天的
`main`（`bff62ca`）為準；`terminu-fix` 分支目前只有文件改動，程式碼跟 `main` 相同。


## 先看：locku、webu、sshu 修完的經驗（2026-09-27 更新，含 webu 第二、三輪與 locku 對照 v0.1.4–v0.1.5）

locku 照 tdp v0.1.0 修完（v0.1.2、v0.1.3），再對照 v0.1.4 修完，v0.1.5、v0.1.6 不用修；webu 照 v0.1.0 修完（v0.4.0），
第二輪對齊 v0.1.5、第三輪對齊 v0.1.6；sshu 照 v0.1.1 修完，再對照 v0.1.3–v0.1.6 收尾。第二輪以後都只在本機 commit、
尚未發版。修的時候發現這些，這裡的各條也適用：

**動手前**

- **（locku、webu）先把本檔定案、commit，再改程式。** 本檔由 terminu session 預先寫好，放在工作樹裡。動手前
  `git status` 看到它、讀完，把 user 對「待確認」的裁定寫回本檔（「待確認」改成「已定案」）並 commit；之後才改程式，
  程式的 commit 跟本檔分開。
- **（webu）動手前先把 fix 清單互相對一遍。** 清單裡的條目可能互相衝突。
- **（webu 第二輪）對照新版 tdp 時，逐條讀 `rules.md` 全文，不只看 CHANGELOG。** webu 有三處是規則沒改、app 早就不符合、
  上一輪沒抓到的。filu 這次也是：第 15 條（yank viewport 的選取是 K11 的模式）的規則從 v0.1.1 就在。
- **（webu 第二輪）條文讀不通時，可能是 tdp 的缺口，不是 app 的違反。** 寫下來回報 terminu（webu 多行輸入的 `Tab`，
  tdp v0.1.5 補上），app 不改程式、也不寫成偏離。
- **（sshu）清單的「現況」要拿程式碼核對過才算數。** 照抄別的 app 的清單會修一半。
- **（webu、sshu）行號很快就過期。** 前幾條一改，後面引用的行號全部位移；換註解時用內容比對，不要照行號。
- **刻意保留的行為寫成偏離**：寫進 `dev-remarks.md`「偏離 tdp」並附理由，而不是留在本檔。跟既有 UX 衝突的規則，
  先問它的 origin UX 在 filu 還存不存在（sshu）。

**疊層與路由**

- **（sshu、locku、webu）離開流程用自己的 popup，它的 `?` 也另開一個。** `Ctrl-C` 在任何地方都叫得出離開流程（K8：
  打字中也是），所以它可能疊在另一個 confirm 上；借用同一個會蓋掉使用者正在回答的問題。filu 的 `quitMenu` 已經是
  自己的一個 instance；它的 `?` 另開一個（locku、webu 叫 `quitHelp`）：help 可能在 quit picker 底下（在 help 上按 `q`），
  也可能在它上面（在 quit picker 上按 `?`），共用同一個就得記「誰先開」。
- **「最上面」只能有一個答案，「放在最上層」要改三個地方：按鍵路由、`Esc` 的處理（`closeTop`）、繪製順序**（D3）。
  順序固定成 `quitHelp` > `quitMenu` > `help` > 其他，三處照同一個順序各寫一次。少改一個就是一種 bug：路由錯了 `Enter`
  會確認底下看不見的框、打的字進底下的輸入框，`Esc` 錯了關錯層，繪製錯了畫面上看不到（locku 修掉的三個 bug 都是這種）。
  filu 目前沒有 `closeTop`，每個 popup 在自己的 `update()` 裡處理 `Esc`。
- **（locku v0.1.4）「有框握著鍵盤」的判斷要把離開的框算進去。** 執行 menu 的一列後，locku 用 `boxUp()` 決定留住整疊
  還是清掉。global operation popup 的 `[q]uit` 會開出 quit picker；判斷若不認 `quitMenu`，底下的 global operation popup
  與 Space menu 會被清掉，`Esc` 就回不去（第 6、9 條）。
- **（webu）F4 可以統一在按鍵路由處理**：記下按鍵前最上層的等級，按鍵後若不是 `Esc`、最上層掉了一級以上，就關掉
  底下的 menu。執行 menu 的一列時，在 **dispatch 回傳的 model** 上判斷有沒有開出新框，不要在舊 model 上關（會關在
  沒人回傳的副本上；locku 也踩過 value receiver 的同類陷阱）。
- **F3 不只看 toast**：判斷「popup 還在不在」用開啟中或已開（`owns()` 一類），不用含關閉中的 `isActive()`。
  locku 的 bug 在 toast，sshu 照抄只寫了 toast，實際上每一個 popup 都用 `isActive()`；filu 也是（見第 8 條）。
- **K9 的離開流程**：離開流程開著時，`Ctrl-C` 直接離開；`q` 在離開流程上不再疊一個。
- **S3**：splash 的判斷放在 `Ctrl-C` 之前，任何鍵都只關 splash（filu 目前已是如此，`app.go` :265）。
- **層數會變多**：Space menu → global operation popup → quit picker → 它的 `?`，或 Space menu → Delete confirm → `?` →
  quit picker → 它的 `?`（locku 最多五層）。D2 的層色只有四階，第四層起同色。
- **（locku v0.1.4）新 popup 要一個不漏地接上**：`AnimTickMsg` 的 batch（`app.go` :259）、`WindowSizeMsg` 的 `setSize`
  （:240–:253）、層數 / 層色、`view.go` 的繪製清單（:55–:98），以及測試裡推動畫的地方（例：`gchord_test.go` 以
  `AnimTickMsg{Target: …}` 推）。漏了一個，那個 popup 在測試裡就停在動畫中途、接不到鍵。

**menu 與 key reference**

- **（sshu）global operation 用一份清單**：一份 `globalActions` 是 global operation popup 的唯一來源；所有 panel 的
  Space menu 走同一個組法（item 區、panel 區，最後接 global 那一列）。
- **（locku v0.1.4）沒有熱鍵的列 commit 一個按不出來的 key**，在 Space menu 的分支裡攔下來開 popup，不進 dispatch。
  filu 要注意：`bracketHotkey()`（`spacemenu.go` :201）會把**出現在 label 裡的多字元 key** 原地括起來（`[go]to`），
  所以 `Global operation` 列的 key 不能是 label 的子字串（例如不能用 `"global operation"`，否則畫成 `[Global operation]`）。
- **（sshu、webu、locku）key reference 從 Space menu 讀**：panel 的 key reference 由它的 Space menu 列產生，只收按得出來的
  鍵（單一字元、`go` 和絃、label 已寫出的 `[/]`、`[.]`），menu-only 的列（`Global operation`）不列，空的區塊標題拿掉；
  label 若是 `[Enter] …` 的寫法，先去掉 `[Enter] ` 再放進說明，panel 自己列了 `Enter` 時，core key 那段不再重複通用的
  `Enter`。再接 core key，兩邊不會不一致。
- **（locku v0.1.4）D4 依內容算寬時多留一欄**，否則最長那一行貼著右框；也不要比下框 hint 窄。
- **（webu 第三輪）框的寬度在打開時定一次，不跟著狀態變**（L2）：下框 hint 隨狀態變（有沒有錯誤、在不在選取中）、
  值變長，框都不能跟著縮放；值更長就捲動、尾端在畫面上。第 10、15 條都會碰到。
- **（webu 第三輪）hint 的用語說「這個鍵做什麼」，一個動作在 hint 上只露一個鍵。**
- **（webu）label 已寫出鍵的列不要再括一次**（D4）：`[/] Search` 不能變成 `[/] [/] Search`。
- **（webu）每個 panel 的 Space menu 都接上 global 區之後，原本只有一區的 menu 要補上區塊標題**（M2）。
- **（webu 第二輪、sshu）模式的 help 與按鍵清單讀同一張鍵表**，按鍵清單只用方向鍵移動（K11，模式的鍵跟導覽鍵重疊時）。
  filu 的 yank viewport 選取就是這種模式（第 15 條）。
- **（webu）改完一條就回頭檢查其他畫面的提示。** `Space` 不再關 popup 之後，下框的 `Space close` 要跟著改。

**輸入**

- **（sshu）K3 改成一律送出以後，驗證要把「有沒有填」一起問**：以前「沒填就不會送出」替驗證擋掉了必填檢查；
  `Enter` 一定送出之後，空白要由驗證本身抓到（第 10 條）。
- **（webu 第二輪）將來要做 input group 時，沿用現有的 input popup**：原本的欄位當第 0 欄，多一個其餘欄位的清單與
  目前欄位，單欄的框不用改。filu 目前沒有 input group。
- **（sshu、locku）灰字不一定是提議，「單一輸入框還是 input group」看同一個框裡有幾個欄位。** filu 的 Rename、Zip
  預填的是值，不是灰字提議（見「已經符合」）。

**測試與驗證**

- **每修一條補一個 model test**，測試名稱或註解寫明 tdp 條目（locku `internal/ui/app_test.go`）。
- **每一個修正做一次 mutation**：把修正單獨改回舊行為，確認對應的測試會紅。新測試用到新欄位時「拿舊程式碼跑一次」
  編譯不過，逐處 mutation 才量得到（sshu、locku）。三種假象：
  - **mutation 本身編譯不過**（留下沒用到的變數、不存在的常數）：`go test` 失敗但不是測試抓到；看是不是 `build failed`，
    改成能編譯的寫法（`if false && …`、`_ = x`）重跑（webu）。
  - **兩個機制做同一件事，互相掩護**：改掉任一個測試都不紅；只留一個（webu）。
  - **「已經在問」一類的判斷**（quit picker 上再按 `q` 不疊第二個）：拿掉後重開一次、內容一樣，測試看不出；要斷言
    「沒有重播開啟動畫」（`isInteractive()`）才抓得到（webu）。
  - 另外，mutation 沒被抓到先看是不是量錯了東西（用被改的函式判斷結果、改在走不到的分支、情境剛好被另一條驗證擋住，
    sshu）；唯一的全域動作是離開時，T1「整疊清掉」只能在離開的那一拍量（locku）。
- **（sshu）守舊規則的測試要改寫，不是刪掉**：改名、反轉斷言，測試名稱寫出新規則。filu 的 `helppopup_test.go`、
  `spacemenu_test.go`、`quit_test.go` 等守著現在的行為，改規則時會一起紅。
- **（locku v0.1.4、webu 第三輪）改完要把畫面實際印出來看**：在套件裡暫時寫一個測試 `t.Log(ansi.Strip(m.View()))`，
  看完就刪。疊層順序、貼框、hint 被截掉只有這樣看得到；實機看到的寬度問題，照同一個畫面寬 render 出來量，測試才有東西守。
- **（locku）`tea.Quit` 會被包在 `tea.Batch` 裡**：判斷「有沒有離開」要遞迴看進 `BatchMsg`。

**其他**

- **（webu 第二輪）shell 包 `zsh -c '…'` 時，patch 腳本與 commit message 寫進檔案再執行**（`git commit -F`）：
  內容裡的撇號（`mode's`）會截斷外層的單引號。
- **文件裡的 tdp 連結釘在 tdp 的版本 tag（目前 `v0.1.6`）**（本檔與 README、dev-remarks 已改好）。帶日期、記錄
  「當時對照哪一版」的句子不改；連結版本與「對照哪一版」的說法要一起動（sshu、locku）。
- **不發版**：user 2026-09-27 裁定，家族全部 app 與 tdp 都穩定下來再一起發。修完一批時 README 與 `dev-remarks.md`
  等描述行為的段落同一個 commit 改；將來發版、打 tag 前先 `git branch --show-current`（locku 的教訓）。
- 完整紀錄：terminu repo 的 `.local/family-fix/locku/`、`.local/family-fix/webu/`、`.local/family-fix/sshu/`（本機）；
  sshu 的本機 `main`（`terminu-fix` 已併回）可當 `?` key reference、global operation popup、離開的 popup、K3、K11 的實作範例。

---

## 1. `Ctrl-C` 在 panel 上直接結束、在 popup 裡被吞掉 —— K9、K8、D3

- **現況**：`internal/ui/app.go` `Update()` 的主 switch，`case "ctrl+c"`（:466）直接 `m.shutdown()`，
  不開 cd-on-quit picker、有任務在跑也直接走。這個 switch 只在沒有 popup 時才走得到；popup 開著時
  `Ctrl-C` 交給該 popup 的 `update()`，而 `spaceMenu.update()`（所有 picker 共用）、`confirmPopup.update()`、
  `helpPopup.update()`、`breadcrumbPopup.update()`、`detailYank.update()`、`searchModel.update()`（輸入態與
  清單態）、`inputPopup.update()` 都沒有處理它，按了沒反應。quit picker（`quitMenu`）開著時再按
  `Ctrl-C` 也一樣被吞。
- **規則**：`q` 與 `Ctrl-C` 做同一件事 —— 進入離開流程（filu 的離開流程就是 cd-on-quit picker）；
  `Ctrl-C` 在輸入態仍然有效；離開流程進行中再按一次 `Ctrl-C` 立刻離開。離開的框是自己的 popup，疊在整疊最上面（D3）。
- **怎麼改**：`Ctrl-C` 的處理提到 popup 路由之前（splash 之後、PTY 之後，K10）：quit picker 開著時 `m.shutdown()`，
  否則 `m.openQuitMenu()`，疊在開著的 popup 上（不關底下那疊；在 picker 上 `Esc` 回到原本的框）。quit picker 在
  最上層要三處一起改，順序 `quitHelp`（第 4 條）> `quitMenu` > `help` > 其他：按鍵路由（目前 `quitMenu` 排在 confirm、
  Space menu、sort / goto / openIn / search picker 之後，:422）、`Esc` 的處理、繪製順序（`view.go` 目前在 confirm、
  input、help 之前畫 `quitMenu`，:69）。README 兩份「按鍵一覽」的 `Ctrl+C  Quit now (stops any copy or move in progress)`
  改成「同 `q`；在離開畫面上再按一次立刻離開」之類的說法。

## 2. `q` 只在沒有 popup 時是離開；help 與 finder 裡另有意義 —— K1、K9、K2

- **現況**：`q` 在 `app.go` 主 switch（:468）才開 quit picker。popup 開著時：`helppopup.go` `update()`
  （:62）把 `q` 當成關閉 help；`search.go` `update()` 清單態（:299）的 `q` 是「回到輸入列」；其他 popup
  （Space menu 與各 picker、confirm、breadcrumb、yank viewport）按 `q` 沒反應。
- **規則**：`q` 是 core key，除了輸入態以外在每一個 surface 都是「進入離開流程」；letter hotkey 與
  popup 自己的鍵不能佔用它。quit picker 開著時再按 `q` 不疊第二個。
- **怎麼改**：`q` 跟第 1 條的 `Ctrl-C` 放在同一處（輸入態除外：`inputPopup`、finder 輸入態）。help 拿掉
  `q`；finder 清單態「回到輸入列」改用 `Tab`：輸入列與結果清單是 finder 裡的兩個同層物件，`Tab` 在兩者之間切換（K2），
  並更新 finder 下框 hint（`search.go` `hint()` :760 的 `q=input`）。finder 的輸入列沒有灰字提議，所以 v0.1.6 K2
  「單一輸入框的 `Tab` 接受提議」不適用，`Tab` 可以拿來換到結果清單。輸入列上的 `Enter` 維持現狀：它是輸入列這個
  單一欄位的 submit（K3，v0.1.3「submit 的對象可以是單一欄位，由 app 決定」），送出後 focus 到結果；沒有結果時清單
  已寫出 `(no matches)`（:673），不算送不出去卻不說。「quit picker 上再按 `q` 不疊第二個」的測試要斷言沒有重播
  開啟動畫（見「先看」的 mutation）。

## 3. `Space` 會關掉 Space menu 以外的 popup —— K5

- **現況**：
  - `spacemenu.go` `update()` 的 `case "esc", " "`（:132）關閉 menu。`spaceMenu` 同時被拿來當 sort
    picker、Goto / New tab picker、Search chooser、quit picker、Open with picker、Favorites 的 Open in
    picker，所以 `Space` 在這些 popup 上全都等於 `Esc`；下框 hint 一律寫 `j/k move   Space close`（:227）。
  - `confirm.go` `update()` 的 `case "esc", "n", " "`（:48）：`Space` 取消 confirm。
  - `breadcrumbpopup.go` `update()` 的 `case "esc", "b", " "`（:71）。
  - `helppopup.go` `update()` 的 `case "esc", "?", " ", "q"`（:62）。
  - 其他 popup 已經符合：yank viewport、finder 清單態不理 `Space`；input popup 與 finder 輸入態的 `Space` 是字元（K8）。
- **規則**：`Space` 只開關它自己開的 Space menu；其他 popup（由 `Enter` 或熱鍵打開的 confirm、menu、
  viewport、global operation popup、key reference……）上按 `Space` 不作用，它們由 `Esc` 或自己的流程關閉。
- **怎麼改**：`spaceMenu` 加一個旗標（例如 `spaceToggle`），只有真正的 Space menu（`newSpaceMenu()`）
  讓 `Space` 關閉，而且只在它是最上層（上面沒有疊別的框）時才關；其他 instance（包括第 6 條新增的 global operation
  popup）的 `" "` 不作用，hint 改成 `j/k move · Enter run · Esc close`（D4）。confirm、breadcrumb、help 拿掉 `" "`。
  breadcrumb 同時拿掉 `b`（開它的熱鍵兼關閉；其他熱鍵開的 popup 都只認 `Esc`，2026-09-28 user 裁定，見「已定案」第 3 題）。
  confirm 的下框 hint `enter/y confirm   esc cancel`（`confirm.go` :59）順手照 D3 改成 `Enter <動詞> · Esc cancel`
  （家族預設，不是違反）。

## 4. `?` 在 popup 上沒有反應 —— K6、M4、D3

- **現況**：`?` 只在 `app.go` 主 switch（:470）處理，打開全 app 的 help。Space menu 與各 picker、confirm、
  breadcrumb、yank viewport、finder 清單態開著時按 `?` 都沒有反應。confirm 的 `y` / `n`、Goto 收藏清單的
  `f`（取消收藏，只寫在標題 `Favorites · f unfavorite`，`goto.go` :64）、yank viewport 的 `v` / `y` / `0` / `$` 等
  popup 自己的鍵，只有下框 hint 或標題寫了一部分。
- **規則**：`?` 在任何 surface 都有回應；focus 在 popup 上時（Space menu 與 global operation popup 也是 popup），
  `?` 打開**這個 popup** 的 key reference：唯讀、可以捲動、沒有游標、不能執行，只列這個框裡能按的鍵。再按 `?`
  或 `Esc` 關掉，回到底下的 popup（F4）。
- **怎麼改**：每個 popup 給一份自己的 key reference（menu / picker：`j/k`、`g/G`、`Enter`、熱鍵或數字、`Esc`；
  confirm：`Enter` / `y` 接受、`Esc` / `n` 取消；breadcrumb：`j/k`、`Enter` 跳過去、`Esc`；yank viewport：
  移動鍵、`v`、`y`、`Esc`（選取中的 `?` 是模式的 help，見第 15 條）；finder 清單態：移動鍵、`Enter`、`Tab` 回輸入列、
  `Esc`；Goto 收藏清單加上 `f`；quit picker：數字、`Enter`、`Esc`、再按 `Ctrl-C` 立刻離開）。key reference 疊在該 popup 上，
  路由與繪製都在最上層（D3）：目前 `m.help.isActive()` 的路由排在 detailYank、finder、breadcrumb 之後（:303），`view.go`
  也在它們之前畫 help（:81），兩處都要改。quit picker 的 `?` 另開一個 help popup（`quitHelp`），排在 `quitMenu` 之上，
  見「先看」與第 1 條。finder 輸入態的 `?` 是字元（K8），不打開。新 popup 照「先看」的清單接上動畫、尺寸、繪製與測試。

## 5. panel 上的 `?` 是全 app 共用的一份，沒列這個 panel 的鍵 —— K6、M4、D4

- **現況**：`helppopup.go` 的 `helpPopup` 在三個 panel 上都顯示同一份 `helpRows`（:24–:40）：panels / move / do
  三組，只有 core key、導覽鍵與 `z`；panel `[1]` 的 `o`、`O`、`m`、`y`、`r`、`D`、`f`、`F`、`c`、`v`、`/`、`go`、
  `b`、`t`、`w`、`a`、`S`、`s`、`.`，panel `[2]` 的 `y`，panel `[3]` 的 `p`、`m`、`y`、`Z`、`C`、`D`、`o` 都不在上面。
  它把所有列一次畫完（`drawPopupBox`），沒有捲動。唯讀、不能執行這點已經符合。
- **規則**：panel 上的 `?` 打開這個 panel 的 key reference：至少列出這個 panel 能按的鍵與 core key；唯讀、可以捲動，
  沒有游標、不能執行，不是 menu。寬度依最長的說明計算（D4）。能執行的全域動作不在這裡，在 global operation popup
  （第 6 條）。
- **怎麼改**：key reference 由 focus panel 的 Space menu 列產生（`buildSpaceMenu()` 的 item / panel 區，只收按得出來
  的鍵：單一字元、`go` 和絃；`Global operation` 這種 menu-only 的列不列、空的區塊標題拿掉），再接 core key 與導覽鍵
  （`Tab`、`1 2 3`、`h l`、`j k`、`g G`、`u d`、`Enter`、`Esc`、`Space`、`?`、`q`），兩邊就不會不一致（sshu、webu、locku
  的做法）。加上捲動（`j/k` 或方向鍵捲、沒有游標）。寬度計算（:79–:85）已經依最長說明，保留，但多留一欄、不比下框
  hint 窄（locku）。README 兩份「開始使用」表格裡 `?` 的說明（`Help — every app-wide action in one list`）與「按鍵一覽」的
  `?  Help` 改成 key reference 的說法（例：`Keys — what you can press here`）。

## 6. Space menu 沒有 global operation 那一列，也沒有 global operation popup —— M2、M4、K9、F4

- **現況**：`app.go` `buildSpaceMenu()`（:801）經 `groupedMenu()`（:895）只組 `item operation` 與
  `panel operation` 兩區，只剩一區時不加標題（:895–:904）；離開 app 不在任何一個 panel 的 Space menu 裡，只能靠
  `q`（以及 help 裡的一行說明）。沒有 global operation popup。
- **規則**：panel 上的 Space menu 最後一區是 `global operation`，**固定一列** `Global operation`（全域動作只有一個
  時也一樣）；`Enter` 打開 global operation popup，疊在 Space menu 上，是一種 menu，列出全部全域動作，`j/k` 選、
  `Enter` 或熱鍵執行；離開 app 必須在這裡。`Esc` 回到 Space menu（F4）。panel 上的 Space menu 一律帶區塊標題。
  目前所在畫面的切換列照 M6 變暗（filu 只有一個畫面，目前不適用）。
- **怎麼改**：
  - 定義一份 `globalActions`（目前只有 `[q]uit`，說明例：`pick a dir to cd to, then leave`），是 global operation
    popup 的唯一來源（sshu 的做法）。global operation popup 用一個新的 `spaceMenu` instance（不是 Space menu，
    `Space` 不關它，見第 3 條），層色比 Space menu 深一層（D2）。filu 的其他動作都作用在某個 panel 或 cursor 上
    （切分頁、Goto、Search、Shell 都作用在 `[1]`），不進 global operation popup。
  - `groupedMenu()` 改成一律加標題，最後接上分隔線、`global operation` 標題與 `Global operation` 那一列；只剩一區時
    不加標題的分支拿掉。`Global operation` 列沒有熱鍵，commit 一個按不出來、也不是 label 子字串的 key（`bracketHotkey()`
    會把 label 裡找得到的多字元 key 括起來），在 Space menu 分支（:356–:367）攔下來開 popup，不進 `dispatchFocusKey()`。
  - global operation popup 的 `[q]uit` 打開 cd-on-quit picker，疊在 global operation popup 上（F4）：picker 上
    `Esc` 回到 global operation popup、再 `Esc` 回到 Space menu；選定目錄才離開。離開流程跟 `q` / `Ctrl-C` 是同一個
    （第 1 條），在 picker 上再按 `Ctrl-C` 立刻離開。執行這一列之後「有框握著鍵盤」的判斷要把 `quitMenu` 算進去，
    否則底下兩層會被清掉（locku 的 `boxUp()`）。選定目錄離開時，整疊在同一拍清掉（T1 只能在這一拍量）。
  - `app.go` :476 的 `if len(items) == 0 { return m, nil }` 拿掉（global 那一列永遠在，menu 不會空；M7）。
  - README 兩份「按鍵一覽」的 Space menu 表格補上 `Global operation` 與 global operation popup。

## 7. 暫時不能執行的列被藏起來、熱鍵還會 toast —— M6

- **現況**：
  - `buildSpaceMenu()` 在 `len(m.tabs) >= maxTabs` 時不列 `Tab`（:832）、在只有一個分頁時不列
    `Close tab`（:836）；作用對象（`[1]` 這個 panel、當前分頁）都存在，只是現在不能做。
  - 分頁已滿時按 `t`，`handleListKey()`（:537–542）跳 toast `Tab limit reached (5) — close one with w`。
  - `openin.go` `openOpenInMenu()`（:24）分頁已滿時不列 `New tab`。
- **規則**：對象存在、但現在不能執行：列照樣出現、變暗，說明欄維持原本那句，不另外寫原因；cursor 可以停
  在上面，`Enter` 與熱鍵都不作用。（對象不存在才不出現 —— marks bucket 是空的時候不列 `Copy`、
  `Move here`、`Zip`、`Clear` 是對的。）
- **怎麼改**：`menuItem` 加 `disabled`，render 變暗；`spaceMenu.update()` 在 disabled 列上的 `Enter` 與
  熱鍵都不回傳 key。`Tab`、`Close tab`、Open in 的 `New tab` 改成 disabled 而不是不列；`t` / `w` 在不能做
  時直接不作用，拿掉 `tabLimitToast()`。先確認畫面上別處看得出原因（分頁列已經畫出 5 個分頁）。README「瀏覽」一節不受影響。

## 8. toast 開著時 `Esc` 不關 toast；關到一半的 popup 吃掉按鍵 —— F3、K4

- **現況**：
  - `toastModel`（`toast.go`）只靠計時自己關；`app.go` `Update()` 的按鍵路由沒有檢查 toast。toast 開著時按 `Esc`，
    鍵照常送到 panel —— `handleListKey()` 的 `case "esc"`（:531）把分頁帶到上一層目錄；若底下還有 popup，關掉的是
    那個 popup。
  - 其他每一個 popup 的路由都是 `if m.X.isActive() { if !m.X.isInteractive() { return m, nil } … }`（:273–:448）：
    `popupAnimator.isActive()`（`animation.go` :47）連關閉動畫中的也算，所以正在關的 popup 把所有按鍵吞掉。popup
    還不能疊的時候看不出來；第 4、6、9 條讓 popup 疊起來以後，關到一半再按 `Esc`，這一下會被吞掉，而不是關底下那層。
- **規則**：任何看得到的 popup —— 包括會自動消失的 toast —— 按 `Esc` 都立即開始關閉；`Esc` 一次關一層，
  最上層先關。已經在跑關閉動畫的 popup 不再理會 `Esc`，也不再接收其他按鍵。
- **怎麼改**：`popupAnimator` 加 `owns()`（開啟中或已開，不含關閉中）。按鍵路由在 PTY 之後、其他 popup 之前加一段：
  `m.toast.owns()` 時 `Esc` 呼叫 toast 的關閉並 return。其他 popup 的路由判斷從 `isActive()` 改成 `owns()`，
  讓正在關的那層把鍵讓給底下（開啟動畫中仍照舊不收鍵）。整個路由一起看，不只 toast（sshu 的教訓）。

## 9. 從 popup 開出的 popup，取消後回不到原本的框 —— F4、K4、D2

- **現況**：
  - `app.go` Space menu 分支（:363–364）選定一列後 `tea.Batch(cmd, m.dispatchFocusKey(key), m.spaceMenu.close())`：
    Space menu 先關，再打開 confirm（Delete、Open、Shell、Unfavorite、Clear）、input（Rename、Add、Zip）或 picker
    （Open with、Sort、Search、Goto、Tab、Breadcrumb、Open in）。在這些 popup 上按 `Esc` 取消，回到的是 panel，
    不是 Space menu。
  - Search chooser 選了 filename / content 之後先關自己再開 finder（:414–:417）；Goto picker 選了 Search 也是先關
    （`advanceGotoFlow()`）。在 finder 上 `Esc` 回到 panel，不是 chooser / picker。
  - 所有 popup 的層色都是 `popupLayerColor(1)`；疊起來以後分不出層次。
- **規則**：從 popup A 開出 popup B 時，A 預設留在底下；取消 B 回到 A，`Esc` 只關最上層，底下的階層原樣呈現。
  完成 B 之後 A 要不要留，依 T1 判斷（短的 confirm / 訊息通常保留；完成動作清掉整疊，D3）。
- **怎麼改**：Space menu 選到「會開 popup」的列時不關 menu，讓新 popup 疊在上面；取消回到 menu，完成後連同 menu
  一起清掉。直接執行、不開 popup 的列（Mark、Yank、Hidden、Zoom……）照舊執行後關 menu。Search chooser、Goto
  picker 開 finder 時同樣留在底下。建議統一在按鍵路由處理（webu 的做法，見「先看」），在 dispatch 回傳的 model 上判斷
  有沒有開出新框；「有框握著鍵盤」要把 `quitMenu` 算進去（第 1、6 條）。依疊的深度給層色（D2）。

## 10. input popup 的 `Enter` 不驗證、送不出去也不說 —— K3、L2

- **現況**：`inputpopup.go` `update()` 的 `case tea.KeyEnter`（:78–79）一律關閉並回報 committed；
  `app.go` `performInput()`（:926–956）再處理：名字是空白就直接 `return nil`（popup 已經關了，什麼都沒
  發生）；Rename 用 `os.Rename`，目標名稱已存在時會直接覆蓋同名檔案；Add 用 `O_EXCL`，已存在就靜靜失敗。
  框寬在每次 render 時依標題、hint、說明與目前的值重算（`renderFull()` :114–:116）。
- **規則**：`Enter` 一定是 submit。submit 前檢查欄位，全部合格才送出；有不合格就**不送出**：focus 停在（跳到）
  **第一個不合格的欄位**，並揭露錯誤（哪個欄位、為什麼）。filu 的 input popup 只有一個欄位，所以失敗時 popup
  留著、focus 留在那一欄、說出原因。`Enter` 不代替 `Tab` 換欄位。框的寬度不隨內容浮動（L2）。
- **怎麼改**：驗證搬到 `Enter` 當下、關閉之前：空白名字（「沒填」也是一種不合格，要由驗證本身抓到，不再靠
  `performInput()` 靜靜 return）、名字含 `/`（Rename）、目標已存在（Rename、Add）、Zip 名字為空都不送出，popup
  留著並揭露原因（例如框標題尾綴或輸入列下方一行紅字）；再按一次 `Enter` 仍停在那裡。只有驗證通過才關閉並執行。
  下框 hint `enter confirm   esc cancel`（`inputpopup.go` :97）照 D3 改成 `Enter <動詞> · Esc cancel`。加上錯誤揭露
  以後，框寬改成在 `open()` 時算一次（標題、說明、預填的值、最寬的 hint 與錯誤訊息），之後不變；值更長就照現在的
  做法從左邊截、尾端留在畫面上（webu 第三輪）。

## 11. 檔案操作的錯誤被丟掉 —— F5

- **現況**：`app.go` `_ = moveToTrash(...)`（:333，Delete）、`_ = os.Rename(...)`（:939，Rename）、
  `_ = os.MkdirAll(...)`（:944、:946，Add）、Add 的 `os.OpenFile` 失敗不處理（:947）；`open.go`
  `openFileCmd()` 的 `_ = openFile(path)`（:20，註解 :15 寫「Errors are dropped for now」）與 `openWithCmd()` 的
  `_ = c.Start()`（:36）。失敗時畫面上什麼都沒有。（copy / move / zip 的錯誤有進 Tasks，不在此列。）
- **規則**：錯誤必須立刻出現（toast 或 popup），`Esc` 可關，且不能阻塞 app。
- **怎麼改**：這些路徑把 error 帶回來，失敗時用 toast 顯示（例如 `Cannot move report.pdf to the trash:
  permission denied`）；`openFileCmd` / `openWithCmd` 回傳一個錯誤 msg，由 `Update()` 轉成 toast。

## 12. PTY 沒有 app 的出口鍵 —— K10、M1

- **現況**：`app.go` `Update()`（:270–271）在 `m.pty.isActive()` 時把每一個鍵都交給子程序（`pty_unix.go`
  `ptyKeyBytes`，包括 `Esc` 與 `Ctrl-C`，這部分是對的）；離開 shell 只能在 shell 裡打 `exit`，下框常駐
  `type exit to close`（`pty_unix.go` :263）。shell 卡住或跑著全螢幕程式時，使用者沒有辦法回到 filu。
- **規則**：PTY 裡按鍵屬於子程序；app **至少**指定一個出口鍵讓 focus 離開 PTY（選子程序幾乎不會用到的組合，例：
  kbu 的 `Alt-t`、sshu 的 `Alt+Esc`），focus 在 PTY 時常駐揭露。出口鍵以外要不要保留其他 app 的組合鍵、按了出口鍵
  之後 focus 落在哪裡，由 app 決定；保留的鍵跟出口鍵一樣常駐揭露。
- **怎麼改**：在 PTY 路由前攔一個出口鍵（建議跟 kbu 或 sshu 對齊），按下時關閉 PTY popup（結束 shell，或保留
  session 下次再接回 —— 由 app 決定；依 T1，出來後回到 panel），下框改成 `exit or <鍵> to close`。filu 的 PTY 只有
  一格 shell，不需要其他組合鍵。README「開啟、編輯，以及其他」一節的 `type exit to come back` 一起改。

## 13. panel `[1]` 的切換分頁不在 Space menu 裡 —— M3

- **現況**：panel `[1]` 上 `h` / `l`（與方向鍵）切換分頁（`handleListKey()` :533–536），但 `buildSpaceMenu()` 的
  panel `[1]` 分支（:803–:846）沒有這一列；panel `[3]` 同樣的動作有列（`Switch tab`，:853）。`[1]` 的切換分頁只能
  靠事先知道熱鍵（help 裡有一行 `h l`，但那不是能執行的清單）。三個 panel 的其他熱鍵都已經在各自的 Space menu 裡
  （`t` / `w` 在不能做時不列，見第 7 條）。
- **規則**：每個 panel 的每個 item operation 與 panel operation，都在該 panel 的 Space menu 裡；letter hotkey 是
  清單裡某一列的捷徑。
- **怎麼改**：panel `[1]` 的 `panel operation` 加一列 `Switch tab`（`l`，說明例：`next tab (h/l)`），跟 `[3]` 同一種
  寫法；只有一個分頁時照 M6 變暗（對象存在、現在不能做），跟第 7 條的 `Close tab` 一致。

## 14. 程式碼註解仍引用 VTP 的 § 編號、ZLC 與 u-family —— 文件對齊

- **現況**：`internal/ui` 等處的註解用 VTP 的 § 編號、舊名 ZLC 與「u-family」，以及已退場的
  `filu-implementation.md` 的 §8（panel chrome）（不影響行為）。只換這些；提到 kbu 的地方（`kbu form`、
  `ported from kbu`）是指 kbu 的程式碼，不動；`app.go:2`、`persist.go:12` 的 `IDEA.md` 不動。第 4、5、6 條改完後，
  `?` 相關的註解順著新行為寫（下表已照 v0.1.6 的名稱）。2026-09-27 依內容重新核對，下表就是全部（測試檔沒有）。
- **怎麼改**：照 [terminu `vtp/README.md` 的對照表](https://github.com/vulcanshen/terminu/blob/v0.1.6/vtp/README.md) 換成 tdp 編號；
  依內容比對，不要照行號：

| 檔案:行 | 現在 | 換成 |
|---|---|---|
| `cmd/filu/main.go:1` | `a ZLC terminal file manager (kbu u-family)` | `a terminal file manager (terminu family)` |
| `internal/ui/app.go:68` | `§A.1 contextual popup` | `Space menu (tdp K5, M2)` |
| `internal/ui/app.go:88` | `§A.2 global help cheatsheet` | `? key reference (tdp K6, M4)` |
| `internal/ui/app.go:470` | `§A.2 global help cheatsheet` | `? key reference (tdp K6, M4)` |
| `internal/ui/app.go:472` | `the u-family logo` | `the filu mark (terminu family)` |
| `internal/ui/app.go:799` | `ZLC §A.1 completeness` | `tdp M3` |
| `internal/ui/spacemenu.go:37` | `the §A.1 contextual popup` | `the Space menu (tdp K5, M2)` |
| `internal/ui/helppopup.go:10` | `the §A.2 non-contextual entry` | `the ? key reference (tdp K6, M4)` |
| `internal/ui/view.go:11` | `kbu colour hierarchy (§2 / §B)` | `colour hierarchy (tdp D2, P4)` |
| `internal/ui/view.go:25` | `§8.0/§8.2 powerline caps`（已退場的 implementation 文件） | `panel chip powerline caps (tdp D1)` |
| `internal/ui/marks.go:133` | `§B: one element, one semantic` | `tdp P4: one element, one meaning` |
| `internal/ui/chrome.go:16` | `§8.1`（已退場的 implementation 文件） | `tdp D1` |
| `internal/ui/chrome.go:33`、`:64` | `§8.2`（已退場的 implementation 文件） | `tdp D1` |

非 Go、但不在這次文件修改範圍內的設定檔：

| 檔案:行 | 現在 | 換成 |
|---|---|---|
| `.goreleaser.yaml:32` | `the rest of the u-family (kbu)` | `the rest of the terminu family (kbu)` |
| `.goreleaser.yaml:41` | `description: "ZLC terminal file manager (kbu u-family) — …"`（Homebrew formula 的說明，使用者看得到） | 例如 `"Terminal file manager — content search, split preview, marks"`，不帶 ZLC / u-family |

## 15. yank viewport 的選取是模式，但 core key 在裡面沒有作用 —— K11、M3、K6

（新增，2026-09-27 對照 v0.1.6 時逐條讀全文發現；規則從 v0.1.1 就在，tdp 術語「模式」的例子就是「filu yank viewport
裡的選取」。）

- **現況**：`detailyank.go` `update()`（:125–:212）：`v` 切換 `visual`（選取），選取中 `Esc` 先離開選取（:131–:134），
  這點已經符合。選取中按 `Space`、`?`、`q`、`Tab` 都沒有 case，按了沒反應；`Ctrl-C` 也被吞（第 1 條）。模式裡能按的
  鍵是 `h j k l` / 方向鍵、`0` `$`、`gg` `G`、`u` `d`、`v`、`y`（選取中是複製選取、否則複製全部）、`Esc`，但下框 hint
  不分狀態都只寫 `v:visual   y:copy   Esc:close`（:307），移動鍵哪裡都沒列。footer（`view.go` `footerBar()` :438）在
  viewport 底下仍露出 `space menu   ? help`（框高 `height-4`、置中），M1 這點符合。
- **規則**：focus 在模式裡時，`Space` 開 / 關**這個模式的按鍵清單**（列出模式裡能按的鍵，每一列按那個鍵或 `Enter`
  直接執行，執行後清單關掉）；`?` 是這個模式的 help（唯讀）；`Esc` 離開模式；`q`、`Ctrl-C` 照 K9 進入離開流程；
  `Tab` 可以暫停，但按了要有回應，說明先 `Esc` 離開模式。模式的鍵跟導覽鍵重疊時（filu 正是 `h j k l`、`u d`、`g G`），
  按鍵清單只用方向鍵移動，其餘鍵一律是「執行那一列」。模式自己的鍵必須出現在模式的按鍵清單裡。選取以外，viewport
  本身是 popup：`Space` 不作用（K5），`?` 是 viewport 的 key reference（第 4 條）。
- **怎麼改**：
  - 把模式的鍵寫成一張結構化的鍵表（鍵、說明），模式的按鍵清單與模式的 help 都從它產生（webu 的 `selectKeys`、sshu
    的選取模式）；viewport 的 key reference（第 4 條）也可以從同一張表取非選取狀態的那幾列。
  - 選取中 `Space` 開按鍵清單（新 popup，疊在 viewport 上，方向鍵移動，`Enter` 或按該鍵執行後關掉清單），再按 `Space`
    關掉；`?` 開模式的 help；`q` / `Ctrl-C` 走第 1、2 條的離開流程；`Tab` 跳 toast（例：`Esc leaves the selection first`）。
  - 下框 hint 分兩種狀態寫（例：選取外 `v select · y copy all · Esc close`，選取中 `y copy · Esc leave · Space keys`），
    框寬在打開時照最寬的 hint 定下來，切換狀態時不變（L2）。
  - 新 popup 照「先看」的清單接上動畫、尺寸、繪製與測試；測試要守「選取中 `Space` 開清單、選取外 `Space` 不作用」
    兩個方向，並逐處 mutation。

## 16. Space menu 的標題不是 `[N] label` —— D4（家族預設，不是違反）

（新增。D4 是 family default，照用最省事、不照用不算違規；webu 第二輪照改，這裡列出來讓 filu 決定。）

- **現況**：`buildSpaceMenu()` 的標題：panel `[1]` 是 cursor 所在項目的名稱，沒有項目時是 `CWD`（:805–:808）；
  `[2]` 是 `Preview`（:850）；`[3]` 是 `Tasks` / `Favorites` / `Marks`（:860、:869、:886）。panel 自己的膠囊是
  `[1] <分頁記號>`（`view.go` :193）、`[2] Preview`（:268）、`[3]` 加 `Marks` / `Tasks` / `Favorites` 的 tab 列（:162）。
- **規則**：D4「menu 標題是 focus panel 的 `[N] label`」。
- **怎麼改**：`[2]` 改成 `[2] Preview`，`[3]` 改成 `[3] Marks` / `[3] Tasks` / `[3] Favorites`（跟膠囊一致）。
  `[1]` 改成 `[1] <cursor 項目名>`（例 `[1] report.pdf`，沒有項目時 `[1] CWD`）：符合 D4 的 `[N]`，又保留 item operation
  作用在哪個檔案上（2026-09-28 user 裁定，見「已定案」第 4 題）。

## 17. 檔案列上的 `Enter` 不做事 —— K3（2026-09-28 user 裁定，見「已定案」第 1 題）

- **現況**：`handleListKey()` 的 `case "enter"`（`app.go` :525–530）只進目錄，檔案列上是 no-op；
  `dev-remarks.md`「設計決定」第一條記的是這個行為與 P4 的理由。
- **裁定**：檔案列上的 `Enter` 打開**這個檔案的 metadata popup**；目錄列的 `Enter` 維持進目錄。
- **怎麼改**：
  - 新增一類 popup（F1：唯讀資訊框，不兼 menu、不兼 viewport）：沒有游標、不能執行，`Esc` 關閉，`?` 是它的 key
    reference（第 4 條），`Space` 不作用（第 3 條），`q` / `Ctrl-C` 走離開流程（第 1、2 條）。
  - 欄位：完整絕對路徑（symlink 多一列指向的目標）、類型（沿用 preview 的 magic bytes 判型）、精確大小（人話單位 +
    bytes）、Modified / Accessed / Changed（macOS 為 Created）完整日期時間、權限（`rwx` + 八進位）、`owner:group`。
  - **文字 word-wrap、能折行，所有資訊都完整揭露在 popup 裡**（長路徑折行，不截斷）。框寬在打開時定一次（L2），
    內容高過畫面時才捲動（沒有游標）。
  - 取不到的欄位（例如 stat 失敗）照 F5 在 popup 裡寫出原因，不留空白。
  - 新 popup 照「先看」的清單接上動畫、尺寸、繪製與測試。
  - `dev-remarks.md`「設計決定」的 `Enter` 那條改寫成新行為；README 兩份的五鍵表 `Enter` 說明、「瀏覽」一節一起改。

---

## 已經符合、不用修的（對照 v0.1.6）

下次對照時不必重查：

- **K2、K8（v0.1.5，多行文字寫入狀態的 `Tab` 是縮排）**：filu 沒有多行文字輸入。input popup 是單行（`inputpopup.go`
  :20），finder 的輸入列也是單行；`rg -i "textarea|multi.?line"` 在非測試程式裡沒有輸入相關的結果。能寫多行的只有
  `[s]hell` 的 PTY，按鍵屬於子程序（K10）。
- **K2（v0.1.6，單一輸入框有灰字提議時 `Tab` 接受提議）**：filu 沒有灰字提議。Rename 預填的是原名（`app.go` :577）、
  Zip 預填的是 `suggestZipName()`（:697），兩者都是可以直接編輯的**值**，不是灰字；Add 是空的。input popup 裡 `Tab`
  沒有 case、不作用，也不會漏到底下去切 panel（input popup 的路由排在主 switch 之前，:311）。finder 輸入列同樣沒有
  提議（第 2 條讓 `Tab` 在輸入列與結果清單之間切換）。
- **K10（v0.1.4，至少一個出口鍵）**：filu 的 PTY 只有一格 shell，只需要出口鍵（第 12 條）。
- **M3 與 P3「同一個動作在兩區」**：沒有熱鍵同時出現在兩個區。全域動作只有離開；切分頁、Goto、Search、Shell、Sort
  都作用在 `[1]`，是 `[1]` 的 panel operation（webu 的 `P` / `N` 那種「作用在 panel 的動作放在 global」在 filu 沒有）。
- **K5 在其他 popup**：yank viewport 與 finder 清單態不理 `Space`；input popup 與 finder 輸入態的 `Space` 是字元（K8）。
  要改的只有第 3 條列的四處。
- **M2 vs M6（`Favorite` 只在目錄列出現）**：檔案**永遠**不能收藏，不是「現在不能」，所以是動作對這個項目不成立，
  照 M2 不列；M6 的變暗只給狀態一變就能做的動作（例如分頁已滿的 `Tab`，第 7 條）。2026-09-28 user 裁定。
- **S3、S4**：splash 在路由第一站（`app.go` :265）；`V` 只在主 switch（:472），popup、輸入態、PTY 都叫不出來。

---

## 已定案（原「待確認」，2026-09-28 user 逐題裁定）

1. **（已定案，2026-09-28）`Enter` 對檔案列不做事，算不算符合 K3？** → 不算。檔案列上的 `Enter` 改成打開該檔案的
   metadata popup（唯讀、文字折行、資訊全部揭露在框裡），見第 17 條。
2. **（已定案，2026-09-28）`Favorite`（`f`）只在 cursor 是目錄時才列（`app.go` :819）。** → 維持不列，已移到
   「已經符合」。
3. **（已定案，2026-09-28）`b` 也能關掉 breadcrumb popup（`breadcrumbpopup.go` :71）。** → 拿掉 `b`，只留 `Esc`，
   併入第 3 條。
4. **（已定案，2026-09-28）panel `[1]` 的 Space menu 標題要不要照 D4 改成 `[1] …`？** → 兩者並列：
   `[1] <cursor 項目名>`（空目錄 `[1] CWD`），併入第 16 條。

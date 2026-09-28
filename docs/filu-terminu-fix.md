# filu — terminu fix

filu 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.7/principle)（tdp v0.1.7）的地方，逐條待修。
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
  上一輪沒抓到的。filu 這次也是：原第 15 條（yank viewport 的選取是 K11 的模式，已修）的規則從 v0.1.1 就在。
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
  與 Space menu 會被清掉，`Esc` 就回不去（filu 的判斷是 `boxOverSpaceMenu()`，已把 `quitMenu` 算進去；原第 6、9 條已修）。
- **（webu）F4 可以統一在按鍵路由處理**：記下按鍵前最上層的等級，按鍵後若不是 `Esc`、最上層掉了一級以上，就關掉
  底下的 menu。執行 menu 的一列時，在 **dispatch 回傳的 model** 上判斷有沒有開出新框，不要在舊 model 上關（會關在
  沒人回傳的副本上；locku 也踩過 value receiver 的同類陷阱）。
- **F3 不只看 toast**：判斷「popup 還在不在」用開啟中或已開（`owns()` 一類），不用含關閉中的 `isActive()`。
  locku 的 bug 在 toast，sshu 照抄只寫了 toast，實際上每一個 popup 都用 `isActive()`；filu 也是，已修（原第 8 條：路由改用 `owns()`）。
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
  filu 的 yank viewport 選取就是這種模式（`selectKeys`，原第 15 條已修）。
- **（webu）改完一條就回頭檢查其他畫面的提示。** `Space` 不再關 popup 之後，下框的 `Space close` 要跟著改。

**輸入**

- **（sshu）K3 改成一律送出以後，驗證要把「有沒有填」一起問**：以前「沒填就不會送出」替驗證擋掉了必填檢查；
  `Enter` 一定送出之後，空白要由驗證本身抓到（filu 的 `nameCheck()`，原第 10 條已修）。
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

（以下第 19–22 條：2026-09-28 修完第 1–18 條後，拿 tdp v0.1.7 `rules-zh_TW.md` 全文逐條再對一次時找到。）

## 22. Open with 的 Default 不 confirm，`o` 卻會 —— F6（待確認）

- **現況**：`o` 用 OS 預設 app 開啟前先 confirm（`confirmOpen`）；`O` 的 picker 選 `Default`（`openwith.go` `runOpenWith()`
  idx 1）直接 `openFileCmd`，同一個動作不 confirm。其他 open-with app 也不 confirm。
- **規則**：F6：一個動作一旦決定要 confirm，每次都 confirm。
- **待 user 裁定**：picker 裡的選擇算不算已經確認過。

---

## 已經符合、不用修的（對照 v0.1.6）

下次對照時不必重查：

- **K2、K8（v0.1.5，多行文字寫入狀態的 `Tab` 是縮排）**：filu 沒有多行文字輸入。input popup 是單行（`inputpopup.go`
  :20），finder 的輸入列也是單行；`rg -i "textarea|multi.?line"` 在非測試程式裡沒有輸入相關的結果。能寫多行的只有
  `[s]hell` 的 PTY，按鍵屬於子程序（K10）。
- **K2（v0.1.6，單一輸入框有灰字提議時 `Tab` 接受提議）**：filu 沒有灰字提議。Rename 預填的是原名（`app.go` :577）、
  Zip 預填的是 `suggestZipName()`（:697），兩者都是可以直接編輯的**值**，不是灰字；Add 是空的。input popup 裡 `Tab`
  沒有 case、不作用，也不會漏到底下去切 panel（input popup 的路由排在主 switch 之前，:311）。finder 輸入列同樣沒有
  提議（`Tab` 已在輸入列與結果清單之間切換，原第 2 條）。
- **K10（v0.1.4，至少一個出口鍵）**：filu 的 PTY 只有一格 shell，只需要出口鍵（`Alt+Esc`，原第 12 條已修）。
- **M3 與 P3「同一個動作在兩區」**：沒有熱鍵同時出現在兩個區。全域動作只有離開；切分頁、Goto、Search、Shell、Sort
  都作用在 `[1]`，是 `[1]` 的 panel operation（webu 的 `P` / `N` 那種「作用在 panel 的動作放在 global」在 filu 沒有）。
- **K5 在其他 popup**：yank viewport 與 finder 清單態不理 `Space`；input popup 與 finder 輸入態的 `Space` 是字元（K8）。
  原第 3 條列的四處已修（`spaceToggle`，confirm、breadcrumb、help 拿掉 `" "`）。
- **M2 vs M6（`Favorite` 只在目錄列出現）**：檔案**永遠**不能收藏，不是「現在不能」，所以是動作對這個項目不成立，
  照 M2 不列；M6 的變暗只給狀態一變就能做的動作（例如分頁已滿的 `Tab`，原第 7 條已修）。2026-09-28 user 裁定。
- **S3、S4**：splash 在路由第一站（`app.go` :265）；`V` 只在主 switch（:472），popup、輸入態、PTY 都叫不出來。

---

## 已定案（原「待確認」，2026-09-28 user 逐題裁定）

1. **（已定案，2026-09-28）`Enter` 對檔案列不做事，算不算符合 K3？** → 不算。檔案列上的 `Enter` 改成打開該檔案的
   metadata popup（唯讀、文字折行、資訊全部揭露在框裡）；原第 17 條，已修（`metaPopup`）。
2. **（已定案，2026-09-28）`Favorite`（`f`）只在 cursor 是目錄時才列（`app.go` :819）。** → 維持不列，已移到
   「已經符合」。
3. **（已定案，2026-09-28）`b` 也能關掉 breadcrumb popup（`breadcrumbpopup.go` :71）。** → 拿掉 `b`，只留 `Esc`，
   併入原第 3 條，已修。
4. **（已定案，2026-09-28）panel `[1]` 的 Space menu 標題要不要照 D4 改成 `[1] …`？** → 兩者並列：
   `[1] <cursor 項目名>`（空目錄 `[1] CWD`），併入原第 16 條，已修。

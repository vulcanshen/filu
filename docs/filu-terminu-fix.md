# filu — terminu fix

filu 發版前要修的 bug，逐條待修。修好一條就刪掉一條。

盤點日期：2026-10-06。依據 `main` 的 `04f7b72`（已對齊 tdp v0.1.23，工作區乾淨）。**這一輪不是 tdp 改版**：是 input 盤點
（2026-09-29，terminu `.local/input-survey/filu.md`）翻出來、跟之後 tdp components 怎麼定無關的 bug；五個 app 修完就發版。
tdp 版本不變，連結不用改。這份清單由 terminu session 寫好留在工作樹，還沒 commit。


## 先看

- 每條先寫一個會紅的測試，再修；CHANGELOG `[Unreleased]` 的 Fixed 各記一條（合併或拆開由 filu 定）。
- 第 1 條是五個 app 共通的做法，五份清單的「做法」同一段文字：畫出來的樣子與行為要一樣，程式怎麼寫各 app 自己定。
- 最後「這一輪不修」列的要等 components 的 input 檔定案，不要先動。
- 修完刪掉這份清單。不 push、不打 tag、不發版：發版是下一步，版號由 user 在 terminu session 一個一個定。把這一輪寫進
  terminu `.local/family-fix/filu/README.md`。


## 1. 單行的值收進換行、Tab 與控制字元 —— 五個 app 共通

**現況**：
- Rename、Add、Zip（`inputPopup`）：貼上的控制字元留在值裡，畫面經 `safeName`（`list.go:130-143`）濾掉、看不到（貼 `x\ny`
  顯示 `xy`，值是 `x\ny`）；`nameCheck`（`app.go:1174-1201`）不檢查，檔名就帶著使用者看不到的換行建出去。
- Search、Find、Goto 的 query（`searchModel`）不經 `safeName`，換行原樣輸出，那一列斷成兩列、框錯位。

**做法**（五個 app 同一段，2026-10-06 user 定案；之後寫進 tdp components 的 input 檔）：

- 範圍：單行的值 —— 一行文字、路徑、密碼／PIN、搜尋與篩選列。多行編輯框不在這條（只有 webu 的 editor，另有一條）。
  只收數字的欄位照舊只收 `0`–`9`。
- 收進值：只看以文字進來的字元（貼上的那一段）。按下去的 `Tab`、`Enter`、`Ctrl-J` 這些鍵照舊做它們原本的事（K2、K3），
  不變成值裡的字元（locku 修的時候補的，2026-10-06）。
  - 換行（`\r\n` 算一個，單獨的 `\n`、`\r` 也各算一個）與 Tab 原樣留在值裡，不換成空白、不刪。`\r\n` 原樣存或存成
    `\n` 都可以（locku 原樣存；webu 存成 `\n`，之後一個 rune 就是一個單位，`Backspace`、遮罩、量寬都不用另外處理），
    畫、數、刪都當一個。
  - 其他控制字元（其餘的 C0、DEL、C1）丟掉。
- 預填的值（原本的名字、從檔案或別的程式讀來的值、提議）打開時就走同一個過濾：換行與 Tab 照樣畫、照樣擋，其他控制字元
  丟掉，不然 ESC 會直接送進終端機（webu 修的時候補的，2026-10-06）。
- 畫：換行畫成 `\n`、Tab 畫成 `\t`，Red `#f38ba8`，佔 2 格，跟手打的 `\`、`n`（一般值的顏色）分得開。量寬、截斷、
  捲動都把它當成一個 2 格寬、不能切開的單位。整列畫成灰色的列（沒在打的篩選列、提議）`\n`、`\t` 也跟著灰，整列一次
  上色（webu 的做法）。
- 遮罩的值：照樣遮罩，一個換行或 Tab 也是一顆遮罩符號，不露出 `\n`；使用者靠錯誤列知道。
- 刪：`Backspace` 一次刪掉整個（它本來就是一個字元）。
- 送出：值會被拿去用的 input（送出、存檔、執行、交給別的程式），值裡有換行或 Tab 時 `Enter` 不送出，錯誤列說出哪一欄
  不能有換行或 Tab（英文；句式、大小寫照該 app 現有的錯誤訊息，例：`Name can't have line breaks or tabs`）。其他照 K3：
  多欄表單 focus 跳到第一個不合格的欄位、label 變 Red；有「第一次送出後每鍵重驗」的照舊。
  - 原本送出不會失敗、所以沒預留錯誤列的 input，現在會失敗了，照 F7 打開時就預留錯誤列。
- 搜尋與篩選列（值只拿來找東西，不存、不執行；`Enter` 選的是清單裡的項目）：只照上面畫，不擋。

**為什麼**：單行的值裡換行沒有意義。原樣畫出來會把框畫壞；偷偷換成空白或刪掉，又改了使用者的值而看不出來（user：
「應該轉成 `\n` 或 `\t` 這種明確顯示」）。只在畫面上轉、值裡留原字元，是為了跟手打的 `\n` 分得開，也不會把沒有意義的
字元送出去。

**filu 要改的地方**：
- 收字：`inputPopup` 的 `update`（`KeyRunes` 附加到尾端那裡）、`searchModel` 的 query 編輯。
- 畫：Rename、Add、Zip 的值不再經 `safeName`（它會把換行藏起來），改畫 Red `\n`／`\t`；`truncPathLeft`（`width.go:122-130`）
  把它當成一個 2 格單位。三個 finder 的 query 同樣畫。結果列照舊走 `safeName`（那是磁碟上的名字，不是使用者打的值）。
- 擋：Rename、Add、Zip 的檢查。
- 預填：Rename 的原檔名、Zip 的建議名。macOS 的自訂圖示檔就叫 `Icon\r`（filu 2026-07 修框線時遇過），是現成的測試案例：
  Rename 它，值裡看得到 Red `\n`，不刪掉就送不出去。
- 不擋：Search、Find、Goto 的 query。


## 2. Add 的值可以建到目前目錄外面

**現況**：值可以含 `/`，中間目錄一併建立（`src/新檔案.go` 建在 `src` 底下）。但值含 `../` 時照 `filepath.Join` 解析，可以
建在目前目錄以外；以 `/` 開頭的值被 `filepath.Join` 接在目前目錄底下，打的跟建出來的不一樣。檢查在 `nameCheck`，送出在
`performInput`（`app.go:1206-1240`）。盤點是讀程式碼判斷的，沒有實測。

**怎麼改**：檢查時把值（去掉結尾 `/`）`filepath.Clean` 之後，以 `/` 開頭，或落在目前目錄外面（是 `..` 或以 `../` 開頭）→
錯誤列，框留著；訊息照 filu 現有的句式（例：`<值> is outside this directory`）。中間目錄照舊可以建；`a/../b` 這種清理後
仍在目前目錄裡的照舊可以。先寫測試確認現在真的會建到外面。README 兩份（第 147 行）要不要補一句「只建在目前目錄底下」
由 filu 定。


## 3. Zip 的值只剩 `/` 或 `.` 時，框關了、什麼都沒做

**現況**：檢查只擋空值（`app.go:1176-1178`）。送出時 `zipFileName`（`zip.go:59-68`）取 basename，`/`、`.` 處理成空字串，
`startZip`（`zip.go:72-89`）直接不做事：popup 照樣關閉、沒有任何訊息。

**怎麼改**：檢查時 `zipFileName` 得到空字串 → 錯誤列（例：`Not a name for a zip file`），框留著。


## 4. CHANGELOG `[Unreleased]` 與註解說「CJK 字型的 icon 就是兩格」

**現況**：tdp D7（v0.1.23）不拿「一定佔兩格」的字型當例子 —— filu 自己實測過，Maple Mono NF CN 的 icon 看起來兩格，游標只
前進一格。CHANGELOG `[Unreleased]` 的 Fixed 卻寫「On fonts that draw icons two cells wide (CJK Nerd Fonts)」；程式註解也有三處：
`cmd/filu/main.go:37`（`CJK fonts draw them 2-wide`）、`internal/ui/view.go:272`（`All slots are wide icons on a CJK Nerd Font`）、
`internal/ui/iconwidth_unix_test.go:14`（`CJK font`）。webu 這一輪找到同一類，順手改了。

**怎麼改**：`[Unreleased]` 那句拿掉括號裡的例子，或照 locku 寫成不說死的「as some made for CJK do」；三處註解照實寫（icon
佔幾格看字型與終端機，啟動時量）。已發版的段落是歷史，不動。


## 這一輪不修

沒有。


## locku、webu 先修完的經驗（2026-10-06）

- **值裡的 `\t`、`\n` 不能直接交給 lipgloss**：`Render` 會把 Tab 換成空白、在換行處斷成兩列。先換成要畫的 `\n`、`\t`，
  再上色。
- **同一個值有兩條路進來，要用同一個過濾。** locku 的設定畫面不過濾、鎖定畫面只收 `IsPrint`，兩邊各自合理，合起來就設得出
  一個解不開的 PIN。
- **測試要用跟舊行為不同的輸入。** 貼 `12\n34` 時字元數剛好等於單位數，測不出「`\r\n` 算一個」；換成 `12\r\n34` 才分得出來。
- **bracketed paste 是一整個 `KeyRunes`**（`Paste: true`，換行、`\r`、Tab、ESC 都在裡面）；打字進來的 `KeyRunes` 不會有控制
  字元，所以每個 `KeyRunes` 都過濾是安全的，不用看 `msg.Paste`。按下去的 `Tab`、`Enter`、`Ctrl-J` 是別的 `msg.Type`。
- **灰的列要整列一次上色。** 把提議、沒在打的篩選列拆成「灰字＋`\n`」好幾段，webu 原本「整列是一段 Overlay0」的測試就紅了；
  先畫成純文字，再整列上色一次。
- **預填的值不只一個入口。** webu 加書籤的標題提議每一鍵之後重算，那裡也要過濾；每一個入口都要有測試，拿掉其中一處的過濾要紅。
- **「這個框沒有錯誤列」的舊測試要換量法。** 每個框都有錯誤列之後，拿沒有錯誤列的框當基準量高度就失效了；改量「被拒時原因
  寫進框裡、高度跟打開時一樣」。
- **實機確認用 `tmux paste-buffer -p`**：送的是 bracketed paste。tmux 預設把 LF 換成 CR，正好也驗到「單獨的 `\r` 算一個換行」。
- **`[Unreleased]` 與測試註解也要 grep。** webu 的 CHANGELOG `[Unreleased]` 還寫著舊的變數名與「CJK 字型就是兩格」：已發版的段落是
  歷史不動，還沒發的段落會原樣出現在下一版的說明裡。

## 待確認

沒有。

# filu — Implementation

[**VTP** — Vulcan's TUI Design Principle](https://github.com/vulcanshen/thoughts/blob/main/vtp.md)
是一套**與領域無關的通用 TUI 設計原則** —— 目標:
不看文件、不背 hotkey,靠一套跨 surface 不變的基礎操作就能用完整個 app。VTP
**不屬於任何單一 app**:kbu 是它在 K8s domain 的一個實現、**filu 是它在
filesystem domain 的另一個平行實現**。兩者是 sibling、共用同一套 VTP,不是
誰派生自誰。

本文件是 filu 對 VTP 的**具體落地紀錄**,結構鏡射同為實現的
`kbu-implementation.md`(平行參照、非上位),逐節對照 filu 的實作 —— VTP
是 **interface**、本文件是 filu 這個 **implementation class**(kbu 是另一
個 class)。想知道**為什麼**這樣做、看 VTP;想知道 filu **怎麼**做,看這裡。

> **設計權威順序**:`.forge/meta/IDEA.md`(filu 專屬決定)> **VTP**
> ([Vulcan's TUI Design Principle](https://github.com/vulcanshen/thoughts/blob/main/vtp.md))。
> `kbu-implementation.md` 是**平行實現的參照、不是 filu 的上位權威**。衝突
> 時以 IDEA.md 為準、VTP 其次。
>
> **狀態標記**:本文件描述**當前已落地**的實作(對齊 `v0.3.0`)。尚未完成者
> 標 `(planned)`,不宣稱未落地的行為。實作狀態總表見 §9。

本文件另收錄幾個 filu 打磨過程沉澱下來的**實作案例**(隨寬逐階縮字、動態
tab 與 breadcrumb 分工、quit 用 picker 而非 confirm、原生串流 finder、marks
bucket 兩層 pick、依動態 tab 分欄的 zoom、preview yank visual、破壞性動作
一律先 confirm、Zip 輸出到 temp 再交給既有落地路徑),分散在對應章節,並在
§8 集中談 panel chrome。

---

## §A. Implementation in filu

通用 §A.0(score)+ §A.1(contextual track)+ §A.2(non-contextual
track)在 filu 的具體實現。

### §A.0 filu score 對照

| 軸 / 結果 | filu 值 | 計算 |
|---|---|---|
| **X. 揭露程度** | ~1.0 | Space menu 列出當前 focus 的 contextual 動作 100%、`?` help popup 列出全域動作 100%。以 user 學習單位計:`o` open 對所有型別通用算 1 個 action、`m` mark 對所有 entry 通用算 1 個 |
| **Y. core-key role 數量** | 5 | `Tab`(focus 切換)/ `Enter`(確認·進入)/ `Esc`(取消·回上層)/ `Space`(contextual 入口)/ `?`(non-contextual 入口)。`Ctrl+C`(硬退)與 `q`(cd-on-quit picker)不另計 role —— 見 §A.0.Y |
| `min(1, 5/Y)` 係數 | 1.0 | Y = 5、無 penalty |
| **Score** | `~1.0 × 1.0` = **~100%** | 不靠事先學就能用 |

filu 的定位不是「再做一個 yazi」,而是「**第一次開就能不看文件開到底**」的
檔案管理器 —— letter hotkey 是加速捷徑、不是必經之路,光靠 `Space` + `?`
就能走完該 focus 的所有動作。

### §A.0.Y filu core-key 集合(5 個)

| Core-key | filu 語意 | 對應通用條款 |
|---|---|---|
| `Tab` | focus 切到下個 panel(`1`–`3` 直達 alias) | §4.1 |
| `Enter` | 進入目錄 / popup 內確認 | §4.1 |
| `Esc` | 關閉最上層浮層 / 回上層目錄(LIFO back) | §4.3 |
| `Space` | §A.1 contextual 入口(Space menu) | §A.1 |
| `?` | §A.2 non-contextual 入口(help popup) | §A.2 |

5 個,剛好通用 §A.0.Y 上限。letter hotkey(`o`/`O`/`m`/`y`/`r`/`D`/`f`/`F`/
`c`/`v`/`a`/`S`/`s`/`.`/`b`/`t`/`w`/`z`/`p`/`Z`/`C`/`/`)與導覽 chord
(`gg` 跳頂、`go` goto)**不算 core-key**,是入口內動作的加速捷徑。

**`Enter` 不開檔**:filu 的 `Enter` **只進目錄**,對檔案列是 no-op。開檔是
`[o]pen`(OS 預設 app,先 confirm)/ `[O]pen with`(挑 app)的職責。理由是
§B 一元素一語意 —— 「進入」與「交給外部程式」是兩件不同代價的事,`Enter` 一
鍵兼職會讓 user 在目錄與檔案間游標移動時無法預期後果。

**`Ctrl+C` 與 `q` 為何不各記一個 role**:`q` 不是「取消」—— 它開 cd-on-quit
picker(選離開時要 `cd` 去哪),語意屬「離開 app 並帶目錄回 shell」,是一個
全域動作(列在 footer + help)。`Ctrl+C` 是逃生硬退,與 `Esc`(退浮層 / 退
目錄)語意不同、不重疊,屬 emergency exit,不佔 core-key role 上限。取消
role 由 `Esc` 單獨承載(§4.3)。

**`gg` / `go` chord 為何不 +Y**:`gg` 是 vim 跳頂(單 `g` 待命等第二鍵、對齊
kbu)、`go` 開 Goto picker —— 兩者都是既有動作的**加速捷徑**、走 letter-hotkey
層,不是新的 core-key role。實作:單一 `AppModel.pendingG` 掛在主 switch 的
chokepoint(所有 popup return **之後**、只管主面板),`gg` 落既有 `case "g"`、
`go` 呼 `handleListKey("go")`。

### §A.1 Contextual track in filu — Space menu

每個 panel focus 都接 `Space`,列出「這個 focus 此刻能做什麼」。入口自身在
**footer** 揭露(`view.go footerBar`):

```
 space menu   ? help   tab/1-3 panels   q quit
```

user 第一次開、沒看 README,從 footer 就知道按 `Space` 會跳選單。

filu 各 focus 的 Space menu(`app.go buildSpaceMenu`,`groupedMenu` 分
item-region / panel-region、cursor-first,見 §6.6):

| Focus | item-region 動作(對游標項) | panel-region 動作(對這個面板 / tab) |
|---|---|---|
| **[1] List** | Open `o`、Open with `O`、Mark `m`、Yank `y`、Rename `r`、Delete `D`、(Favorite `f`,僅目錄) | (Copy `c`、Move here `v`,僅 bucket 非空)、Search `/`、Goto `go`、Favorite `F`、Breadcrumb `b`、(Tab `t`,未達上限)、(Close tab `w`,>1 個 tab)、Add `a`、Sort `S`、Shell `s`、Hidden `.`、Zoom `z` |
| **[2] Preview** | Yank `y`(開 yank viewport) | Zoom `z` |
| **[3] Marks** | Pick `p`、Yank `y`、Unmark `m` | (Zip `Z`、Clear `C`,僅 bucket 非空)、Switch tab `l`、Zoom `z` |
| **[3] Tasks** | Delete `D` | Switch tab `l`、Zoom `z` |
| **[3] Favorites** | Open in `o`、Delete `D`(unfavorite) | Switch tab `l`、Zoom `z` |

**完整性 audit**:新增一條 contextual 動作,必須同步在對應 focus 的 Space
menu 加 entry,不能只綁 letter hotkey,否則是原則破洞。反例警覺:`Copy`/`Move
here`/`Zip`/`Clear` 只在 marks bucket 有內容時才出現在 menu —— 這是 state
gating,不是隱藏;`Favorite f` 只對目錄列出現,同理。

### §A.2 Non-contextual track in filu — `?` help popup

`?` 在任何 surface 跳出全域動作清單(`helppopup.go`)。入口自身同樣在 footer
揭露(`? help`)。

filu 是檔案管理器,domain 動作幾乎都是 contextual(對 cursor item / 對當前
tab),所以 **§A.2 全域軌很薄**:

| 全域動作 | key | help popup 內列 |
|---|---|---|
| 說明 | `?` | ✓ |
| 離開(cd-on-quit picker) | `q` | ✓ |
| 硬退 | `Ctrl+C` | ✓(footer / README) |
| 切面板 / 切 tab / 導覽 | `Tab`·`1-3`·`h/l`·`Enter`·`Esc` | ✓(core-key 揭露) |
| Zoom | `z` | ✓ |

沒有 kbu 那種 namespace / context 全域 toggle —— filesystem 的「當前位置」
是每個 tab 各自的 CWD(contextual),不是一個全域狀態。

### filu contextual / non-contextual 動作邊界(audit 用)

| 動作 | contextual? | track |
|---|---|---|
| 對 cursor item 做 Open/Mark/Yank/Rename/Delete/Favorite | ✓ 對 cursor row | §A.1 |
| 對當前 tab 做 Add/Sort/Hidden/Shell/Copy·Move-land/Search/Goto/Breadcrumb/Tab/Close | ✓ 對當前 tab | §A.1 |
| Preview yank·zoom | ✓ 對 cursor item 的視角 | §A.1 |
| 對 marks bucket 做 Pick/Unmark/Zip/Clear | ✓ 對 [3] 這個面板的內容 | §A.1 |
| Help / Quit | ✗ 全域 | §A.2 |

---

## §B. 元素專職化 in filu

| 元素 | 專職語意 | 不准兼職 |
|---|---|---|
| **Lavender**(`userColor`) | 使用者足跡:breadcrumb 路徑文字、unfocused panel 的 cursor bar、breadcrumb popup「你在此」層、Find preview 的命中行 bar | panel / popup border |
| **Subtext1**(`handColor`) | focused panel 的 cursor bar(「當前的手」) | 其他狀態 |
| **structural blue**(`focusColor` `#89b4fa`) | panel border 亮=focus、chrome、list 的 mark glyph | 不拿去做檔案型別色 |
| **綠 `#a6e3a1`** | 「兩狀態同時成立 / 已進下一步」:mark+favorite 合併 glyph、eza executable-bit、perm `x` | 純裝飾 |
| **黃 `#f9e2af`** | favorite(星)、eza `r` bit / owner | 不當 mark |
| **Peach / Red** | warning / error override(quit 有任務跑的紅字、Task error、perm `w`) | 不參與 popup layer scale |
| **Popup border 色** | popup layer 明度(`popupLayerColor`,lavender→sapphire) | 不 hardcode |
| **eza 型別色**(`fileColor`) | 檔案型別 / executable-bit | 不當狀態色 |
| `markGlyph` `f0b14` | list mark 欄「在 marks bucket 裡」 | 不當 [3] 的 land-subset 勾 |
| `markPickGlyph` `f05d` | [3] Marks「在 land 子集」 | 不當 list 的 bucket 勾 |
| `iconPin` `f04ce`(星) | 「這個目錄是 favorite」 | 不當 mark |
| `markFavGlyph` `f0a74` | mark **且** favorite 的合併單格 glyph | 不拆成兩格並排 |
| `tabMarks`(rocket/cat/dog/paw/egg) | tab 位置身分 | 不編碼路徑(路徑歸 breadcrumb) |
| `[X]label` bracket | letter hotkey discoverability | 純 label 不加 bracket |
| `Esc` | 關閉 / 退出 | 永遠不當「確認」 |
| `Enter` | 進入目錄 / popup 確認 | **不開檔**(開檔是 `o` / `O`) |
| Nerd Font 型別 glyph | 型別訊號(這列是什麼型別) | 不當熱鍵 signal、不當純裝飾 |

**三個 §B 實例值得點名**:

- **兩種 pick 勾分家**(`marks.go`):list 的 `m` 標「這個檔**在 marks bucket
  裡**」用 `markGlyph`(f0b14);[3] Marks 的 `p` 標「這個 bucket 項**被選進下
  一次 land 的子集**」用 `markPickGlyph`(f05d,check-circle)。兩者是**不同語
  意的兩個狀態**,若共用同一 glyph,user 會分不清「在 bucket」與「在 land 子
  集」—— 所以刻意分成兩個 glyph、不混用。
- **mark 欄是單格、狀態合併不並排**(`list.go markCell`):一列可能同時「在
  bucket」(藍)與「是 favorite」(黃星)。filu **不**把兩個 glyph 並排 ——
  並排會讓欄寬跟狀態綁動、破壞 Width stability;改用**第三個合併 glyph**
  `markFavGlyph`(綠)佔同一格。永遠一格,`markCellW()` 取三者最大值鎖死。
- **[3] Marks 的列 icon 是真型別 icon**:bucket 只存路徑,所以 `pathIcon()` 用
  `os.Lstat` 回讀型別再走跟 list 同一支 `fileIcon` —— 全部畫成通用檔案 glyph 曾
  是個 bug(v0.3.0 修)。symlink 不算 dir,對齊 list `readEntries` 的 DirEntry
  語意。顏色維持 `userColor`:eza 色盤是 list 的語言,「已被我 carry」是 bucket
  的語言,兩套不混。

---

## §1. 空間結構 in filu

### 1.1 版面 grid(3 panel,零頂部 content row)

```
┌────────────────────────────────────┬─────────────────┐
│ [1] 🚀\🐱  ← tab bar(border title)│ [2] Preview     │
│ ~/Documents/sideproj/filu          │                 │
│ ────────────────────────────────── │  (右欄,上 2/3) │
│  M  2026-08-31 14:02  vulcan  rw-  │                 │
│ 󰈙 internal/  …(檔案清單、多欄)     │                 │
├────────────────────────────────────┴─────────────────┤
│ [3] Marks \ Tasks \ Favorites        (全寬,下 1/3)   │
└──────────────────────────────────────────────────────┘
 space menu   ? help   tab/1-3 panels   q quit    ← footer(唯一 content row)
```

grid `[1][1][2] / [3][3][3]`:上排 `[1]` list 與 `[2]` Preview 以 **2:1** 並排
(資訊密度高的 list 值這個寬度)、佔 `midH * 2 / 3`;下排 `[3]` 是一個**全寬的
tabbed 面板**(Marks | Tasks | Favorites)、佔剩下的 1/3。

**頂部沒有任何 content row** —— breadcrumb 與目錄狀態都已收進面板內部
(§8.2 / §8.3),`View()` 的 `midH = height − 1`,只扣 footer 一列。
`view.go normalMiddle` 用 `joinH`/`joinV`(display-width aware)組版。

> **演化**:早期是 5-panel(`[1]`Places·`[2]`list·`[3]`Preview·`[4]`Carries·
> `[5]`Meta)+ header 麵包屑 + top status bar 兩條頂部列。v0.2.0 收斂成現在的
> 3-panel:Places 併進 `[3]` 的 Favorites tab 與 Goto picker、Meta 併進 list 的
> **多欄**(mtime / owner / perms / size)、Carries 與 Tasks 併成一個 tabbed
> `[3]`。v0.2.8 再把頂部兩列拆光。方向一致 —— **面板數是成本,能收進既有面板的
> 就不開新面板**。

### 1.2 窄寬可用、Width stability、與「隨寬逐階縮字」

- **窄寬**:`w < 72` 時放棄 grid,只顯示 list 單欄(`view.go normalMiddle`);
  `z` Zoom 是任何面板的逃生艙(見本節下方 zoom)。
- **Footer 行數固定**:footer N=1,選定即鎖死、絕不 reflow(通用 §1.3)。
- **list 欄位隨寬遞減、name 永不消失**:`computeListCols(w)` 依序丟掉
  owner → size → mtime → perms → mark,name 欄永遠保底 `colNameMin = 12`。
  這是**階梯式**降階、不是等比壓縮 —— 每一階都仍然可讀。

#### 隨寬逐階縮字(breadcrumb + Goto picker 的 Favorites 清單共用一套)

動態文字要塞進固定寬度的 slot 而**不讓寬度跟內容綁動**(通用 §1.2 Width
stability)。filu 對「一條路徑要縮進 N 格」用一套**漸進四階**演算法
(`header.go fitPathSegments`):

1. **放得下 → 全名**:`~/Documents/sideproj/filu`。
2. **不夠 → 從前段起逐段縮成首字元**(current/末段永不縮):
   `~/D/sideproj/filu` → `~/D/s/filu`。一次縮一段、縮到剛好放得下就停,盡量
   保留細節。
3. **還不夠 → 中間 `…`**,保住 root + 盡量多的尾段:`~/…/sideproj/filu`。
4. **極窄 → 末段單獨呈現**(由 `padDisp` / `truncate` 硬截兜底)。

`fitPathSegments(segs, fits)` 是**泛型**:傳入不同的 `fits` predicate 就能在
不同度量下複用 —— breadcrumb 量的是 **renderCrumb 的渲染寬**、Goto picker 的
Favorites 清單量的是 **plain 字串寬**(`fitPath` → `joinSegs`)。同一套四階邏
輯、兩種量測。**別再各寫一份**:兩處若各自實作,縮法會漂移、user 會覺得
「breadcrumb 跟 favorites 的縮法不一樣」。

#### 動態 tab(1–5)與 breadcrumb 分工

panel `[1]` 的目錄分頁是**動態 1–5 個**(`t` 開新分頁、`w` 關 active、至少留 1;
都在 Space menu、標籤 `Tab`/`Close tab`)。兩個 Width stability 決定:

- **tab 標籤 = 一個動物 glyph、不是 dir basename**(`view.go tabMarks`:
  rocket_launch / cat / dog / paw / egg_easter)。理由:basename 長度不定、會讓
  tab bar 寬度跟內容綁動;單一 glyph 是**固定寬位置標記**,tab 數在 1–5 之間增
  減、bar 寬度階梯穩定。**「這個 tab 在哪」由面板內第一列的 breadcrumb 承載**
  —— tab bar 只負責「有幾個分頁、哪個 active」。這是分工(§B):tab-mark 專職
  身分、breadcrumb 專職路徑。
  - **第一格固定是 rocket**(= `iconCWD`,quit picker 用的同一顆):tab 1 每次
    啟動都開在 launch directory、從不持久化,所以它承接了被移除的 top status
    row 原本要傳達的「這裡是你啟動的地方」。
  - 其餘四格是 Material Design 的動物:好認、單 glyph、彼此不會看混。曾用羅馬
    數字 `Ⅰ`–`Ⅲ`,是 EA-ambiguous 寬度、CJK 字型畫 2 格(§3.2)。
- **上限 `maxTabs = 5` 恆定**:不是美學,是讓 tab bar 寬度與 zoom 分欄數有上
  界、版面可預期。撞上限時 `t` 出 toast 而不是靜默失敗。
- **`t` 開新分頁走 picker**(`goto.go openTabMenu`):Same(目前目錄)/
  Favorites(挑一個最愛)/ Search($HOME 的目錄 finder)。「開新分頁」本身就是
  「要開在哪」的選擇 —— 照 §6.1 判準,那是 menu 不是一次確認。

#### zoom 依實際 tab 數分欄

`z` Zoom(Space menu 的 panel operation)把當前面板展開佔滿全畫面。**有 tab
的面板 zoom 時,把分頁攤成等寬並排欄** —— 且**依當下實際 tab 數分欄**,不是
硬編:`view.go zoomListView` 用 `splitN(w, len(m.tabs))` 依 tab 數等分,`h/l`
在欄間切 focus。所以開 1 個 tab 就佔滿 1 欄、開 5 個就 5 欄;**每一欄各自顯示
自己目錄的 breadcrumb**(breadcrumb 住在面板內部的直接紅利,舊的全域 header
做不到)。`[2]`/`[3]` 各有自己的 zoom(full-screen preview / marks 面板)。
zoom 是 Space menu 的 panel op(不是 `?` 全域),`z` 再按退出、不借用 `Esc`。

---

## §2. 色彩 in filu

### 2.1 配色錨點(catppuccin-mocha)

| 錨點 | 用途 |
|---|---|
| `baseHex` `#1e1e2e`(base) | cursor bar 上的前景字色、亮底 chip 的深字 |
| `focusColor` blue `#89b4fa` | panel border(亮=focus)、tab chip 底、list mark glyph、finder nav-mode cursor |
| `borderDim` surface2 `#585b70` | unfocused chrome |
| `crustHex` `#11111b` | tab-bar recessed 底 |
| `userColor` lavender `#b4befe` | 使用者足跡(§B):breadcrumb 文字、失焦 cursor bar、marks 列 |
| `handColor` subtext1 `#bac2de` | focused cursor bar |
| `dimColor` overlay0 `#6c7086` | 退階 / 次要文字 / 行號 gutter / breadcrumb 的 `/` 分隔 |
| surface0 `#313244` | breadcrumb 下方的分隔線、focused tab bar 的細分隔 |
| 綠 `#a6e3a1` / 黃 `#f9e2af` / 紅 `#f38ba8` | eza perm/size 配色(`x`綠·`r`黃·`w`紅、size 綠、owner 黃);同一套也用在 mark 欄三態 |
| Peach / Red | warning / error override |

**eza 配色來源**:`lscolors_data.go` / `ezadata.go` 是從使用者的 `LS_COLORS`
快照與 eza 原始碼**烘進 binary** 的表,所以 runtime 不需要 `LS_COLORS`,也不
會因終端環境不同而變色。

### 2.2 明度作 z-axis

- **popup**:border 用 `popupLayerColor(layer)` 做 layer 明度插值
  (lavender→sapphire,越上層越亮),不 hardcode(通用 §2.5)。
- **cursor bar**:focused = subtext1(亮)、unfocused = lavender(退成足跡
  色),也是 z-axis 的一種(focus 亮 / 失焦退階)。
- **tab chip**:active = border 色亮底深字、inactive = crust 凹陷底 —— 明度即
  「哪一個在前面」。

> **已移除**:v0.2.8 前 header 另有一套 blue→crust 的**路徑深度漸層**
> (`crumbColorAt` / `crumbTextAt` + WCAG 翻色 + powerline 三角),隨 header 一起
> 刪除。它留下的教訓仍然有效,收在 §8.2 的備考。現在 filu 只有**一套**明度色
> 帶(popup layer),沒有第二套要防混用。

### 2.3 顏色帶專職化 + override

見 §B 表。override 色(Peach/Red)不參與任何 layer scale —— 它們是「跳出階
層之外的警示」,不是階層上的一階(通用 §2.4)。**list 的 perms 欄逐字上色**
(type 藍、`r` 黃、`w` 紅、`x` 綠、`-` dim,`view.go colorPerm`)沿用 eza 語
彙,讓「權限」在 filu 裡讀起來就是 user 在 shell 裡熟悉的那套顏色語言。

---

## §3. 符號語彙 in filu

### 3.1 Nerd Font 是設計、必裝

filu 用 Nerd Font glyph 當視覺語彙(型別 icon、powerline chip cap、tab mark、
mark 勾),是設計的一部分、不做降級(通用 §3.1)。source 裡不放 PUA glyph 字
面,一律用 `string(rune(0x...))` 建構。型別 icon 的對照表
(`ezadata.go` 的 `dirIcon`/`nameIcon`/`extIcon`)由 eza 原始碼產生,解析順序
也照 eza 的 `icon_for_file`。

### 3.2 CJK icon 寬度(CPR 偵測)

CJK Nerd Font(如 Maple Mono NF CN)可能把 file-type icon 畫成 2 格。filu 啟動
時用 **CPR**(cursor position report `\x1b[6n`,`iconwidth_unix.go
DetectIconWidth`、在 `tea.NewProgram` 之前)實測 icon 實際格寬,存入
`iconCells`;`width.go` 的 display-width 層據此保留版面空間。`isWideIcon` 涵蓋
PUA(型別 icon、tab mark);powerline caps `U+E0A0–E0D7`(圓頭 `E0B6`/`E0B4`、
斜線 `E0B8`/`E0B9`)刻意排除(它們單寬)。多數終端 `iconCells=1`、此層 no-op。
**不能靠終端的 East-Asian-Width 全域旋鈕解**:那會連帶改動其他字元的寬度。

### 3.3 控制字元清洗(`safeName`)

檔名可能含控制字元(macOS 自訂圖示檔字面叫 `Icon\r`,含 CR)。畫出原始 CR 會
把游標打回行首、打碎版面。`safeName`(`list.go`)在**顯示時、套 lipgloss 前**
剝掉控制字元(`< 0x20` 或 `0x7f`;剝 ESC 也擋 ANSI injection),檔案操作仍用真
實名。套在所有 name-render 點(list / tree / archive / marks / favorites /
breadcrumb / rename input)。

### 3.4 Surface 標籤(類型訊號 + 內容訊號)

- **panel chrome**:`[N] label`(型別訊號 `[N]` + 內容訊號 label)。
- **tab bar**:`[1]` 用動物 glyph = 位置標記(§1.2)、`[3]` 用
  `Marks`/`Tasks`/`Favorites` 文字標籤(三個不同內容,需要名字);兩者的分隔都
  是斜線 `\`(`E0B8` 實心 / `E0B9` 細)。
- **mark 欄**:三態一格(§B)。
- **欄位表頭兼 sort 指示**:list 的 column header 不在 sort chain 裡就 dim、在
  就加亮 + 升降箭頭(`sortColHeader`)—— 一列兼兩職是刻意的,因為它們講的是同
  一件事(這欄是什麼 / 這欄現在怎麼排)。
- **行號 gutter**:「號碼 + 一空格」(無 `│` 分隔符,對齊 kbu yaml popup);
  preview text/binary 與 yank viewport 共用同一 gutter 形式。
- **popup content 列刻意不放 glyph**(CJK-safe):popup 用 `lipgloss.Width`
  量、對 ambiguous/PUA glyph 會低估,glyph 只擺 title/hint(邊框列)。例:
  breadcrumb popup 的「你在此」層改用 lavender 文字標記、不用 `●`。

---

## §4. 互動 in filu

### 4.1 Core 5 鍵 + 導覽

見 §A.0.Y。五鍵語意在任何 panel / popup 都不變。

- **`h`/`l`** = 切當前 focused panel 的 tab(`[1]` 切目錄分頁、`[3]` 切
  Marks/Tasks/Favorites;`[2]` 無 tab、h/l no-op)。
- **vim 導覽**(`j/k/u/d`)跨 surface 同義;**跳頂 `gg`**(單 `g` 待命等第二
  鍵、對齊 kbu)、`G` 跳底,適用三個面板與所有清單型 popup。
- **`go`** = Goto picker chord(空出 `g` 前綴後接 `o`;`gt` 因 vim 是 go-to-tab
  而放棄)。

### 4.2 Letter hotkey ⊆ Space menu

filu 每個 contextual letter hotkey 都在對應 focus 的 Space menu 現身(§A.1
表)。動作鍵盡量小寫(`o`/`m`/`y`/`r`/`c`/`v`/`a`/`b`/`f`/`s`/`p`);
`D` delete、`O` open-with、`S` sort、`F` favorite-this-dir、`Z` zip、`C` clear
維持大寫 —— 避開小寫衝突(`d` = half-page-down、`o`/`f`/`s`/`z`/`c` 已有其他
語意),同時大寫本身就是「影響範圍更大」的弱訊號。

### 4.3 取消鍵 = `Esc`(finder 清單態的 `q` 例外語意)

`Esc` 通殺:有浮層先關浮層,否則回上層目錄(`list.parent()`)。**兩段式**:
yank viewport 的 visual 模式下,第一次 `Esc` 先退 visual、第二次才關(§6.5)。

**finder 清單態的 `q`**:finder 是「輸入 → Enter 交清單 → j/k 選 → Enter
reveal」。清單態的 `Esc` = **離開 finder**(同其他 popup),`q` = **回到輸入列**
改 query。這裡 `q` 不是「取消 role」的一部分(它只在 finder 清單這個
sub-surface 有「回輸入」的意義),`Esc` 仍是唯一的取消 role,語意一致。

### 4.4 Hotkey discoverability — bracket `[X]label`

Space menu / help 用 `[X]label` 揭露 letter hotkey(`spacemenu.go bracketHotkey`);
單字 chord 也就地 bracket(`[go]to`);純 label 不加 bracket。

> **⚠️ 數字 key 一律前綴、不內嵌**:`bracketHotkey` 對 `0`–`9` 走
> `"[" + key + "] " + label`,不做字面比對。內嵌會把 `432hz` 這種 label 畫成
> `4[3]2hz` —— 數字在 label 裡是**內容**,在 hotkey 裡是**序號**,兩者長得一樣
> 但語意無關。字母才適合內嵌(`[Z]ip`、`[C]lear`)。

### 4.5 panel [2] Preview yank + visual selection

`[2]` Preview 聚焦時 `y` 開一個 **yank viewport**(`detailyank.go`)覆蓋 preview
內容,是「選文字複製」的專用視角:

- vim-style **cursor**(`j/k/g/G/u/d` 移動)+ **`v` 進 visual**、字元級選取。
- 有選取 → `y` 複製選取內容;沒選取 → `y` 複製全部。皆走 OSC 52(可跨 SSH /
  tmux)+ toast 回饋。
- **兩段式 Esc**(§6.5):visual 中先退 visual、再按才關 viewport。
- text/binary preview 帶「號碼 + 空格」的 display-only gutter(§3.4);複製時
  gutter 不進剪貼簿。
- **hard-wrap 的續行不補換行**:preview 為了塞進面板寬會把長行折斷
  (`previewModel.cont` 標記哪幾行是續行),複製時這些折點**不能**變成真的
  `\n`,否則貼出來的 base64 / 長 URL 會斷掉。

這是「同一份內容的另一個視角」(§6 viewport 類),不是新面板 —— 讀時覆蓋 `[2]`、
關掉回原內容。

### 4.6 破壞性動作一律先 confirm

「會改變磁碟或把控制權交出去」的動作,filu 一律先跳 message popup 確認
(`confirmKind`):

| 動作 | confirm 問什麼 |
|---|---|
| `D` Delete(list) | 要丟進系統垃圾桶的項目 |
| `D` Delete(Favorites tab) | 要取消最愛的目錄 |
| `o` Open | 要交給 OS 預設 app 開的檔 |
| `s` Shell | 要在哪個目錄開 `$SHELL` |
| `C` Clear(Marks) | 要清掉幾個 mark |

判準:**代價不可見或不可逆 → confirm**。`Open` 之所以也要問,是因為交給外部
app 之後 filu 就管不到了;`Clear` 之所以要問,是因為 bucket 是慢慢累積起來的、
一鍵歸零沒有 undo。反面:`m` mark / `p` pick / `f` favorite 都是**可逆的一鍵
toggle**,不 confirm —— 對可逆動作加確認只會製造噪音。

---

## §5. Mouse in filu

`(planned)` —— IDEA 規劃沿用通用 §5 的 mouse mapping(與 kbu 同款:左鍵
focus+select、雙擊 Enter、右鍵 Space、滾輪 u/d),**目前尚未 wire**。原則:
mouse 必為 keyboard 的 mapping、不引入新語意(通用 §5)。

---

## §6. 浮層 in filu — Popup Convention

### §6.1 Popup 4 類 taxonomy(+ 三個設計案例)

| 類型 | filu 實例 | 特徵 |
|---|---|---|
| **menu**(選單) | Space menu、Sort picker、**Quit(cd-on-quit)picker**、**Goto / New-tab picker**、**Search chooser**、Open-with picker、Favorites 的 Open-in picker、Breadcrumb popup、**finder**(name / content / goto) | 分 region / 清單、cursor-first、選一個執行 |
| **message**(訊息) | confirm(delete / unfavorite / open / shell / clear marks)、input popup(rename / add / **zip 檔名**)、Toast(yank 回饋) | 短、確認或收一行字 / auto-dismiss |
| **viewport**(捲動視窗) | `?` help、`[2]` yank viewport(`detailyank`) | 可捲、可 cursor / visual selection |
| **pty**(內嵌終端) | `s` Shell(`$SHELL`) | 外部全螢幕程式在 filu 內 render |

全部走共用 `drawPopupBox`(kbu form:title 嵌上邊框、hint 嵌下邊框、內容行夾在
兩條 padding row 之間)。**例外**:yank viewport 與 finder 用
`drawPopupBoxPad(pad=false)` 讓內容貼齊邊框(無前導空白列)。

#### 案例:quit 用 picker、不是單純 confirm

離開 app 大可只跳一個 yes/no confirm。filu 刻意把它做成 **menu(picker)**
(`quit.go` `quitMenu`),因為「離開時 shell 要 `cd` 去哪」**是個選擇、不是一次
確認**:

- picker 列出 distinct 目標 —— **LaunchDir**(啟動目錄,用 rocket glyph 呈現)
  + 各分頁的當前目錄,**去重**(同一目錄開在多個 tab 只列一次,`quitTargets`)。
- 選法照 menu 通則:數字直達 或 `j/k` + `Enter`;`Esc` 留下不退。
- **有任務在跑時**,picker 頂端插一條紅字 warning header(override 色,§2.3)
  —— 這時「離開會中斷 copy/move」需要 user 知情,但仍是「選去哪」+「知道有代
  價」,不是把整個動作降級成 confirm。
- cd-on-quit 本身是 **OS 限制**:子程序改不了父 shell 的 cwd,只有 shell
  builtin 能 —— 靠 `eval "$(filu shell)"` wrapper 讀 filu 寫的暫存檔再 `cd`。

分類判準(通用 §6.1):動作有**多個對象要選** → menu;只需**一次是/否** →
message。quit「選 cd 目標」屬前者,所以是 menu 不是 message。

#### 案例:原生 finder(Search `/` chooser / Goto `go`)

finder 是 filu **自畫的 picker**(`search.go`,snacks/Telescope 形式,**非 fzf
binary**),同一個 UI 帶三種模式,都是 menu 類、都有 preview、都**串流載入**:

| 模式 | 怎麼進 | 比對什麼 | 範圍 | preview |
|---|---|---|---|---|
| **by name** | `/` → chooser 選 `filename` | 檔名 fuzzy(記憶體、匹配品質排序) | 當前 tab 子樹 | 選中檔(從頭) |
| **by content** | `/` → chooser 選 `content` | 內容(`rg --files-with-matches`,去重) | 當前 tab 子樹 | scroll 到命中行 + lavender bar |
| **goto** | `go` → picker 選 `Search`;或 `t` 新分頁的 `Search` | 路徑 fuzzy | `$HOME`、**只列目錄**(含 hidden) | 選中 dir 的 tree |

- **`/` 是一個 chooser、不是兩個熱鍵**:早期 `/`=名稱、`f`=內容各佔一鍵。收攏
  成 `/` → {filename, content} 之後,`f` 才空出來給 Favorite。少一個要記的鍵、
  多一層一眼可見的選擇 —— 這正是 §A.1 想要的方向。
- **⚠️ 不接 fzf binary**:走過 fzf-in-PTY(彩色 rg + 每鍵 reload + preview 把
  vt10x 畫爆、root 改 Home 掃整棵卡死)—— native picker 解掉這兩類 bug。fzf 的
  **串流概念**有偷師、原生實作(見 §7.2)。
- **串流載入**:`fd`(缺退 Go walk)一吐就顯示、**依 fd 走訪序、不排序**,首批
  近乎立即、載入中可濾;撞 `finder_cap`(預設 50000、`config.yaml` 可調)停;
  `ignore_dirs`(預設含 `node_modules`、OrbStack、Go module cache 等)整棵跳過。
- **絕對路徑錨定**:query 以 `/` 或 `~/` 開頭時,finder 改成對那條絕對路徑
  fuzzy(錨在最深的存在目錄、往下 depth-3),所以 home 之外的目錄也搆得到。
- **goto 掃 hidden 目錄**:`fd --type d --hidden`、`walkDirFiles` 也不跳
  dotdir —— `~/.m2` 這種找不到曾是**兩層 bug**(code 不掃 hidden × 該目錄在
  `ignore_dirs` 黑名單)。噪音靠黑名單擋、不靠「不掃」擋。
- **modal + Esc/q**:輸入態 → Enter 交清單 → Enter reveal 到 `[1]`(cd 到該檔/
  目錄);清單態 `Esc` 離開、`q` 回輸入(§4.3)。nav 態的 cursor 轉藍
  (`focusColor`),與輸入態一眼可分。
- **mtime 是爛訊號的教訓**:Goto 一度用 mtime 排序,但 mtime 追的是「OS/工具碰
  過」不是「你想跳過去」(把某 dir 從 Finder sidebar 移除反而把它排到最前),
  全砍、改 fd 走訪序 + 靠打字找。想要真 recency 只能 zoxide 式磁碟快取(未做)。

#### 案例:Zip 打到 temp,再走既有的落地路徑

`[3]` Marks 的 `Z` 把 land 子集打成一個 zip。**輸出位置固定是
`os.MkdirTemp("", "filu-zip-")`**,不是任何工作目錄 —— 因為使用情境是「打包完再
搬去某個 `[1]` 的目錄」,**輸出位置不等於目的地**。

- 打完之後,任務用 `landMsg.produced` 回報產物,`handleLandMsg` 把 zip **加進
  bucket 並設成唯一 pick** —— 接著用既有的 `c` / `v` 落地就好。**不另造一套
  「送到哪裡」的機制**。
- 檔名**每次都問**(`inputZip` popup),由 `suggestZipName` 預填:單一項目去副
  檔名、同目錄的一組用該目錄名、跨目錄退回時間戳。輸入值過
  `zipFileName`(取 basename + 補 `.zip`),所以打 `../../etc/passwd` 也只會變成
  `passwd.zip`、跑不出 temp 目錄。
- 打包範圍就是 `landItems()` —— 有 pick 打 pick、沒 pick 打整桶,**與 Copy /
  Move here 同語意**,不新增第三種規則。
- 已提過並否決的輸出位置:active tab 的目錄(污染一個可能只是中途站的目錄)、
  picks 的共同父目錄(跨目錄時退化成 `/` 或 `~`)、`config.yaml` 開一個
  `zip_dir` 旋鈕(需求還沒出現就先做設定)。

### §6.2 開關動畫 — PopupAnimator

所有 popup 用 `popupAnimator`(line→expand 開、reverse 關),各自獨立 animator
name 避免 tick 互撞(通用 §6.2)。

> **⚠️ 接線坑**:新加一個 popup 時,**必須**把它的 `handleTick` 接進 `Update`
> 的 `AnimTickMsg` batch,否則一開就 hang 在動畫第一格。直接把 `anim.state` 設
> 成 open 的測試**抓不到這個 bug** —— 要用真的 tick 驅動才測得出來。

### §6.3 Border 色 = layer 明度

popup border 用 `popupLayerColor(layer)`,不 hardcode(§2.2 / 通用 §6.3)。

### §6.4 Stack 預設保留 source

popup 疊 popup 預設保留底層 source。**Context-shift 例外**:PTY(`s` shell)
佔滿畫面、退出後才回 list;yank viewport / finder 覆蓋所屬面板內容。

### §6.5 取消鍵通殺 + auto-dismiss

`Esc` 關任何 popup;toast 有計時 auto-dismiss(id 世代守衛防舊 timer 誤關新
toast)。yank viewport 的 `Esc` 兩段式(先退 visual)、finder 清單態 `Esc` 離開
`q` 回輸入(§4.3)。

### §6.6 Menu region cursor-first

Space menu 分 item-region(對 cursor item)/ panel-region(對當前 panel),
cursor-first 排序(`groupedMenu`);單一類動作時不分 region、直接列(通用 §6.6)。

---

## §7. 時間軸 UX in filu

### 7.1 Marks bucket 兩層 pick(list mark / [3] Marks pick)

marks bucket 是**延遲決定 cp/mv** 的模型(對齊 macOS Finder Cmd+V /
Cmd+Opt+V),分**兩層**:

- **list `m` Mark(進 bucket)**:游標檔按 `m` → 丟進 bucket(常駐的 reference
  list、不是 mode)。list 對「在 bucket 裡」的檔案在最前面固定一格畫
  `markGlyph`;沒在 bucket 的該格留白 —— **固定 `mark + space + icon` 版位**,
  mark/unmark 只換那一格 blank↔glyph、**icon 不左右位移**(`list.go`)。等同一
  個 in-place multi-select。
- **`[3]` Marks `p` Pick(進 land 子集)**:focus 進 `[3]` Marks、對某項按 `p`
  → 標它進「下一次 land 的子集」,用 `markPickGlyph`。
- **Land(cp/mv 落地才決定)**:有 land-subset → 只對 picked;沒 pick → 對整個
  bucket(`landItems()`)。cp 不改 bucket(可連續複製到多目錄)、mv 更新路徑保
  持有效。**`Z` Zip 吃的也是 `landItems()`** —— 同一套子集語意,三個動作共用。
- **兩層 pick 兩個 glyph**:「在 bucket」(成員)vs「在 land 子集」(子集)是兩
  個狀態,分別用 `f0b14` / `f05d`,不共用(§B)。
- **`C` Clear 只丟選取、不動檔案**:清空 bucket 與 picks,磁碟上的檔案原封不
  動;先 confirm(§4.6)。

### 7.2 Streaming 不退階(Task 進度 + finder 載入)

- **Task 進度**:落地(cp/mv/zip)跑 goroutine,進度經 channel → `tea.Msg` 餵回
  `[3]` Tasks tab 即時更新(running/done/error),即使 `[3]` 失焦也不退階
  (通用 §7.2)。任務存 `state.yaml`,`action` 是自由字串(加一種新任務不必動
  白名單)。
- **finder 載入**:finder 邊掃邊串流出清單(`fd` 一吐就顯示、不等整趟走完、不
  排序),首批近乎立即、載入中可濾 —— 這是 fzf 串流概念的原生版(§6.1)。
  「資訊正在到達」屬 streaming、不 dim。

---

## §8. Panel chrome in filu

### §8.0 三件套(零頂部 content row)

| 件 | filu 實作 |
|---|---|
| **border title** | powerline chip(`singleChip`,`[N] label`);有 tab 的面板用 tab bar(`tabBar` / `tabBarPad`) |
| **tab bar** | `[1]` 目錄分頁(動物 glyph)、`[3]` Marks/Tasks/Favorites(文字);§1.2 |
| **border hint** | 面板下邊框的 key legend —— `[1]`:`enter into  esc back  jkud move  hl switch tab`;`[3]` Marks:`m mark   c copy   v move`;`[3]` Favorites:`o open in   D remove`。popup 同一位置放自己的提示(如 yank viewport 的 `v:visual  y:copy  Esc:close`) |
| **panel 內第一列** | `[1]` 的 breadcrumb + 分隔線(§8.2) |

**沒有任何全域頂部列** —— 早期的 header 麵包屑與 top status bar 都已收進面板
(§8.2 / §8.3),`ptyChromeRows = 0`(shell popup 從第一列吃到底)。

### §8.1 Focus 訊號

box 用 border 色(structural blue 亮 / surface2 暗)+ 字重表示 focus / unfocus。
`[2]` Preview 是**參考視角**(邊看別的面板邊讀),失焦**不 dim**、保色;list 的
cursor bar 失焦退成 lavender 足跡色(§2.2)。

### §8.2 breadcrumb = panel [1] 的第一列(純文字)

`[1]` 的第一列是這個 tab 的 breadcrumb(`header.go crumbRow`),下面接一條
surface0 的 `─` 分隔線,再才是檔案清單:

- **純文字、不是 chip**:lavender 路徑段 + dim `/` 分隔,**沒有背景色塊**。理
  由是正上方的 border 就是一排 tab chip —— 兩排 chip 疊在一起會互相打架;純文
  字讓 breadcrumb 讀起來就是「一條路徑字串」。
- **不帶 tab 標記**:哪個 tab 由正上方的 tab bar 講,breadcrumb 只講路徑
  (§B 分工)。
- **絕對路徑 root 特判**:`pathSegments` 把 `/` 當成獨立首段,`renderCrumb` 讓
  它直接充當第一個分隔符,避免畫出 `//`。
- **永遠單行**:過長走 §1.2 的漸進四階,最後由 `truncate` 硬截。
- **住在面板裡的紅利**:zoom 成多欄時,**每一欄各自顯示自己的路徑** —— 全域
  header 時代只能顯示 active tab 那一條。

> **⚠️ 拆頂部列留下的兩層後遺症**(v0.2.8 埋、v0.3.0 修一半、後續補完):
>
> 1. **`listBody` 多吃了 2 列**(breadcrumb + 分隔線),但 `listRows()` 還停在舊
>    的 `-3`(border 2 + 欄位表頭 1),cursor 因此能移到最後一列**之下**。修法:
>    抽 `const listChromeRows = 2`,讓 `listBody` 與 `listRows()` 共用同一常數。
> 2. **`midHeight()` 還在扣已經不存在的兩列** —— 它寫 `height - 3`(header +
>    status + footer),但 `View()` 早就只扣 footer 的 `height - 1`。於是高度預算
>    比實際 render **少 2 列**:panel 畫得出來的最後 1–2 列,cursor 永遠到不了
>    (`detailRows()` 同源,preview 也捲不到最後幾行)。**zoom 更糟**:zoom 時面
>    板吃滿整個 region,但 `listPanelHeight()` / `detailRows()` 仍算 2/3,整整
>    **少了三分之一**。修法:`midHeight()` 改回 `height - 1`,兩支預算函式各自加
>    上 zoom 分支。
>
> **教訓一:凡是在 panel body 上方加/減裝飾列,必定要同步 cursor 的 row 預算**
> —— 兩邊各自寫死數字就是漂移的種子。
>
> **教訓二:回歸測試要對「真正畫出來的東西」斷言,不要自己重算一次高度。** 第一
> 版 `TestListRowsMatchesListBody` 拿 `listPanelHeight()` 當高度去渲染 body 再跟
> `listRows()` 比 —— 兩邊同源,自己跟自己比永遠綠,`midHeight()` 與 `View()` 對
> 不上完全看不見。改成 `TestListRowsMatchesRender` / `TestDetailRowsMatchesRender`
> 之後:走 `middleView(w, height-1)`(`View()` 傳的同一個值)、數畫面上真正出現
> 幾列,四種尺寸(含 zoom)一次全露餡。

> **備考(已移除的 powerline header)**:v0.2.8 之前 header 是一條全域的
> powerline 麵包屑,用 blue→crust 漸層 + 實心三角 `` 造路徑深度階層。它沉澱
> 出兩條仍然有效的通則:(1)**「看得見的三角分隔」⟺「相鄰兩段要有夠大色差」**
> —— 實心三角的斜邊**就是**兩段背景的色界,兩段近同色就切不出三角,所以低對比
> 的 lavender→sapphire 不能拿來當路徑漸層;(2)**亮→暗漸層必須讓文字色隨背景
> 翻**,且要用 **WCAG 對比度**擇優、不要用 Rec.601 明度中點(後者翻得太早)。
> 現在的斜線 `\` 分隔同樣吃第 (1) 條。

### §8.3 目錄狀態收進 list 的欄位

早期有一條 top status bar 顯示 active tab 目錄本身的 `perm · owner:group ·
item count · hidden count` + 磁碟餘量。v0.2.0 之後這些資訊**收進 list 的多欄**
—— 每一列自己帶 mtime / owner / perms / size,比「只講當前目錄一列」資訊量更
高,也不用再花一整條螢幕列。

- **顯示順序**:`mark | mtime | owner | perms | size | name`;窄化時依
  owner → size → mtime → perms → mark 的順序退場(§1.2)。
- **eza 配色**(§2.3):perm 逐字(type 藍、`r` 黃、`w` 紅、`x` 綠、`-` dim)、
  owner 黃、size 綠。
- **每 frame 零 I/O**:欄位在 `listModel.reload()` 一次算好(單次 `stat`,
  **非遞迴**),render 只讀快取字串。**不放**遞迴 content size(會 walk 整棵子
  樹、卡);目錄的 size 欄畫 `-`。
- **表頭兼 sort 指示**(§3.4):column header 依 per-directory 的 sort chain
  dim / 加亮 + 箭頭。

---

## §9. 實作狀態

### 已落地(v0.3.0)

- **3-panel layout**(`[1]` list · `[2]` Preview · `[3]` Marks|Tasks|Favorites,
  grid `[1][1][2] / [3][3][3]`,上 2/3 : 下 1/3、上排 2:1)+ footer;
  `z` zoom(依動態 tab 數分欄);`w < 72` 窄寬 fallback。
- `[1]` 動態 1–5 目錄分頁(`t` picker / `w`、動物 glyph 標記)、`h/l` 切;
  面板內 breadcrumb;vim 導覽 + `gg`/`go` chord;`Enter` 進目錄;`Esc` back;
  多欄清單(mark/mtime/owner/perms/size/name)+ per-directory sort chain。
- `[1]` 動作:Open `o`(confirm)、Open with `O`、Mark `m`、Yank `y`、
  Rename `r`、Add `a`、Delete `D`(→系統垃圾桶,confirm)、Favorite `f`/`F`、
  Copy `c` / Move here `v`(async land)、Sort `S`、Shell `s`(內嵌 PTY,
  confirm)、Hidden `.`、Search `/`、Goto `go`、Breadcrumb `b`。
- `[2]` Preview 分類(dir tree / archive / image / text with chroma / binary
  hex)+ PDF 文字抽取;yank viewport(cursor + `v` visual,hard-wrap 續行不補
  換行)。
- `[3]` **Marks**(`p` land-subset pick、`m` unmark、`y` yank、`Z` zip、
  `C` clear)/ **Tasks**(async 進度、`D` 移除紀錄)/ **Favorites**
  (`o` open-in picker、`D` unfavorite(confirm))。
- **marks bucket 兩層 pick、延遲 cp/mv**;zip 打到 temp 後成為唯一 pick、接既
  有落地路徑。
- **finder** 三模式(`/` chooser 的 filename / content、`go` goto)原生 picker、
  全 preview、串流載入、絕對路徑與 `~/` 錨定、goto 掃 hidden 目錄。
- Popup 全套(menu/message/viewport/pty)+ animator + layer 色;quit picker;
  breadcrumb popup;input popup(rename / add / zip 檔名)。
- fsnotify 即時刷新;`state.yaml` session 持久化(tabs / marks / favorites /
  tasks / per-dir sorts);`config.yaml`(`finder_cap` / `ignore_dirs` /
  `open_with`,遵守 `XDG_CONFIG_HOME`);cd-on-quit(`filu shell` wrapper)。
- CLI:`filu [path]`(目錄則開該目錄、檔案則開父目錄並把游標停在該檔,必要時
  自動顯示 dotfile)、`filu shell`、`filu version`、`filu iconwidth`。
- CJK icon 寬度 CPR 偵測;`safeName` 控制字元清洗。
- 平台:unix-first(build tag),`GOOS=windows` 刻意不編;`CGO_ENABLED=0`。
- 發佈:goreleaser(4 個 unix target)+ Homebrew **formula** + `install.sh`
  + GitHub Actions。

### 規劃(planned)

- **Mouse**:沿用通用 §5 mapping(與 kbu 同款)。
- **每列錯誤 `!` 前綴**:broken symlink / 無權限 / 上次操作失敗(目前僅面板層
  error note)。
- **zoxide 式磁碟快取索引**(給 goto 真 recency,只在串流不夠時做)。
- chmod / extract、真圖(kitty/sixel)、sort filter、續傳:見 IDEA。

---

## 附錄 — filu hotkey 全表

### Core key(跨 surface 不變、5 個)

| 鍵 | 語意 |
|---|---|
| `Tab` / `1`-`3` | 切面板 |
| `Enter` | 進目錄 / popup 確認(**不開檔**) |
| `Esc` | 關浮層 / 回上層 |
| `Space` | contextual 入口(Space menu) |
| `?` | non-contextual 入口(help popup) |

### Contextual letter hotkey(在 Space menu 現身)

| Focus | 鍵 | 動作 |
|---|---|---|
| `[1]` List | `o` `O` `m` `y` `r` `D` `f` `F` | Open / Open with / Mark / Yank / Rename / Delete / Favorite item / Favorite this dir |
| `[1]` List(panel) | `c` `v` `/` `go` `b` `t` `w` `a` `S` `s` `.` `z` | Copy / Move here / Search / Goto / Breadcrumb / Tab / Close tab / Add / Sort / Shell / Hidden / Zoom |
| `[2]` Preview | `y` `z` | Yank viewport / Zoom |
| `[3]` Marks | `p` `y` `m` `Z` `C` `l` `z` | Pick / Yank / Unmark / Zip / Clear / Switch tab / Zoom |
| `[3]` Tasks | `D` `l` `z` | Delete 紀錄 / Switch tab / Zoom |
| `[3]` Favorites | `o` `D` `l` `z` | Open in / Unfavorite / Switch tab / Zoom |

### Non-contextual(全域)

| 鍵 | 動作 |
|---|---|
| `?` | help popup |
| `q` | cd-on-quit picker(離開並帶目錄回 shell) |
| `Ctrl+C` | 硬退 |

### Vim-style 導覽(跨 surface 同義)

| 鍵 | 動作 |
|---|---|
| `j` / `k` | 上下 |
| `gg` / `G` | 頂 / 底(`gg` = 單 `g` 待命的 chord) |
| `u` / `d` | half-page 上 / 下 |
| `h` / `l` | 切當前 focused panel 的 tab |
| `go` | Goto picker(g-prefix chord) |

---

## 結語

filu 是通用設計原則在 filesystem domain 的一個實現(kbu 是 K8s domain 的另一
個)—— 兩者平行、共用同一套通用原則,不是誰派生自誰。filu 落地的原則骨架:
core-key 5 個不變、Space + `?` 雙軌揭露、popup 四類 taxonomy、明度 z-axis、
marks bucket 延遲決策、破壞性動作先 confirm。filu 自己長出來的幾個實作案例 ——
隨寬逐階縮字(breadcrumb / favorites 共用)、動態 tab 與 breadcrumb 分工、quit
用 picker、原生串流 finder、兩層 pick 兩個 glyph、mark 三態合併成單格、依動態
tab 分欄的 zoom、preview yank visual、Zip 打到 temp 再走既有落地路徑 —— 都收
在對應章節。凡設計決定以 `.forge/meta/IDEA.md` 為準;凡通用原則以
**VTP**([Vulcan's TUI Design Principle](https://github.com/vulcanshen/thoughts/blob/main/vtp.md))為準。本文件隨
實作演進更新,不宣稱未落地的行為。

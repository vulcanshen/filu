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
> **狀態標記**:本文件描述**當前已落地**的實作。尚未完成者標 `(planned)`,
> 不宣稱未落地的行為。實作狀態總表見 §9。

本文件另收錄幾個 filu 打磨過程沉澱下來的**實作案例**(隨寬逐階縮字、動態
tab 與 header 連動、quit 用 picker 而非 confirm、原生 finder、carry / carries
兩層 pick、依動態 tab 分欄的 zoom、preview yank visual、header 漸層背景/前景
選色),分散在對應章節,並在 §8 集中談 header chrome。

---

## §A. Implementation in filu

通用 §A.0(score)+ §A.1(contextual track)+ §A.2(non-contextual
track)在 filu 的具體實現。

### §A.0 filu score 對照

| 軸 / 結果 | filu 值 | 計算 |
|---|---|---|
| **X. 揭露程度** | ~1.0 | Space menu 列出當前 focus 的 contextual 動作 100%、`?` help popup 列出全域動作 100%。以 user 學習單位計:`e` edit 對所有文字檔通用算 1 個 action、`Enter` open 對所有型別通用算 1 個 |
| **Y. core-key role 數量** | 5 | `Tab`(focus 切換)/ `Enter`(確認·進入·開檔)/ `Esc`(取消·回上層)/ `Space`(contextual 入口)/ `?`(non-contextual 入口)。`Ctrl+C`(硬退)與 `q`(cd-on-quit picker)不另計 role —— 見 §A.0.Y |
| `min(1, 5/Y)` 係數 | 1.0 | Y = 5、無 penalty |
| **Score** | `~1.0 × 1.0` = **~100%** | 不靠事先學就能用 |

filu 的定位不是「再做一個 yazi」,而是「**第一次開就能不看文件開到底**」的
檔案管理器 —— letter hotkey 是加速捷徑、不是必經之路,光靠 `Space` + `?`
就能走完該 focus 的所有動作。

### §A.0.Y filu core-key 集合(5 個)

| Core-key | filu 語意 | 對應通用條款 |
|---|---|---|
| `Tab` | focus 切到下個 panel(`1`–`5` 直達 alias) | §4.1 |
| `Enter` | 進入目錄 / 開檔(交 OS) | §4.1 |
| `Esc` | 關閉最上層浮層 / 回上層目錄(LIFO back) | §4.3 |
| `Space` | §A.1 contextual 入口(Space menu) | §A.1 |
| `?` | §A.2 non-contextual 入口(help popup) | §A.2 |

5 個,剛好通用 §A.0.Y 上限。letter hotkey(`p`/`y`/`e`/`c`/`m`/`r`/`a`/
`D`/`S`/`P`/`.`/`z`/`/`/`f`/`b`)與導覽 chord(`gg` 跳頂、`go` goto)
**不算 core-key**,是入口內動作的加速捷徑。

**`Ctrl+C` 與 `q` 為何不各記一個 role**:`q` 不是「取消」—— 它開 cd-on-quit
picker(選離開時要 `cd` 去哪),語意屬「離開 app 並帶目錄回 shell」,是一個
全域動作(列在 footer + help)。`Ctrl+C` 是逃生硬退,與 `Esc`(退浮層 / 退
目錄)語意不同、不重疊,屬 emergency exit,不佔 core-key role 上限。取消
role 由 `Esc` 單獨承載(§4.3)。

**`gg` / `go` chord 為何不 +Y**:`gg` 是 vim 跳頂(單 `g` 待命等第二鍵、對齊
kbu)、`go` 開 goto finder —— 兩者都是既有動作的**加速捷徑**、走 letter-hotkey
層,不是新的 core-key role。實作:單一 `AppModel.pendingG` 掛在主 switch 的
chokepoint(所有 popup return **之後**、只管主面板),`gg` 落既有 `case "g"`、
`go` 呼 `handleListKey("go")`。

### §A.1 Contextual track in filu — Space menu

每個 panel focus 都接 `Space`,列出「這個 focus 此刻能做什麼」。入口自身在
**footer** 揭露(`view.go footerBar`):

```
 space menu   ? help   tab/1-5 panels   q quit
```

user 第一次開、沒看 README,從 footer 就知道按 `Space` 會跳選單。

filu 各 focus 的 Space menu(`app.go buildSpaceMenu`,`groupedMenu` 分
item-region / panel-region、cursor-first,見 §6.6):

| Focus | item-region 動作 | panel-region 動作 |
|---|---|---|
| **[1] Places** | Jump(`Enter`)、UnPin(`P`,僅 Pinned 項) | — |
| **[2] List** | Pick `p`、Yank `y`、Edit `e`、Rename `r`、Delete `D`、(Pin `P`,僅目錄) | Copy `c`、Move `m`、Search `/`、Find `f`、Goto `go`、**Breadcrumb `b`**、Tab `t`、Close tab `w`、Add `a`、Sort `S`、Hidden `.`、Zoom `z` |
| **[3] Preview** | Yank `y`(開 yank viewport) | Zoom `z` |
| **[4] Carries** | Pick `p`、Yank `y`、Delete `D` | Tab `l`、Zoom `z` |
| **[4] Tasks** | Redo `R`、Delete `D` | Tab `l`、Zoom `z` |
| **[5] Meta** | Yank `y` | Zoom `z` |

**完整性 audit**:新增一條 contextual 動作,必須同步在對應 focus 的 Space
menu 加 entry,不能只綁 letter hotkey,否則是原則破洞。反例警覺:`Copy`/`Move`
只在 bucket 有內容時才出現在 menu —— 這是 cursor-state gating,不是隱藏。

### §A.2 Non-contextual track in filu — `?` help popup

`?` 在任何 surface 跳出全域動作清單(`helppopup.go`)。入口自身同樣在 footer
揭露(`? help`)。

filu 是檔案管理器,domain 動作幾乎都是 contextual(對 cursor item / 對當前
tab),所以 **§A.2 全域軌很薄**:

| 全域動作 | key | help popup 內列 |
|---|---|---|
| 說明 | `?` | ✓ |
| 離開(cd-on-quit picker) | `q` | ✓ |
| 硬退 | `Ctrl+C` | ✓ |
| 切面板 / 切 tab / 導覽 | `Tab`·`1-5`·`h/l`·`Enter`·`Esc` | ✓(core-key 揭露) |

沒有 kbu 那種 namespace / context 全域 toggle —— filesystem 的「當前位置」
是每個 tab 各自的 CWD(contextual),不是一個全域狀態。

### filu contextual / non-contextual 動作邊界(audit 用)

| 動作 | contextual? | track |
|---|---|---|
| 對 cursor item 做 Pick/Yank/Edit/Rename/Delete/Pin | ✓ 對 cursor row | §A.1 |
| 對當前 tab 做 Add/Sort/Hidden/Copy·Move-land/Search/Find/Goto/Breadcrumb/Tab/Close | ✓ 對當前 tab | §A.1 |
| Preview/Meta yank·zoom | ✓ 對 cursor item 的視角 | §A.1 |
| Help / Quit | ✗ 全域 | §A.2 |

---

## §B. 元素專職化 in filu

| 元素 | 專職語意 | 不准兼職 |
|---|---|---|
| **Lavender**(`userColor`) | 使用者足跡:Pinned 項、unfocused panel 的 cursor bar、breadcrumb popup「你在此」層 | panel / popup border、path 漸層 |
| **Subtext1**(`handColor`) | focused panel 的 cursor bar(「當前的手」)、status bar 數值 | 其他狀態 |
| **綠 `#a6e3a1`** | 「已選 / 已進下一步」訊號:[2] carried 勾、[4] Carries land-subset 勾、eza executable-bit、perm `x` | 純裝飾 |
| **Peach / Red** | warning / error override(quit 有任務跑的紅字、Task error、perm `w`) | 不參與 popup layer scale |
| **Panel border 色** | structural(blue 亮=focus / blue 暗=unfocus) | 不拿去做 user state |
| **Popup border 色** | popup layer 明度(`popupLayerColor`,lavender→sapphire) | 不 hardcode、不與 header 漸層混用 |
| **Header path 漸層**(blue→crust,`crumbColorAt`) | 路徑**深度**階層(root→current) | 不用 popup layer scale、不與 structural blue 混用 |
| **eza 型別色**(`fileColor`) | 檔案型別 / executable-bit | — |
| `pickGlyph` `f0bf3` | panel [2]「在 carry bucket」 | 不當 panel [4] 的 land-subset 勾 |
| `carryPickGlyph` `f05d` | panel [4] Carries「在 land 子集」 | 不當 panel [2] 的 bucket 勾 |
| `[X]label` bracket | letter hotkey discoverability | 純 label 不加 bracket |
| `Esc` | 關閉 / 退出 | 永遠不當「確認」 |
| Nerd Font 型別 glyph | 型別訊號(這列是什麼型別) | 不當熱鍵 signal、不當純裝飾 |

**兩個 §B 實例值得點名**:

- **兩種 pick 勾分家**(`carry.go`):panel [2] 的 `p` 標「這個檔**在 carry
  bucket 裡**」用 `pickGlyph`(f0bf3,filled checkbox-circle);panel [4] Carries
  的 `p` 標「這個 bucket 項**被選進下一次 land 的子集**」用 `carryPickGlyph`
  (f05d,check-circle)。兩者是**不同語意的兩個狀態**,若共用同一 glyph,user
  會分不清「在 bucket」與「在 land 子集」—— 所以刻意分成兩個 glyph、不混用。
- **兩套顏色階層不混**:popup 巢狀深度用 `popupLayerColor`(lavender→sapphire);
  header 路徑深度用 `crumbColorAt`(blue→crust)。兩者都是「用明度/色相編碼階
  層」,但語意不同(浮層 z-axis vs 路徑深度),各佔一套色帶,互不借用(見 §2.2
  / §8.2)。

---

## §1. 空間結構 in filu

### 1.1 版面 grid(5 panel + 兩條頂部 content row)

```
  Ⅱ ⟩ ~ ⟩ Documents ⟩ sideproj ⟩ filu             ← header:powerline 麵包屑
 drwxr-xr-x  vulcan:staff  42 items · 3 hidden  …free  ← top status bar(目錄狀態)
┌──────────┬──────────────────┬──────────────────────┐
│ [1] pin  │ [2] list         │ [3] Preview          │
│  Places  │  (1–3 目錄分頁)  │   (右欄上 2/3)        │
│  Pinned  │                  ├──────────────────────┤
├──────────┴──────────────────┤ [5] Meta (右欄下 1/3)│
│ [4] Carries │ Tasks         │                       │
└─────────────────────────────┴──────────────────────┘
 space menu   ? help   tab/1-5 panels   q quit         ← footer(content row)
```

grid `[1][2][3] / [1][2][3] / [4][4][5]`:左欄 `[1]` pin + `[2]` list 並排
上 2/3、`[4]` carry 橫跨 `[1]+[2]` 兩欄的下 1/3;**右欄鏡射左欄的 2/3 : 1/3
切分** —— `[3]` Preview 上 2/3、`[5]` Meta 下 1/3。頂部兩條**獨立 content row**
(header 麵包屑 + status bar)、底部一條 footer,夾住中間帶邊框的 panel 區。
`view.go normalMiddle` 用 `joinH`/`joinV`(display-width aware)組版;所有列
的高度統一由 `midHeight() = height − 3`(header + status + footer)推導。

> **早期演化**:原本 `[3]` 是「Preview / Meta 兩 tab 的右側全高面板」。後來
> 拆成 `[3]` Preview + `[5]` Meta 兩個常駐面板(right column 鏡射左欄),不再
> 有 tab 切換 —— 一元素一語意:preview 與 meta 是兩個並存對象,該 split 不該
> 塞進同一面板 tab(§B / 通用「同對象多視角→tab、不同對象並存→split」)。

### 1.2 窄寬可用、Width stability、與「隨寬逐階縮字」

- **窄寬**:`w < 72` 時放棄 grid,只顯示 list 單欄(`view.go normalMiddle`);
  `z` Zoom 是任何面板的逃生艙(見 §1 下方 zoom)。
- **Footer / status bar 行數固定**:footer N=1、status bar N=1,選定即鎖死、
  絕不 reflow(通用 §1.3)。status bar 的欄位在目錄載入時算好、render 只讀
  快取字串,不會因內容多寡改高度(見 §8 status bar)。

#### 隨寬逐階縮字(header 麵包屑 + panel [1] Pinned 共用一套)

動態文字要塞進固定寬度的 slot 而**不讓寬度跟內容綁動**(通用 §1.2 Width
stability)。filu 對「一條路徑要縮進 N 格」用一套**漸進三階**演算法,header
麵包屑與 panel [1] Pinned 項**共用同一個 helper**(`header.go fitPathSegments`):

1. **放得下 → 全名**:`~/Documents/sideproj/filu`。
2. **不夠 → 從前段起逐段縮成首字元**(current/末段永不縮):
   `~/D/sideproj/filu` → `~/D/s/filu`。一次縮一段、縮到剛好放得下就停,盡量
   保留細節。
3. **還不夠 → 中間 `…`,只留 root + 末段**:`~/…/filu`。
4. **極窄 → 末段單獨呈現**(由 `padDisp` 硬截兜底)。

`fitPathSegments(segs, fits)` 是**泛型**:傳入不同的 `fits` predicate 就能在
不同度量下複用 —— header 量的是 **powerline 渲染寬**(`renderCrumb`),panel [1]
量的是 **plain 字串寬**(`joinSegs`)。同一套三階邏輯、兩種量測。**別再各寫一
份**:兩處若各自實作,縮法會漂移、user 會覺得「header 跟 pin 的縮法不一樣」。

#### 動態 tab(1–3)與 header 連動、上限恆定

panel [2] 的目錄分頁是**動態 1–3 個**(`t` 在當前 tab 的目錄開新分頁、`w` 關
active、至少留 1;都在 Space menu、標籤 `Tab`/`Close tab`)。這裡有兩個 Width
stability 決定:

- **tab 標籤 = 羅馬數字 `Ⅰ`/`Ⅱ`/`Ⅲ`**(`view.go tabNumeral`),**不是** dir
  basename。理由:basename 長度不定、會讓 tab bar 寬度跟內容綁動;羅馬數字是
  **固定寬位置標記**,tab 數在 1–3 之間增減、bar 寬度階梯穩定。**「這個 tab
  在哪」由 header 麵包屑承載**(header 永遠顯示 active tab 的完整路徑)——
  tab bar 只負責「有幾個分頁、哪個 active」,path 資訊外包給 header。這是分工
  (§B):tab-label 專職位置、header 專職路徑。
- **上限 `maxTabs = 3` 恆定**:不是美學,是讓 tab bar 寬度與 zoom 分欄數有上
  界、版面可預期。
- **gotcha**:`nf-md-roman_numeral_*` 在 Nerd Fonts 3.4.0 不齊(缺 1/5),改用
  Unicode 羅馬數字 `U+2160–2162`;這三碼是 EA-ambiguous 寬度、CJK 字型畫 2 格,
  已加進 `isWideIcon`,否則破框(見 §3.2)。

#### zoom 依實際 tab 數分欄

`z` Zoom(Space menu 的 panel operation)把當前面板展開佔滿全畫面。**有 tab
的面板 zoom 時,把分頁攤成等寬並排欄** —— 且**依當下實際 tab 數分欄**,不是
硬編 3 欄:`view.go zoomListView` 用 `splitN(w, len(m.tabs))` 依 tab 數等分,
`h/l` 在欄間切 focus。所以開 1 個 tab 就佔滿 1 欄、開 3 個就 3 欄。`[3]`/`[5]`/
`[4]` 各有自己的 zoom(full-screen preview / meta / carries|tasks)。zoom 是
Space menu 的 panel op(不是 `?` 全域),`z` 再按退出、不借用 `Esc`。

---

## §2. 色彩 in filu

### 2.1 配色錨點(catppuccin-mocha)

| 錨點 | 用途 |
|---|---|
| `baseHex` `#1e1e2e`(base) | cursor bar 上的前景字色、亮底 chip 的深字 |
| structural blue `#89b4fa` | panel border(亮=focus / 暗=unfocus);也當 header 漸層的 root 端 |
| crust `#11111b` | tab-bar recessed 底;也當 header 漸層的 current 端 |
| `userColor`(lavender `#b4befe`) | 使用者足跡(§B) |
| `handColor`(subtext1) | focused cursor bar、status bar 數值 |
| `dimColor`(overlay0) | 退階 / 次要文字 / 行號 gutter / status bar 單位字 |
| 綠 `#a6e3a1` / 黃 `#f9e2af` / 紅 `#f38ba8` | eza perm/size 配色(`x`綠·`r`黃·`w`紅、size 綠、owner 黃) |
| Peach / Red | warning / error override |

### 2.2 明度作 z-axis + header 的第二套深度漸層

- **popup**:border 用 `popupLayerColor(layer)` 做 layer 明度插值
  (lavender→sapphire,越上層越亮),不 hardcode(通用 §2.5)。
- **header 路徑深度**:另有一套**獨立**的 blue→crust 漸層(`crumbColorAt`),
  編碼「路徑深度」而非「浮層深度」。兩套色帶專職化、互不借用(§B)。header
  漸層的完整幾何(為什麼是 blue→crust 這個跨度、為什麼靠明度差而不是分隔符)
  見 §8.2 —— 那是本 doc 最花篇幅的一個實作案例。
- **cursor bar**:focused = subtext1(亮)、unfocused = lavender(退成足跡
  色),也是 z-axis 的一種(focus 亮 / 失焦退階)。

### 2.3 顏色帶專職化 + override

見 §B 表。override 色(Peach/Red)不參與任何 layer scale —— 它們是「跳出階
層之外的警示」,不是階層上的一階(通用 §2.4)。**status bar 的 eza perm 逐字
上色**(type 藍、`r` 黃、`w` 紅、`x` 綠、`-` dim)沿用檔案清單同一套 eza 語
彙,讓「權限」在狀態列與清單裡讀起來是同一種顏色語言(`view.go colorPerm`)。

---

## §3. 符號語彙 in filu

### 3.1 Nerd Font 是設計、必裝

filu 用 Nerd Font glyph 當視覺語彙(型別 icon、powerline chip cap、pick 勾),
是設計的一部分、不做降級(通用 §3.1)。source 裡不放 PUA glyph 字面,一律用
`string(rune(0x...))` 建構。

### 3.2 CJK icon 寬度(CPR 偵測)

CJK Nerd Font(如 Maple Mono NF CN)可能把 file-type icon 畫成 2 格。filu 啟動
時用 **CPR**(cursor position report `\x1b[6n`,`iconwidth_unix.go
DetectIconWidth`、在 `tea.NewProgram` 之前)實測 icon 實際格寬,存入
`iconCells`;`width.go` 的 display-width 層據此保留版面空間。`isWideIcon` 涵蓋
PUA(型別 icon)+ 羅馬數字 `U+2160–2162`;powerline caps `U+E0A0–E0D7`(三角
/圓頭)刻意排除(它們單寬)。多數終端 `iconCells=1`、此層 no-op。

### 3.3 控制字元清洗(`safeName`)

檔名可能含控制字元(macOS 自訂圖示檔字面叫 `Icon\r`,含 CR)。畫出原始 CR 會
把游標打回行首、打碎版面。`safeName`(`list.go`)在**顯示時、套 lipgloss 前**
剝掉控制字元(`< 0x20` 或 `0x7f`;剝 ESC 也擋 ANSI injection),檔案操作仍用真
實名。套在所有 name-render 點(list / tree / archive / places / carry / meta /
header 麵包屑 / rename input)。

### 3.4 Surface 標籤(類型訊號 + 內容訊號)

- **panel chrome**:`[N] label`(型別訊號 `[N]` + 內容訊號 label)。
- **tab bar**:羅馬數字 = 位置標記(§1.2),路徑內容外包給 header。
- **pick 勾**:兩個語意兩個 glyph(§B)。
- **行號 gutter**:「號碼 + 一空格」(無 `│` 分隔符,對齊 kbu yaml popup);
  preview text/binary 與 [3] yank viewport 共用同一 gutter 形式。
- **popup content 列刻意不放 glyph**(CJK-safe):popup 用 `lipgloss.Width`
  量、對 ambiguous/PUA glyph 會低估,glyph 只擺 title/hint(邊框列)。例:
  breadcrumb popup 的「你在此」層改用 lavender 文字標記、不用 `●`。

---

## §4. 互動 in filu

### 4.1 Core 5 鍵 + 導覽

見 §A.0.Y。五鍵語意在任何 panel / popup 都不變。

- **`h`/`l`** = 切當前 focused panel 的 tab([2] 切目錄分頁、[4] 切
  Carries/Tasks;[3]/[5] 無 tab、h/l no-op)。
- **vim 導覽**(`j/k/u/d`)跨 surface 同義;**跳頂 `gg`**(單 `g` 待命等第二
  鍵、對齊 kbu)、`G` 跳底,適用 [2]/[3]/[4]/[5]。
- **`go`** = goto finder chord(空出 `g` 前綴後接 `o`;`gt` 因 vim 是 go-to-tab
  而放棄)。

### 4.2 Letter hotkey ⊆ Space menu

filu 每個 contextual letter hotkey 都在對應 focus 的 Space menu 現身(§A.1
表)。動作鍵盡量小寫(`p`/`y`/`e`/`c`/`m`/`r`/`a`/`b`/`f`);`D` delete、`S`
sort、`P` pin、`R` redo 維持大寫(避開小寫 `d` = half-page-down 等衝突)。

### 4.3 取消鍵 = `Esc`(finder 清單態的 `q` 例外語意)

`Esc` 通殺:有浮層先關浮層,否則回上層目錄(`list.parent()`)。**兩段式**:
yank viewport 的 visual 模式下,第一次 `Esc` 先退 visual、第二次才關(§6.5)。

**finder(Search/Find/Goto)清單態的 `q`**:三個 finder 都是「輸入 → Enter 交
清單 → j/k 選 → Enter reveal」。清單態的 `Esc` = **離開 finder**(同其他
popup),`q` = **回到輸入列**改 query。這裡 `q` 不是「取消 role」的一部分(它
只在 finder 清單這個 sub-surface 有「回輸入」的意義),`Esc` 仍是唯一的取消
role,語意一致。

### 4.4 Hotkey discoverability — bracket `[X]label`

Space menu / help 用 `[X]label` 揭露 letter hotkey(`spacemenu.go bracketHotkey`);
單字 chord 也就地 bracket(`[go]to`);純 label 不加 bracket。

### 4.5 panel [3] Preview yank + visual selection

`[3]` Preview 聚焦時 `y` 開一個 **yank viewport**(`detailYank.go`)覆蓋 preview
內容,是「選文字複製」的專用視角:

- vim-style **cursor**(`j/k/g/G/u/d` 移動)+ **`v` 進 visual**、字元級選取。
- 有選取 → `y` 複製選取內容;沒選取 → `y` 複製全部。皆走 OSC 52(可跨 SSH /
  tmux)+ toast 回饋。
- **兩段式 Esc**(§6.5):visual 中先退 visual、再按才關 viewport。
- text/binary preview 帶「號碼 + 空格」的 display-only gutter(§3.4);複製時
  gutter 不進剪貼簿。
- `[5]` Meta 的 `y` 走同一套 viewport、yank 的是 meta 那幾行。

這是「同一份內容的另一個視角」(§6 viewport 類),不是新面板 —— 讀時覆蓋 [3]、
關掉回原內容。

---

## §5. Mouse in filu

`(planned)` —— IDEA 規劃沿用通用 §5 的 mouse mapping(與 kbu 同款:左鍵
focus+select、雙擊 Enter、右鍵 Space、滾輪 u/d),**目前尚未 wire**。原則:
mouse 必為 keyboard 的 mapping、不引入新語意(通用 §5)。

---

## §6. 浮層 in filu — Popup Convention

### §6.1 Popup 4 類 taxonomy(+ 兩個設計案例)

| 類型 | filu 實例 | 特徵 |
|---|---|---|
| **menu**(選單) | Space menu、Sort picker、**Quit(cd-on-quit)picker**、Breadcrumb popup、**Search / Find / Goto finder** | 分 region / 清單、cursor-first、選一個執行 |
| **message**(訊息) | Delete 確認、Toast(yank 回饋) | 短、確認 / auto-dismiss |
| **viewport**(捲動視窗) | `?` help、[3]/[5] Yank viewport(`detailYank`) | 可捲、可 cursor / visual selection |
| **pty**(內嵌終端) | Edit(`$EDITOR`) | 外部全螢幕程式在 filu 內 render |

全部走共用 `drawPopupBox`(kbu form:title 嵌上邊框、hint 嵌下邊框、內容行夾在
兩條 padding row 之間)。**例外**:yank viewport 與 finder 用
`drawPopupBoxPad(pad=false)` 讓內容貼齊邊框(無前導空白列)。

#### 案例:quit 用 picker、不是單純 confirm

離開 app 大可只跳一個 yes/no confirm。filu 刻意把它做成 **menu(picker)**
(`quit.go` `quitMenu`),因為「離開時 shell 要 `cd` 去哪」**是個選擇、不是一次
確認**:

- picker 列出 distinct 目標 —— **LaunchDir**(panel [1] 的啟動目錄,用它自己
  的 `<CWD glyph> LaunchDir` 樣式呈現)+ 各分頁的當前目錄,**去重**(同一目錄
  開在多個 tab 只列一次,`quitTargets`)。
- 選法照 menu 通則:數字直達 或 `j/k` + `Enter`;`Esc` 留下不退。
- **有任務在跑時**,picker 頂端插一條紅字 warning header(override 色,§2.3)
  —— 這時「離開會中斷 copy/move」需要 user 知情,但仍是「選去哪」+「知道有代
  價」,不是把整個動作降級成 confirm。
- cd-on-quit 本身是 **OS 限制**:子程序改不了父 shell 的 cwd,只有 shell
  builtin 能 —— 靠 `eval "$(filu shell)"` wrapper 讀 filu 寫的暫存檔再 `cd`。

分類判準(通用 §6.1):動作有**多個對象要選** → menu;只需**一次是/否** →
message。quit「選 cd 目標」屬前者,所以是 menu 不是 message。

#### 案例:原生 finder(Search `/` / Find `f` / Goto `go`)

三個 finder 共用同一個 filu **自畫的 picker**(`search.go`,snacks/Telescope
形式,**非 fzf binary**),都是 menu 類、都有右側 preview、都**串流載入**:

| 模式 | 鍵 | 比對什麼 | 範圍 | preview |
|---|---|---|---|---|
| **Search** | `/` | 檔名 fuzzy(記憶體、匹配品質排序) | 當前 tab 子樹 | 選中檔(從頭) |
| **Find** | `f` | 內容(`rg --files-with-matches`,去重) | 當前 tab 子樹 | scroll 到命中行 + lavender bar |
| **Goto** | `go` | 路徑 fuzzy | `$HOME`、**只列目錄** | 選中 dir 的 tree |

- **⚠️ 不接 fzf binary**:走過 fzf-in-PTY(彩色 rg + 每鍵 reload + preview 把
  vt10x 畫爆、root 改 Home 掃整棵卡死)—— native picker 解掉這兩類 bug。fzf 的
  **串流概念**有偷師、原生實作(見下、§7.2)。
- **串流載入**:`fd`(缺退 Go walk)一吐就顯示、**依 fd 走訪序、不排序**,首批
  近乎立即、載入中可濾;撞 `finder_cap`(預設 50000、`config.yaml` 可調)停。
- **modal + Esc/q**:輸入態 → Enter 交清單 → Enter reveal 到 panel [2](cd 到
  該檔/目錄);清單態 `Esc` 離開、`q` 回輸入(§4.3)。
- **mtime 是爛訊號的教訓**:Goto 一度用 mtime 排序,但 mtime 追的是「OS/工具碰
  過」不是「你想跳過去」(把某 dir 從 Finder sidebar 移除反而把它排到最前),
  全砍、改 fd 走訪序 + 靠打字找。想要真 recency 只能 zoxide 式磁碟快取(未做)。

### §6.2 開關動畫 — PopupAnimator

所有 popup 用 `popupAnimator`(line→expand 開、reverse 關),各自獨立 animator
name 避免 tick 互撞(通用 §6.2)。

### §6.3 Border 色 = layer 明度

popup border 用 `popupLayerColor(layer)`,不 hardcode(§2.2 / 通用 §6.3)。

### §6.4 Stack 預設保留 source

popup 疊 popup 預設保留底層 source。**Context-shift 例外**:PTY(editor)佔滿、
退出後才回 list;yank viewport / finder 覆蓋所屬面板內容。

### §6.5 取消鍵通殺 + auto-dismiss

`Esc` 關任何 popup;toast 有計時 auto-dismiss(id 世代守衛防舊 timer 誤關新
toast)。yank viewport 的 `Esc` 兩段式(先退 visual)、finder 清單態 `Esc` 離開
`q` 回輸入(§4.3)。

### §6.6 Menu region cursor-first

Space menu 分 item-region(對 cursor item)/ panel-region(對當前 panel),
cursor-first 排序(`groupedMenu`);單一類動作時不分 region、直接列(通用 §6.6)。

---

## §7. 時間軸 UX in filu

### 7.1 Carry-bucket 兩層 pick(panel [2] carry / panel [4] carries)

carry-bucket 是**延遲決定 cp/mv** 的模型(對齊 macOS Finder Cmd+V /
Cmd+Opt+V),分**兩層**,對應兩個面板:

- **panel [2] Pick(進 bucket)**:游標檔按 `p` → 丟進 carry bucket(常駐的
  reference list、不是 mode)。[2] 對「在 bucket 裡」的檔案在最前面固定一格畫
  `pickGlyph`(f0bf3);沒在 bucket 的該格留白 —— **固定 `mark + space + icon`
  版位**,pick/unpick 只換那一格 blank↔glyph、**icon 不左右位移、也不貼到
  glyph**(`list.go`)。等同一個 in-place multi-select。
- **panel [4] Carries Pick(進 land 子集)**:focus 進 [4] Carries、對某項按
  `p` → 標它進「下一次 land 的子集」,用 `carryPickGlyph`(f05d)。
- **Land**(cp/mv 落地才決定):有 land-subset → 只對 picked;沒 pick → 對整個
  bucket。cp 不改 bucket(可連續複製到多目錄)、mv 更新路徑保持有效
  (`carry.go`)。
- **兩層 pick 兩個 glyph**:「在 bucket」(成員)vs「在 land 子集」(子集)是兩
  個狀態,分別用 f0bf3 / f05d,不共用(§B)。

### 7.2 Streaming 不退階(Task 進度 + finder 載入)

- **Task 進度**:落地(cp/mv)跑 goroutine,進度經 channel → `tea.Msg` 餵回
  [4] Tasks tab 即時更新(running/done/pending/error),即使 [4] 失焦也不退階
  (通用 §7.2)。中斷任務存 `state.yaml`、下次可從 Tasks `R` Redo。
- **finder 載入**:三個 finder 邊掃邊串流出清單(`fd` 一吐就顯示、不等整趟走
  完、不排序),首批近乎立即、載入中可濾 —— 這是 fzf 串流概念的原生版(§6.1)。
  「資訊正在到達」屬 streaming、不 dim。

---

## §8. Panel chrome in filu

### §8.0 三件套 + 兩條頂部 content row

| 件 | filu 實作 |
|---|---|
| **border title** | powerline chip(`singleChip`,`[N] label`);有 tab 的面板用 tab bar(`tabBar`,溢位退 `carouselChip`) |
| **tab bar** | [2] 目錄分頁(羅馬數字)、[4] Carries/Tasks;§1.2 |
| **border hint** | popup 下邊框的操作提示(如 yank viewport 的 ` v:visual  y:copy  Esc:close `) |
| **header 麵包屑** | active tab 的完整路徑(獨立 content row,§8.2) |
| **top status bar** | 當前 tab 目錄的狀態(獨立 content row,§8.3) |

### §8.1 Focus 訊號

box 用 border 色(structural blue 亮/暗)+ 字重表示 focus / unfocus。`[3]`/`[5]`
是**參考視角**(邊看別的面板邊讀),失焦**不 dim**、保色;list/pin 的 cursor
bar 失焦退成 lavender 足跡色(§2.2)。

### §8.2 header = powerline 麵包屑(bg/fg 靠明度差、不用分隔符)

header 從扁平路徑列升級成 **powerline 麵包屑**(`header.go`)。結構:
`capLeft(圓左) + [Ⅱ 󰝰] chip + ▶seg ▶seg … + capRight(圓右尾)` ——
第一個 chip 帶 active-tab 羅馬數字 + folder glyph(順序:**數字在前、glyph 在
後**),之後每層目錄一個 chip。**不隨 focus dim**(header 永遠是「你在此」)。

#### 用「from→to 明度差」造階層,而不是用 `` 分隔符

這是本案例的核心決定。段與段之間的分隔有兩種 powerline 手法,filu 選了前者:

- **`capHard`(▶,``)實心填滿三角**:它的斜邊「**就是**」兩段 bg 的顏色
  交界 —— 三角形狀靠 `fg=前段色、bg=本段色` 的色差切出來。
- **`capThin`(❯,``)細線 chevron**:雖有斜線,但**背景仍是 rect**(色
  差邊界是垂直的),斜線和色階不重合 → 破壞三角的無縫感。

所以 filu **不用 `` 當分隔**,改用 **每層不同背景色 + capHard 三角**,靠
**相鄰段的明度差**把階層切出來。但這帶出一個幾何硬限制:

> **「看得見的三角分隔」⟺「相鄰兩段要有夠大色差」** —— 三角就是色差本身,兩
> 段近同色就切不出三角。

因此 header 漸層**不能**沿用 popup 那條低對比的 lavender→sapphire(相鄰段太
近、三角全隱形、中段只剩一坨背景)。改成**跨度大的 blue `#89b4fa` → crust
`#11111b`**(root→current,`crumbColorAt` 依 depth 連續內插):明度落差大、每段
之間都切得出三角,又保留連續漸層。這就是 §B 說的「header 深度漸層是**另一套**
色帶、不與 popup layer scale 混用」。

#### 亮→暗漸層 → 文字色隨背景明度翻(WCAG)

跨度變成「亮 blue → 暗 crust」後,固定深字在 crust 端會變黑字壓黑底。
`crumbTextAt(t)` 用 **WCAG 對比度**擇優:深字 `baseHex` / 亮字 `#cdd6f4`,誰跟
該段 bg 對比高就用誰(翻白點約 t=0.40)。**不用** Rec.601 明度中點門檻 —— 它翻
得太早(blue 本身亮、一下破門檻,約 t=0.21 就翻白)。

#### 過長縮排、收尾

過長時走 §1.2 的漸進三階(前段縮首字元 → 中間 `…`);首個 folder chip 也染 blue
(整條從第一格就 blue、協調),尾端 `capRight` 圓角收(bookend 開頭的圓左)。
width 不變式:powerline caps 在 `isWideIcon` 判單寬、`padDisp` 對齊,header 列
永遠等於終端寬。

### §8.3 top status bar(目錄狀態,eza 配色,每 frame 零 I/O)

header 下一列是 status bar(`view.go statusBar`),顯示 **active tab 目錄本身**
的狀態(不是游標檔 —— 游標檔在 [5] Meta):

- **欄位**:`perm · owner:group · item count · hidden count`(左)+
  `free / total` 磁碟(右)。
- **eza 配色**(§2.3):perm 逐字(type 藍、`r` 黃、`w` 紅、`x` 綠、`-` dim),
  owner 黃、size 綠、單位字 dim。
- **效能(status bar 必須快)**:perm/owner/disk 在 `listModel.reload()` 算一次
  快取(1 個 `stat` + 1 個 `unix.Statfs`,**非遞迴**、微秒級);item/hidden count
  從已載入清單直接數(`readEntries` 順手數 dotfile)。**render 只讀快取字串、每
  frame 零 I/O**。**不放**遞迴 content size(會 walk 整棵子樹、卡);disk free 用
  statfs(讀 superblock,跟遞迴 size 完全兩回事)。這一列同時**兼做 header 與面
  板之間的視覺分隔**(試過 dim 實線太重、空白 spacer,最後定案用 status bar)。

---

## §9. 實作狀態

### 已落地

- **5-panel layout**([1]pin·[2]list·[3]preview·[4]carry·[5]meta,grid
  `[1][2][3]/[1][2][3]/[4][4][5]`)+ header 麵包屑 + top status bar + footer;
  `z` zoom(依動態 tab 數分欄);窄寬 fallback。
- [1] Places(**LaunchDir**/Home/Root)+ Pinned;`P` pin/unpin;pinned path 與
  header 共用漸進縮字。
- [2] 動態 1–3 目錄分頁(`t`/`w`、羅馬數字標籤)、`h/l` 切;vim 導覽 + `gg`/`go`
  chord;`Enter` 進目錄 / OS 開檔;`Esc` back。
- [2] 動作:Pick `p`(f0bf3 勾、固定版位)、Yank `y`、Edit `e`(內嵌 PTY)、
  Copy `c` / Move `m`(async land)、Rename `r`、Add `a`、Delete `D`(→系統垃圾
  桶,確認)、Sort `S`(多層 chain)、Hidden `.`、Pin `P`;Search `/`、Find `f`、
  Goto `go`、Breadcrumb `b`。
- [3] Preview 五分類(dir/archive/image·base64/text·chroma/binary·hex)+ PDF;
  yank viewport(cursor + `v` visual)。
- [4] Carries(f05d land-subset pick)/ Tasks;carry-bucket 兩層 pick、延遲 cp/mv;
  async 進度;Tasks `R` redo。
- [5] Meta(`stat` 等級 metadata,`osstat_{darwin,linux}` build-tag)+ yank。
- **finder** 三模式(`/`·`f`·`go`)原生 picker、全 preview、串流載入、`config.yaml`
  調 cap/ignore。
- Popup 全套(menu/message/viewport/pty)+ animator + layer 色;quit picker;
  breadcrumb popup。
- fsnotify 即時刷新;`state.yaml` session 持久化;cd-on-quit(`filu shell`
  wrapper)。
- CJK icon 寬度 CPR 偵測;`safeName` 控制字元清洗。
- 平台:unix-first(build tag),`GOOS=windows` 刻意不編;`CGO_ENABLED=0`。

### 規劃(planned)

- **Mouse**:沿用通用 §5 mapping(與 kbu 同款)。
- **每列錯誤 `!` 前綴**:broken symlink / 無權限 / 上次操作失敗(目前僅面板層
  error note)。
- **zoxide 式磁碟快取索引**(給 goto 真 recency,只在串流不夠時做)。
- chmod / compress / extract、真圖(kitty/sixel)、sort filter、續傳:見 IDEA。

---

## 附錄 — filu hotkey 全表

### Core key(跨 surface 不變、5 個)

| 鍵 | 語意 |
|---|---|
| `Tab` / `1`-`5` | 切面板 |
| `Enter` | 進目錄 / 開檔(OS) |
| `Esc` | 關浮層 / 回上層 |
| `Space` | contextual 入口(Space menu) |
| `?` | non-contextual 入口(help popup) |

### Contextual letter hotkey(在 Space menu 現身)

| Focus | 鍵 | 動作 |
|---|---|---|
| [1] | `Enter` `P` | Jump / UnPin |
| [2] | `p y e c m r a D S . P z` `/` `f` `go` `b` `t` `w` | Pick / Yank / Edit / Copy / Move / Rename / Add / Delete / Sort / Hidden / Pin / Zoom / Search / Find / Goto / Breadcrumb / Tab / Close tab |
| [3] | `y` `z` | Yank viewport / Zoom |
| [4] Carries | `p y D l z` | Pick / Yank / Delete / Tab / Zoom |
| [4] Tasks | `R D l z` | Redo / Delete / Tab / Zoom |
| [5] | `y` `z` | Yank / Zoom |

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
| `go` | goto finder(g-prefix chord) |

---

## 結語

filu 是通用設計原則在 filesystem domain 的一個實現(kbu 是 K8s domain 的另一
個)—— 兩者平行、共用同一套通用原則,不是誰派生自誰。filu 落地的原則骨架:
core-key 5 個不變、
Space + `?` 雙軌揭露、popup 四類 taxonomy、明度 z-axis、carry-bucket 延遲決策。
filu 自己長出來的幾個實作案例 —— 隨寬逐階縮字(header/pin 共用)、動態 tab 與
header 分工、quit 用 picker、原生串流 finder、兩層 pick 兩個 glyph、依動態 tab
分欄的 zoom、preview yank visual、header 靠明度差造深度階層(而非 `` 分
隔)—— 都收在對應章節。凡設計決定以 `.forge/meta/IDEA.md` 為準;凡通用原則以
**VTP**([Vulcan's TUI Design Principle](https://github.com/vulcanshen/thoughts/blob/main/vtp.md))為準。本文件隨
實作演進更新,不宣稱未落地的行為。

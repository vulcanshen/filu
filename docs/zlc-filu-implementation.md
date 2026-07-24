# filu — ZLC Implementation

filu 是 kbu `u`-family 的成員,把 kbu 已驗證的 ZLC(Zero Learning Curve)設計
系統從 K8s domain 換到 filesystem domain。本文件是 filu 對通用設計原則
(kbu repo `docs/zlc-tui-design-principle.md`)的**具體落地紀錄**,結構鏡射
`docs/kbu-zlc-implementation.md`,逐節對照 filu 的實作。

> **設計權威順序**:`.forge/meta/IDEA.md`(filu 專屬決定)> kbu
> `zlc-tui-design-principle.md`(通用原則)> kbu `kbu-zlc-implementation.md`
> (reference 實作)。三者衝突時以 IDEA.md 為準。
>
> **狀態標記**:本文件描述**當前已落地**的實作。尚未完成者標
> `(planned)` / `(實作中)`,不宣稱未完成的行為。實作狀態總表見文末
> 「§9 實作狀態」。

---

## §A. ZLC implementation in filu

### §A.0 filu ZLC score 對照

| 軸 / 結果 | filu 值 | 計算 |
|---|---|---|
| **X. 揭露程度** | ~1.0 | Space menu 列出當前 focus 的 contextual 動作 100%、`?` help popup 列出全域動作 100%。以 user 學習單位計:`e` edit 對所有文字檔通用、算 1 個 action;`Enter` open 對所有型別通用、算 1 個 |
| **Y. core-key role 數量** | 5 | `Tab`(focus 切換)/ `Enter`(確認·進入·開檔)/ `Esc`(取消·回上層)/ `Space`(contextual 入口)/ `?`(non-contextual 入口)。`Ctrl+C`(硬退)與 `q`(cd-on-quit picker)不另計 role — 見 §A.0.Y |
| `min(1, 5/Y)` 係數 | 1.0 | Y = 5、無 penalty |
| **ZLC score** | `~1.0 × 1.0` = **~100%** | 不靠事先學就能用 |

filu 的定位不是「再做一個 yazi」,而是「**第一次開就能不看文件開到底**」的
檔案管理器 —— letter hotkey 是加速捷徑、不是必經之路,光靠 `Space` + `?`
就能走完該 focus 的所有動作。

### §A.0.Y filu core-key 集合(5 個)

| Core-key | filu 語意 | 對應通用條款 |
|---|---|---|
| `Tab` | focus 切到下個 panel(`1`–`4` 為直達 alias) | §4.1 |
| `Enter` | 進入目錄 / 開檔(交 OS) | §4.1 |
| `Esc` | 關閉最上層浮層 / 回上層目錄(LIFO back) | §4.3 |
| `Space` | §A.1 contextual 入口(Space menu) | §A.1 |
| `?` | §A.2 non-contextual 入口(help popup) | §A.2 |

5 個,剛好通用 §A.0.Y 上限。letter hotkey(`p`/`y`/`e`/`c`/`m`/`r`/`a`/
`D`/`S`/`P`/`.`/`z`/`/`)**不算 core-key**,是兩個入口內動作的加速捷徑。

**`Ctrl+C` 與 `q` 為何不各記一個 role**:`q` 不是「取消」——它開 cd-on-quit
picker(選離開時要 `cd` 去哪),語意屬「離開 app 並帶目錄回 shell」,是一個
全域動作(列在 footer + help)。`Ctrl+C` 是逃生硬退,與 `Esc`(退浮層 / 退
目錄)語意不同、不重疊,屬 emergency exit,不佔 core-key role 上限的計算。
取消 role 由 `Esc` 單獨承載(§4.3)。

### §A.1 Contextual track in filu — Space menu

每個 panel focus 都接 `Space`,列出「這個 focus 此刻能做什麼」。入口自身在
**footer** 揭露(`view.go footerBar`):

```
 space menu   ? help   tab/1-4 panels   q quit
```

user 第一次開、沒看 README,從 footer 就知道按 `Space` 會跳選單。

filu 各 focus 的 Space menu(`app.go` `menuItems()`,`groupedMenu` 分
item-region / panel-region,cursor-first 見 §6.6):

| Focus | item-region 動作 | panel-region 動作 |
|---|---|---|
| **[1] Places** | Jump(`Enter`)、UnPin(`P`,僅 Pinned 項) | — |
| **[2] List** | Pick `p`、Yank `y`、Edit `e`、Rename `r`、Delete `D`、Search `/`(實作中) | Copy `c`、Move `m`、Add `a`、Sort `S`、Hidden `.`、Zoom `z`、Pin `P` |
| **[3] Preview/Meta** | Yank `y`(開 yank viewport) | Tab `l`、Zoom `z` |
| **[4] Carries** | Pick `p`、Yank `y`、Delete `D` | Tab `l`、Zoom `z` |
| **[4] Tasks** | Redo `R`、Delete `D` | Tab `l`、Zoom `z` |

**完整性 audit**:新增一條 contextual 動作,必須同步在對應 focus 的 Space
menu 加 entry,不能只綁 letter hotkey。否則就是 ZLC 破洞。

### §A.2 Non-contextual track in filu — `?` help popup

`?` 在任何 surface 跳出全域動作清單(`helppopup.go` `helpRows`)。入口自身
同樣在 footer 揭露(`? help`)。

filu 是檔案管理器,domain 動作幾乎都是 contextual(對 cursor item / 對當前
tab),所以 **§A.2 全域軌很薄**:

| 全域動作 | letter / key | help popup 內列 |
|---|---|---|
| 說明 | `?` | ✓ |
| 離開(cd-on-quit picker) | `q` | ✓ |
| 硬退 | `Ctrl+C` | ✓ |
| 切面板 / 切 tab / 導覽 | `Tab`·`1-4`·`h/l`·`Enter`·`Esc` | ✓(core-key 揭露) |

沒有 kbu 那種 namespace / context 全域 toggle —— filesystem 的「當前位置」
是每個 tab 各自的 CWD(contextual),不是一個全域狀態。

### filu contextual / non-contextual 動作邊界(audit 用)

| 動作 | contextual? | track |
|---|---|---|
| 對 cursor item 做 Pick/Yank/Edit/Rename/Delete/Search | ✓ 對 cursor row | §A.1 |
| 對當前 tab 做 Add/Sort/Hidden/Copy-land/Move-land | ✓ 對當前 tab | §A.1 |
| Pin/UnPin | ✓ 對 cursor dir / Places 項 | §A.1 |
| Help / Quit | ✗ 全域 | §A.2 |

---

## §B. 元素專職化 in filu

| 元素 | 專職語意 | 不准兼職 |
|---|---|---|
| **Lavender**(`userColor`) | 使用者足跡:Pinned 項、unfocused panel 的 cursor bar | panel / popup border、hotkey signal |
| **Subtext1**(`handColor`) | focused panel 的 cursor bar(「當前的手」) | 其他狀態 |
| **綠 `#a6e3a1`** | 「已選入下一個操作」的勾:[2] carried 項、[4] Carries land 子集 | 純裝飾 |
| **Peach / Red** | warning / error override(quit 有任務跑的紅字警告、Task error 狀態) | 不參與 popup layer scale |
| **Panel border 色** | structural(blue 亮=focus、blue 暗=unfocus) | 不拿去做 user state |
| **Popup border 色** | layer 明度(`popupLayerColor`) | 不 hardcode |
| **eza 型別色**(`fileColor`) | 檔案型別 / executable-bit(綠) | — |
| `[X]label` bracket | letter hotkey discoverability | 純 label 不加 bracket |
| `Esc` | 關閉 / 退出 | 永遠不當「確認」 |
| `Space` / `?` | contextual / non-contextual 入口 | 不當別的 letter hotkey |
| Nerd Font glyph | 型別訊號(這列是什麼型別) | 不當熱鍵 signal、不當純裝飾 |

**一元素一語意的實例**:carried 檔案在 [2] 用**綠勾**、不用改字色 —— 綠勾
專職「已選」,檔名字色仍歸 eza 型別色管,兩者不搶語意。這也讓 [2] 的 Pick
`p` 與 [4] 的 Pick `p` 視覺一致(同一個勾),等同 multi-select。

---

## §1. 空間結構 in filu

### 1.1 版面 grid

```
 [] ~/Documents/sideproj/filu                    ← header 路徑 bar(content row)
┌──────────┬──────────────────┬──────────────────┐
│ [1] pin  │ [2] list         │ [3] Preview│Meta │
│  Places  │  (3 目錄分頁)    │                  │
│  Pinned  │                  │  (active tab 全高)│
├──────────┴──────────────────┤                  │
│ [4] Carries │ Tasks         │                  │
└─────────────────────────────┴──────────────────┘
 space menu   ? help   tab/1-4 panels   q quit    ← footer(content row)
```

grid `[1][2][3] / [1][2][3] / [4][4][3]`:`[1]` pin + `[2]` list 並排上 2/3、
`[4]` carry 橫跨 `[1]+[2]` 兩欄的下 1/3、`[3]` 右側全高。header / footer 各一
條**獨立 content row**(非 border title)。`view.go` `renderNormal` 用
`joinH`/`joinV`(display-width aware)組版。

### 1.2 窄寬可用 + Width stability

- **窄寬**:`w < 72` 時放棄 grid,只顯示 list 單欄(`view.go:117`);Zoom
  (`z`)是任何 panel 的逃生艙,把該 panel 展開佔滿、tab 攤成等寬並排欄。
- **Width stability**:[2] 固定 **3 個目錄分頁**,不可增減(IDEA §1.2)——
  tab bar 寬度不隨數量變。tab 標籤 = last dir basename(`pathBase`)。溢位時
  panel chrome 從完整 tab bar 退成 carousel chip(`carouselChip`,§8)。

### 1.3 Footer 行數

filu 選 **N=1**(單行 footer),一旦選定鎖死。無多行 statusbar。

---

## §2. 色彩 in filu

### 2.1 配色錨點(catppuccin-mocha)

| 錨點 | 用途 |
|---|---|
| `baseHex` `#1e1e2e`(base) | cursor bar 上的前景字色 |
| structural blue | panel border(亮=focus / 暗=unfocus) |
| `userColor`(lavender) | 使用者足跡(§B) |
| `handColor`(subtext1) | focused cursor bar |
| `dimColor` | 退階 / 次要文字 / 行號 gutter |
| 綠 `#a6e3a1` | 「已選」勾 |
| Peach / Red | warning / error override |

### 2.2 明度作 z-axis

popup border 用 `popupLayerColor(layer)` 做 layer 明度插值(越上層越亮),
不 hardcode 每個 popup 的顏色。cursor bar 的明暗(focused subtext1 vs
unfocused lavender)也是 z-axis 的一種:focus = 亮、失焦 = 退成足跡色。

### 2.3 顏色帶專職化

見 §B 表。override 色(Peach/Red)不參與 popup layer scale —— 它們是「跳出
階層之外的警示」,不是階層上的一階。

---

## §3. 符號語彙 in filu

### 3.1 Nerd Font 是設計、必裝

filu 用 Nerd Font glyph 當視覺語彙(型別 icon、powerline chip cap),是設計
的一部分、不做降級。source 裡不放 PUA glyph 字面,一律用 `string(rune(0x...))`
建構(`list.go` `iconDir`/`iconFile`、`preview.go` tree 分支符)。

### 3.2 CJK icon 寬度(CPR 偵測)

CJK Nerd Font(如 Maple Mono NF CN)可能把 file-type icon 畫成 2 格。filu
啟動時用 **CPR**(cursor position report,`\x1b[6n`)實測 icon 實際格寬
(`iconwidth_unix.go` `DetectIconWidth`,在 `tea.NewProgram` 之前),存入
`iconCells`;`width.go` 的 display-width 層據此保留版面空間。`FILU_ICON_WIDTH`
可覆寫、`filu iconwidth` 印偵測值。多數終端 `iconCells=1`,此層為 no-op。

### 3.3 控制字元清洗(`safeName`)

檔名可能含控制字元(macOS 自訂圖示檔字面叫 `Icon\r`,含 CR)。畫出原始 CR
會把游標打回行首、打碎版面。`safeName`(`list.go`)在**顯示時、套 lipgloss
前**剝掉控制字元(`< 0x20` 或 `0x7f`;剝 ESC 也擋 ANSI injection),檔案操作
仍用真實名。套在所有 name-render 點(list / tree / archive / places / carry /
meta / header path / tab label / rename input)。

### 3.4 Surface 標籤

panel chrome 用 `[N] label` 格式(型別訊號 `[N]` + 內容訊號 label);行號
gutter 為「號碼 + 一空格」(無 `│` 分隔符,對齊 kbu yaml popup)。

---

## §4. 互動 in filu

### 4.1 Core 4 鍵語意

見 §A.0.Y。五鍵語意在任何 panel / popup 都不變。`h`/`l` 為跨 panel 統一的
「切當前 focused panel 的 tab」([2] 切目錄分頁、[3] 切 Preview/Meta、[4] 切
Carries/Tasks)。vim 導覽(`j/k/g/G/u/d`)跨 surface 同義。

### 4.2 Letter hotkey ⊆ Space menu

filu 每個 contextual letter hotkey 都在對應 focus 的 Space menu 現身(§A.1
表)。「感覺這個 app 不太需要大寫熱鍵」→ 動作鍵盡量小寫(`p`/`y`/`e`/`c`/
`m`/`r`/`a`),唯 `D` delete 維持大寫(小寫 `d` 是 half-page-down)、`S` sort、
`P` pin、`R` redo 維持大寫。

### 4.3 取消鍵 = `Esc`

`Esc` 通殺:有浮層先關浮層,否則回上層目錄(`list.parent()`)。visual 模式
下(yank viewport)第一次 `Esc` 先退 visual、第二次才關(§6.5)。

### 4.4 Hotkey discoverability — bracket `[X]label`

Space menu / help 用 `[X]label` 揭露 letter hotkey;純 label 不加 bracket。

### 4.5 filu keymap 全表

見文末附錄。

---

## §5. Mouse in filu

`(planned)` —— IDEA 規劃繼承 kbu 的 mouse mapping(左鍵 focus+select、雙擊
Enter、右鍵 Space、滾輪 u/d),**目前尚未 wire**。原則:mouse 必為 keyboard
的 mapping、不引入新語意(通用 §5)。

---

## §6. 浮層 in filu — Popup Convention

### §6.1 Popup 4 類 taxonomy

| 類型 | filu 實例 | 特徵 |
|---|---|---|
| **menu**(選單) | Space menu、Sort picker、Quit(cd-on-quit)picker、**Search**(輸入過濾 + 清單,modal) | 分 region / 清單、cursor-first、選一個執行 |
| **message**(訊息) | Delete 確認、Toast(yank 回饋) | 短、確認 / auto-dismiss |
| **viewport**(捲動視窗) | `?` help、[3] Yank viewport(`detailYank`) | 可捲、可 cursor / visual selection |
| **pty**(內嵌終端) | Edit(`$EDITOR`) | 外部全螢幕程式在 filu 內 render |

全部走共用 `drawPopupBox`(kbu form:title 嵌上邊框、hint 嵌下邊框、內容行
夾在兩條 padding row 之間)。**例外**:yank viewport 用 `drawPopupBoxPad(pad=
false)` 讓內容貼齊上邊框(對齊 kbu yaml popup,無前導空白行)。

### §6.2 開關動畫 — PopupAnimator

所有 popup 用 `popupAnimator`(line→expand 開、reverse 關),各自獨立
animator name 避免 tick 互撞(`spacemenu.go` 註解)。

### §6.3 Border 色 = layer 明度

popup border 用 `popupLayerColor(layer)`,不 hardcode(§2.2)。

### §6.4 Stack 預設保留 source

popup 疊 popup 時預設保留底層 source。**Context-shift 例外**:PTY(editor /
search)佔滿、退出後才回 list;yank viewport 亦覆蓋 [3] 內容。

### §6.5 取消鍵通殺 + auto-dismiss

`Esc` 關任何 popup;toast 有計時 auto-dismiss(id 世代守衛防舊 timer 誤關新
toast)。yank viewport 的 `Esc` 兩段式(先退 visual)。

### §6.6 Menu region cursor-first

Space menu 分 item-region(對 cursor item)/ panel-region(對當前 panel),
cursor-first 排序(`groupedMenu`)。

---

## §7. 時間軸 UX in filu

### 7.1 Carry source-target 關係

carry-bucket 是**延遲決定 cp/mv** 的模型(對齊 macOS Finder Cmd+V /
Cmd+Opt+V):`p` Pick 把檔案丟進 bucket(reference list)→ 切到目標目錄 →
`c` Copy / `m` Move 才落地。bucket 可 `p` 選子集(land subset)。cp 不改
bucket、mv 更新路徑(`carry.go` `removeItem`)。

### 7.2 Task streaming 不退階

落地(cp/mv)跑 goroutine,進度經 channel → `tea.Msg` 餵回 [4] Tasks tab
即時更新(running/done/pending/error 狀態),即使 [4] 失焦也不退階。中斷
任務存 `state.yaml`、下次可從 Tasks 重跑(`R` Redo)。

---

## §8. Panel chrome in filu

### §8.0 三件套

| 件 | filu 實作 |
|---|---|
| **border title** | powerline chip(`singleChip`,`[N] label`);多 tab panel 用 tab bar |
| **tab bar** | [2]/[3]/[4] 的分頁列;溢位退成 carousel chip(`carouselChip`) |
| **border hint** | popup 下邊框的操作提示(如 yank viewport 的 ` v:visual  y:copy  Esc:close `) |
| **header 路徑 bar** | active tab 的完整路徑做 breadcrumb(獨立 content row;breadcrumb popup 為 `planned`) |

### §8.1 Focus 訊號

box 用 border 色(structural blue 亮/暗)+ 字重表示 focus / unfocus。失焦
panel 的 cursor bar 退成 lavender 足跡色(§2.2)。

---

## §9. 實作狀態

### 已落地

- 4-panel layout + header 路徑 bar + footer;`z` zoom;窄寬 fallback。
- [1] Places(CWD/Home/Root)+ Pinned;`P` pin/unpin。
- [2] 3 目錄分頁、`h/l` 切;vim 導覽;`Enter` 進目錄 / OS 開檔;`Esc` back。
- [2] 動作:Pick `p`(+綠勾 multi-select)、Yank `y`、Edit `e`(內嵌 PTY)、
  Copy `c` / Move `m`(async land)、Rename `r`、Add `a`、Delete `D`(→系統
  垃圾桶,確認)、Sort `S`(多層 chain)、Hidden `.`。
- [3] Preview 五分類(dir/archive/image·base64/text·chroma/binary·hex)+
  PDF;Meta;Yank viewport `y`(cursor + `v` visual selection)。
- [4] Carries / Tasks;carry-bucket 延遲 cp/mv;async 進度;Tasks `R` redo。
- Popup 全套(menu/message/viewport/pty)+ animator + layer 色。
- fsnotify 即時刷新;`state.yaml` session 持久化。
- cd-on-quit(`q` picker + `filu shell` wrapper,對標 superfile)。
- CJK icon 寬度 CPR 偵測;`safeName` 控制字元清洗。
- 平台:unix-first(build tag),`GOOS=windows` 刻意不編;`CGO_ENABLED=0`。

### 實作中 / 規劃

- **Search `/`**(已落地,待互動 smoke test):**filu 原生 file finder**
  (snacks/Telescope 形式,非 fzf binary),`internal/ui/search.go`。`/`(僅 [2]
  focus)開一個 filu 自畫的**分割 popup**:左=檔案清單、右=選中檔預覽(窄寬時
  改上下堆疊)。範圍是**當前 focus tab 的目錄子樹**。
  - **開啟即列全部**:`fd`(缺則 Go walk)列該目錄底下所有檔案+目錄,不用先打字。
  - **打字用「內容」過濾**:每次(120ms debounce + generation 守衛)跑
    `rg --files-with-matches <query>` → 只回「內容含關鍵字的**檔案**」(rg 天然
    去重,同檔多筆命中只出現一次);清空 query 立即還原全清單。
  - **選的單位是檔案/目錄**:預覽重用 `previewModel`(dir=tree、text=語法highlight
    +行號…);游標移動即換預覽。內容命中時,rg 帶回每檔**第一個命中行**,預覽
    **自動 scroll 到該行**並以 **lavender 背景**標記那行(上方留幾行 context)。
  - **modal**:輸入態 → **Enter 把 focus 交給清單**(`j/k/u/d`)→ **Enter 收起 +
    panel [2] reveal 該檔**(cd 到該檔目錄 + cursor 落該檔);清單態 **Esc 回輸入**、
    輸入態 Esc 關閉。UI 仿 snacks:輸入列在上(peach chevron prompt (U+F054) +
    blinking block cursor,無底色)、右上 count(灰)、固定 title `Search`。
  - **全程 filu 自畫、無 PTY** → 沒有「畫面亂跑 / 突破 popup」;範圍是當前子樹
    (不掃 Home),streaming + cap 10000 + 5s timeout 防爆;檔名經 `safeName` 才畫。
    跨目錄跳轉走 [1] Places / 導覽,不歸 Search。
- **Mouse**(planned):繼承 kbu mapping。
- **Breadcrumb popup**(planned):header 路徑各層一鍵跳。
- **每列錯誤 `!` 前綴**(planned):broken symlink / 無權限 / 上次操作失敗;
  目前僅 panel 層 error note(`(permission denied)`)。
- **chmod / compress / extract**、真圖(kitty/sixel)、sort filter、續傳:
  見 IDEA「後續可擴充」。

---

## 附錄 — filu hotkey 全表

### Core key(跨 surface 不變、5 個)

| 鍵 | 語意 |
|---|---|
| `Tab` / `1`-`4` | 切面板 |
| `Enter` | 進目錄 / 開檔(OS) |
| `Esc` | 關浮層 / 回上層 |
| `Space` | contextual 入口(Space menu) |
| `?` | non-contextual 入口(help popup) |

### Contextual letter hotkey(在 Space menu 現身)

| Focus | 鍵 | 動作 |
|---|---|---|
| [2] | `p` `y` `e` `c` `m` `r` `a` `D` `S` `.` `P` `z` `/`(實作中) | Pick / Yank / Edit / Copy / Move / Rename / Add / Delete / Sort / Hidden / Pin / Zoom / Search |
| [3] | `y` `l` `z` | Yank viewport / Tab / Zoom |
| [4] Carries | `p` `y` `D` `l` `z` | Pick / Yank / Delete / Tab / Zoom |
| [4] Tasks | `R` `D` `l` `z` | Redo / Delete / Tab / Zoom |
| [1] | `Enter` `P` | Jump / UnPin |

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
| `g` / `G` | 頂 / 底 |
| `u` / `d` | half-page 上 / 下 |
| `h` / `l` | 切當前 focused panel 的 tab |

---

## 結語

filu 用 kbu 的 ZLC 骨架,把 domain 從 K8s 換成 filesystem:core-key 5 個不變、
Space + `?` 雙軌揭露、popup 四類 taxonomy、明度 z-axis、carry-bucket 延遲決策。
凡設計決定以 `.forge/meta/IDEA.md` 為準;凡通用原則以 kbu
`zlc-tui-design-principle.md` 為準。本文件隨實作演進更新,不宣稱未落地的行為。

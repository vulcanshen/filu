# filu

<p align="center"><img src="docs/icon.svg" width="128" alt="filu icon" /></p>

[![GitHub Release](https://img.shields.io/github/v/release/vulcanshen/filu)](https://github.com/vulcanshen/filu/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/vulcanshen/filu)](https://go.dev/)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue)](LICENSE)

**語言**: [English](README.md) · 繁體中文

**終端機檔案管理器** — `Tab` / `Space` / `Enter` / `Esc` 驅動一切。不用背快捷鍵、不用設定、零學習成本。資訊豐富的檔案清單、marks 收集後複製/搬移、串流檔案 finder、即時預覽、cd-on-quit 全都內建。

> _遇事不決,就按_ **`Space`**。

filu 是 `u`-family 的成員,是 [Vulcan's TUI Design Principle](https://github.com/vulcanshen/thoughts/blob/main/vtp.md) 在 filesystem domain 的實現,跟 [kbu](https://github.com/vulcanshen/kbu) 共用同一套設計原則 — 見 [`docs/filu-implementation.md`](docs/filu-implementation.md)。

## Demo

### 上手操作
![basics](docs/demo-basics.gif)

### Marks — 跨 tab 收集後複製 / 搬移
![marks](docs/demo-marks.gif)

### 串流 finder — 模糊檔名 & ripgrep 內容
![finders](docs/demo-finders.gif)

### Favorites — 標記目錄,在 [3] Favorites 分頁管理
![favorites](docs/demo-favorites.gif)

### 預覽,再 yank 到剪貼簿
![preview](docs/demo-preview.gif)

### 當前 tab 目錄的 shell
![shell](docs/demo-shell.gif)

## 五個鍵驅動 filu

| 鍵 | 行為 |
|---|---|
| **`Tab`** | 切換面板 focus(或 `1`–`3` 直達) |
| **`Enter`** | 進入目錄 / 確認一個選擇 |
| **`Space`** | *這裡我能做什麼?* — 在任何面板跳出情境選單 |
| **`Esc`** | 退出 — 回上層目錄 / 關閉任何浮層 |
| **`?`** | 全域說明 — 所有 app 層級動作一次列出 |

遇事不決就按 `Space`。進階快捷鍵(`o` open / `O` open-with / `m` mark / `c` copy / `v` move / `f` favorite / `y` yank / `r` rename / `a` add / `s` shell / `D` delete / `S` sort / `/` search / `go` goto / `b` breadcrumb / `z` zoom / …)是為了加速而存在 — 每一個也都能從 `Space` 選單走到,所以除非你想背,否則什麼都不必記。

## 安裝

> filu **只支援 macOS / Linux**(不出原生 Windows build;請走 WSL)。

### 快速安裝(macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/install.sh | sh
```

下載 release binary,並把缺少的 `ripgrep` / `fd` 一併裝進同一個目錄(從各工具自己的 GitHub release 抓、免 sudo,binary 名就是 `rg`/`fd`,不是 Debian 的 `fdfind`)。該平台沒有預編 binary 時(如 Intel macOS 的 `fd`)才退回印安裝提示。

### Homebrew(macOS/Linux)

```bash
brew install vulcanshen/tap/filu
```

formula 宣告了 `ripgrep` + `fd` 依賴,所以內容搜尋與 finder 列檔開箱即用。

### 從 go install

```bash
go install github.com/vulcanshen/filu/cmd/filu@latest
```

### 從原始碼建置

```bash
git clone https://github.com/vulcanshen/filu.git
cd filu
CGO_ENABLED=0 go build -o filu ./cmd/filu   # 或:make build
./filu
```

`Makefile` 收攏了常用工作 —— `make build`(→ `./filu`)、`make install`
(→ `$GOPATH/bin`,把 `filu` 放上 `PATH`)/ `make uninstall`、`make package`
(打包當前平台的 `.tar.gz` 到 `dist/`)、`make check`(fmt + vet + test)。
跑 `make` 列出全部。

### 移除

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/uninstall.sh | sh
```

## Quick Start

```bash
filu
```

開在你當前的目錄、focus 在檔案清單。`Enter` 進目錄,`Space` 開情境選單,`Esc` 退回,`Tab` 切換面板。

想讓 filu 在離開時把你的 shell 切到某個目錄(`q` picker),在 `~/.zshrc` / `~/.bashrc` 加一行,並用 **`filu`**(不是 `./filu`)啟動:

```sh
eval "$(filu shell)"
```

filu 的 cd-on-quit 對標 [superfile](https://github.com/yorukot/superfile) 的 `cd_on_quit`,finder 則取法 [LazyVim](https://github.com/LazyVim/LazyVim) 的 search。以 Go 與 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 打造。

---

> 以下是操作手冊 — 想看完整功能面、每個快捷鍵、設定細節,繼續往下讀。

## 功能

- **零學習成本** — 所有動作都從 `Space` 選單浮現。進階快捷鍵是為了加速而存在,但你可以完全忽略整張 cheat sheet;`Space` 每次都在情境裡帶你走一樣的選單。上手一句話:*「遇事不決,就按 Space。」*
- **3 面板工作區** — `[1]` 檔案清單(主戰場)、`[2]` Preview、`[3]` 一個 tabbed 的 **Marks | Tasks | Favorites** 面板。上排 list 與 preview 以 2:1 共用(資訊豐富的 list 值得更寬);tabbed 面板在下方橫跨滿寬。`Tab` 輪切三個(或 `1`–`3` 直達);`h`/`l` 切某面板自己的分頁。
- **資訊豐富的檔案清單** — 每一列是一組欄位:狀態 glyph、`Modified`、`Owner`(user:group)、`Perms`(eza 配色 `r` 黃 / `w` 紅 / `x` 綠)、`Size`(eza color-scale,越大越暖;目錄顯示 `-`,絕不遞迴加總)、以及 icon + 檔名。欄位標題兼排序指示,面板變窄時欄位漸進收合(owner → size → modified → perms),檔名永遠最後才收。
- **每目錄各自排序** — `S` 挑欄位(Name / Modified / Owner / Perms / Size)與方向,可疊多層 chain。每個目錄記住自己的排序 —— 幫 `~/Downloads` 設一種它就固定在那,與其他目錄互不影響 —— 存進 `state.yaml`。
- **Powerline 麵包屑 header** — 當前分頁的完整路徑以 powerline 麵包屑橫在最上方,顏色依目錄深度沿 `crust → blue` 漸層(root 是最深的 chip、你所在的當前目錄是 blue)。過長時把前段縮成字首(`~/Documents/x` → `~/D/x`),再不夠中間縮 `…`。
- **Marks 複製與搬移** — 延遲的 cp/mv,像 Finder 的 Cmd+C / Cmd+V。`m` 把檔案 mark 進 marks bucket(清單上打 glyph 標記 — 一種 in-place 多選),接著切到目標目錄,`c` 複製 / `v` 搬移整個 bucket 過去。在 Marks 分頁,`p` 改成 pick 一個要落地的**子集**、`m` 則把某項 unmark。複製會保留 bucket(可連續落地到多個目錄);搬移會更新 bucket 內路徑讓它保持有效。
- **非同步落地 + 可讀 Tasks 分頁** — 複製/搬移在背景跑;進度串流到 `[3]` 的 Tasks 分頁,是帶時間戳的人話 log(`2026-07-28 14:32:07  Copied report.pdf → proj` / `… Move 5 items → proj (2/5 failed)`)。同磁碟搬移是瞬間 `rename`;跨磁碟 / 複製會顯示進度。`D` 從 log 移除一列。中斷的任務存進 `state.yaml`、還原成 pending。
- **原生串流 finder** — filu 自畫的分割 picker(清單 + 預覽),不是 fzf binary。每種模式都串流列檔(`fd` 走訪序、首批近乎立即、載入中可濾):
  - **`/` Search** — 開一個 chooser:**filename**(對當前分頁子樹的檔名做 fuzzy;以 `/` 開頭的 query 則改錨定到絕對路徑,磁碟上任何位置都到得了)或 **content**(用 `ripgrep` 對內容過濾;預覽自動 scroll 到命中行並反白)。
  - **`go` Goto** — 選一個 **Favorite** 目錄,或對 `$HOME` 底下的目錄路徑做 fuzzy **Search**;以 `/` 開頭的 query 會改錨定到該絕對路徑 —— 對整條路徑 fuzzy、深度限錨點下數層 —— 所以 home 以外的目錄也到得了;`Enter` 把當前分頁傳送過去。
  - **`b` Breadcrumb** — 當前分頁祖先目錄的 popup;`Enter` 把分頁跳上任一層。
  - 在結果清單裡,`Esc` 離開、`q` 回到輸入列。掃描上限(`finder_cap`)與要跳過的工具目錄(`ignore_dirs`)可在 `config.yaml` 調。
- **Favorites** — `f` 把游標目錄加最愛(清單上打星標)。`[3]` 的 **Favorites** 分頁以完整路徑列出所有最愛(正被某個 tab 開著的最愛會顯示該 tab 的羅馬數字而非星標) —— 直接在那按 `D` 移除;要跳到某個最愛,用 **Goto → Favorites**。瀏覽器書籤語意 —— 一個你會跳回去的存放點。
- **開啟 / 用 app 開啟** — `o` 先跳確認,再用 OS 預設 app 開當前檔案或目錄。`O`(shift-o)改跳 picker:**Default** 加上你在 `config.yaml` 的 `open_with` 列的 app(VSCode、IntelliJ IDEA…),各自跑 `<cmd> <path>` —— 拿來用 IDE 開當前資料夾很方便。在 filu,開檔靠 `o` / `O`(不是 `Enter`);`Enter` 只導覽。
- **開個 shell** — `s` 先確認目標目錄,再在當前 tab 的目錄開你的 `$SHELL`,跑在內嵌終端裡;下幾個指令、`exit` 就回來(目錄會 reload 以防檔案有變)。它是 **modal 子 shell** —— 靠 `exit` 離開、不能切走。
- **依型別預覽** — 讀 magic bytes 判定:目錄 → 內層 tree、壓縮包(zip / tar / tar.gz…)→ 內容清單、圖片 → base64 `data:` URI、SVG → 語法高亮的 XML、文字 → 語法高亮 + 行號(Chroma / catppuccin-mocha)、二進位 → hex + ASCII、PDF → 抽出的文字 + 頁數。
- **Yank 含 visual 選取** — 在 `[2]` Preview 按 `y` 開一個帶 vim 式游標的 viewport;`v` 進字元級 visual 選取,`y` 複製選取內容(沒選取則複製全部),走 OSC 52(可穿 tmux / SSH)。在檔案列或 Marks 項按 `y` 複製它的完整路徑。
- **刪到系統垃圾桶** — `D`(帶確認對話框)把檔案移到 OS 垃圾桶(macOS Trash / Linux XDG)。還原走你的檔案管理器垃圾桶介面。
- **動態目錄分頁** — `[1]` 預設開一個分頁;`t` 走 `{Same, Favorites, Search}` picker 開新分頁(合計最多五個)、`w` 關掉 active;到上限會 toast 提示、不再默默沒反應。分頁用羅馬數字(`Ⅰ` … `Ⅴ`)標 — 路徑在 header,分頁列只標位置與哪個 active。
- **啟動目錄狀態列** — header 下方一列靠右對齊,顯示 filu 啟動時所在的目錄(以薰衣草色 glyph 標記) —— cd-on-quit picker 會返回的那個固定參照。
- **eza icon + 配色** — 檔案型別 glyph 取自 eza 完整 icon 表(~760 個);顏色來自烘進 binary 的 `vivid generate catppuccin-mocha` `LS_COLORS` palette,依 eza 的優先序解析(目錄 → symlink → executable → 最長 suffix → 副檔名)。執行時不需要 `LS_COLORS` — 每個安裝都是同一套配色,與你終端的 `eza` / `ls` 一致。
- **即時刷新** — 清單分頁監看自己的目錄(fsnotify),外部增刪檔案時自動 reload、保留游標;連續事件會 debounce。
- **session 持久化** — 多開的分頁(dir + cursor)、marks bucket、favorites、tasks、每目錄排序都存進 `state.yaml`;第一個分頁永遠開在啟動目錄,而且每次啟動都 focus 在清單。
- **cd-on-quit** — `q` 開一個 picker,離開時把 shell 留在啟動目錄或任一分頁的目錄(需裝 shell wrapper;見 [cd-on-quit](#cd-on-quit))。
- **vim 式導覽** — `j`/`k`、`u`/`d` half-page、`gg`/`G`、`h`/`l` 切當前面板的分頁。
- **面板 zoom** — `z` 把 focus 的面板展開佔滿全螢幕;再按 `z` 還原 grid。
- **CJK Nerd Font 寬度** — 啟動時用 CPR 偵測 icon 實際格寬,讓面板框在把 file-type icon 畫成 2 格的 CJK Nerd Font(如 Maple Mono NF CN)下不會破。
- **unix-first、靜態 binary** — 只支援 macOS + Linux(不出原生 Windows build — `GOOS=windows` 刻意編譯失敗);`CGO_ENABLED=0` 做靜態 binary。

## 快捷鍵

### 主要互動:五個鍵

| 鍵 | 行為 |
|---|---|
| **`Tab`** | **面板** — focus 移到下個面板(或 `1`–`3` 直達) |
| **`Enter`** | **進入** — 進目錄 / 確認浮層選擇 |
| **`Space`** | **選單** — 在當前 focus 開情境選單。也關閉任何開著的浮層 |
| **`Esc`** | **返回** — 回上層目錄 / 關閉任何浮層 |
| **`?`** | **說明** — 全域(非情境)動作清單 |

只要有情境選單,`Space` 就夠了 — 不必背個別動作鍵。`h`/`l` 切當前面板的分頁(`[1]` 目錄分頁、`[3]` Marks / Tasks / Favorites)。

### 加速鍵 — 游標 + 觸發

```
 游標      j k        u d         gg G        h l(切本面板分頁)
 清單      o open     O open-with  m mark     c copy    v move    f favorite
           y yank     r rename     a add      s shell   D delete  S sort   . hidden   z zoom
 finder    / search   go goto      b breadcrumb
 分頁      t 開分頁   w 關分頁
```

`gg`(跳頂)是 vim g-prefix chord — 單 `g` 待命等第二鍵;`go`(開 Goto picker)用同一個前綴。`G` 跳底。

### 全域

| 鍵 | 動作 |
|---|---|
| `?` | help popup |
| `q` | 離開 — 開 cd-on-quit picker(把 shell 留在選定目錄) |
| `Ctrl+C` | 立即離開(會 kill 進行中的複製/搬移) |
| `y` | 複製 focus 元素的路徑 / 內容到剪貼簿(OSC 52) |

### 各面板 Space 選單

| Focus | 選單項目 |
|---|---|
| **`[1]` List** | Open `o`、Open with `O`、Mark `m`、Yank `y`、Rename `r`、Delete `D`、Favorite `f` · Copy `c`、Move `v`、Search `/`、Goto `go`、Breadcrumb `b`、Tab `t`、Close tab `w`、Add `a`、Sort `S`、Shell `s`、Hidden `.`、Zoom `z` |
| **`[2]` Preview** | Yank `y`、Zoom `z` |
| **`[3]` Marks** | Pick `p`、Yank `y`、Unmark `m` · Switch tab `l`、Zoom `z` |
| **`[3]` Tasks** | Delete `D` · Switch tab `l`、Zoom `z` |
| **`[3]` Favorites** | Delete `D`(取消最愛) · Switch tab `l`、Zoom `z` |

## cd-on-quit

按 `q` 會開一個 picker,列出 distinct 目錄 — 啟動目錄加上各分頁的當前目錄 — 離開時把你的 shell 工作目錄切到你選的那個(對標 superfile 的 `cd_on_quit`)。

**為什麼需要一行 shell 設定(這是 OS 限制、不是偷懶):** 一個 process 只能改**自己**的工作目錄 — 沒有任何 syscall 能改**父** process(你的 shell)的 cwd。filu 是 shell 的子程序,自己 `cd` 影響不到 shell。唯一能原生改 shell cwd 的是 shell **內建指令**,而 filu 是外部 binary。所以標準解法是個兩段式 handshake:程式把選定目錄寫進檔案、shell wrapper 讀檔再 `cd`。在 `~/.zshrc` / `~/.bashrc` 加上 wrapper:

```sh
eval "$(filu shell)"
```

之後用 **`filu`** 啟動(不是 `./filu` — wrapper 是攔截指令名 `filu` 的 shell function,帶路徑的呼叫會繞過它)。沒裝 wrapper filu 也能正常用,只是離開時不會切 shell 目錄。

## 設定

filu 從 OS config 目錄的 `config.yaml` 讀使用者設定。`state.yaml`(自動管理的 session 狀態)放在旁邊;兩者刻意分開 — `config.yaml` 是你手改的檔、`state.yaml` 每次離開自動重寫。

| OS | 路徑 |
|---|---|
| Linux | `$XDG_CONFIG_HOME/filu/` 或 `~/.config/filu/` |
| macOS | `~/Library/Application Support/filu/` |

首次啟動會寫一份帶註解的模板(已存在的檔永不覆蓋)。finder 旋鈕(`finder_cap`、`ignore_dirs`)加上 `[o]pen` 的 app 清單:

```yaml
# finder 最多掃幾筆才停。Goto 會走整個 $HOME,所以這個值決定它的上限 ——
# 調大 = 更多目錄跳得到,調小 = 在大 home 上 fuzzy 過濾比較不卡。
finder_cap: 50000

# finder 直接跳過的目錄 —— 快取、build 產物、IDE metadata、你不會 cd 進去的
# 容器資料。一般名字比對任意層級;含斜線(如 go/pkg)比對路徑。設成 [] 代表
# 不排除任何東西。
ignore_dirs:
  - node_modules
  - .git
  - Library
  - OrbStack
  - go/pkg
  - vendor
  - target
  - __pycache__
  - .venv
  - .idea
  - .vscode
  - .cache
  - .Trash

# [O]pen-with picker 的 app(對檔案或目錄按 O;單按 o 就用 OS 預設開)。每個 entry
# 是 name + 一個指令,filu 會跑 `<cmd> <path>`。「Default」(OS 預設 app)永遠排第一。
open_with:
  - name: VSCode
    cmd: code
  - name: IntelliJ IDEA
    cmd: idea
```

## 系統需求

- **Nerd Font** — filu 用 Nerd Font glyph(型別 icon、powerline chip)當視覺語彙;它是設計的一部分、不是 optional。在把 icon 畫成 2 格的 CJK Nerd Font(如 Maple Mono NF CN)上,filu 啟動時用 CPR 偵測實際格寬並據以排版,框線不會歪。
- **ripgrep** — `/` Search 的 content 模式必需。
- **fd** — 給 finder 列檔用;缺了退回純 Go walk。
- **macOS 或 Linux** — 不出原生 Windows build(`GOOS=windows` 刻意編譯失敗);請走 WSL。
- **Go 1.26+** — 從原始碼建置需要(`CGO_ENABLED=0` 做靜態 binary)。

## License

[GPL-3.0](LICENSE)

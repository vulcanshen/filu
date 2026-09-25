# filu — 開發備忘

README 只介紹「filu 能做什麼」;這份記的是「它為什麼長這樣、內部怎麼做」——
建置方式、平台決策、各功能背後的實作細節。想看完整的 UI 設計落地紀錄,見
[`filu-implementation.md`](filu-implementation.md)。

## 設計脈絡

filu 是 `u`-family 的成員,是 [this TUI Design Principle](https://github.com/vulcanshen/thoughts/blob/main/tui-design/README.md)
在 filesystem domain 的實現,與 [kbu](https://github.com/vulcanshen/kbu)(K8s domain)是
平行的 sibling、共用同一套設計原則。核心主張:core-key 只有 `Tab` / `Enter` /
`Space` / `Esc`(加 `?`),所有情境動作都能從 `Space` 選單走到,letter hotkey 只是
加速、不是唯一入口。

取法對象:cd-on-quit 對標 [superfile](https://github.com/yorukot/superfile) 的
`cd_on_quit`,finder 取法 [LazyVim](https://github.com/LazyVim/LazyVim) 的 search。以 Go
與 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 打造。

## 建置

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

## 平台:只支援 macOS / Linux

`GOOS=windows` **刻意編譯失敗**:平台分岔操作(metadata / hidden / roots / trash /
open)走 platform interface,unix 實作用 build tag,沒有 Windows 實作。Windows 使用者
走 WSL。

不做原生 Windows 的理由(2026-07-27 拍板):platform 插槽只是冰山一角,還有沒插槽的
結構性坑 —— `[s]hell` 內嵌 PTY(`creack/pty` 是 unix-only)、路徑 / `~` / 分隔符散在
多處 render、cd-on-quit 的 POSIX shell wrapper、`rwx` / owner 權限顯示在 Windows 沒有
對應模型。Git Bash 也不算:它是 POSIX 外殼、底下仍是原生 Windows PE。

## 安裝腳本

`install.sh` 下載 release binary 後,會把缺少的 `ripgrep` / `fd` 一併裝進同一個目錄 ——
從各工具自己的 GitHub release 抓、免 sudo,binary 名就是 `rg` / `fd`(不是 Debian 的
`fdfind`)。該平台沒有預編 binary 時(如 Intel macOS 的 `fd`)才退回印安裝提示。
Homebrew formula 則直接宣告 `ripgrep` + `fd` 依賴。

## cd-on-quit 為什麼需要一行 shell 設定

這是 OS 限制、不是偷懶:一個 process 只能改**自己**的工作目錄,沒有任何 syscall 能改
**父** process(你的 shell)的 cwd。filu 是 shell 的子程序,自己 `cd` 影響不到 shell;
唯一能原生改 shell cwd 的是 shell **內建指令**,而 filu 是外部 binary。

所以採兩段式 handshake:`filu shell` 印出一個 shell function,它透過
`FILU_LAST_DIR_FILE` 給 filu 一個暫存檔;filu 離開時把選定目錄寫進去,function 再讀檔
`cd`。這也是為什麼要用 `filu` 而不是 `./filu` 啟動 —— wrapper 攔截的是指令名 `filu`,
帶路徑的呼叫會繞過它。

## 各功能的實作備註

- **3 面板比例** — 上排 list 與 preview 以 2:1 共用(資訊豐富的 list 值得更寬);
  `[3]` 在下方橫跨滿寬。
- **檔案清單欄位** — 狀態 glyph、`Modified`、`Owner`(user:group)、`Perms`(eza 配色
  `r` 黃 / `w` 紅 / `x` 綠)、`Size`(eza color-scale,越大越暖;目錄顯示 `-`,絕不遞迴
  加總)、icon + 檔名。欄位標題兼排序指示;面板變窄時依 owner → size → modified →
  perms 的順序收合,檔名永遠最後才收。
- **每目錄排序** — `S` 可疊多層 chain,每個目錄各自記住,存進 `state.yaml`。
- **麵包屑** — lavender 純文字放在 `[1]` 第一列,下方一條低調分隔線。過長時前段縮成
  字首(`~/Documents/x` → `~/D/x`),再不夠中間縮 `…`,永不折行。
- **Marks bucket** — 複製會保留 bucket(可連續落地到多個目錄);搬移會更新 bucket 內
  路徑讓它保持有效。`p` pick 出來的子集優先於整個 bucket。
- **Zip** — 打包「Copy 會落地的那些」;檔名預設:單項用自己的名字、同目錄多項用該
  目錄名、跨目錄用時間戳。壓縮檔寫進暫存目錄(不動任何工作目錄),並成為 bucket 裡
  **唯一**的 pick,之後走既有的 `c` / `v` 落地路徑。每個 pick 以自己的名字放進壓縮檔
  (目錄保留內部結構),同名的 pick 兩個都留。
- **Tasks** — 同磁碟搬移是瞬間 `rename`;跨磁碟 / 複製才顯示進度。log 是帶時間戳的
  人話(`2026-07-28 14:32:07  Copied report.pdf → proj`)。中斷的任務存進
  `state.yaml`、下次啟動還原成 pending。
- **Finder** — filu 自畫的分割 picker(清單 + 預覽),不是 fzf binary(fzf-in-PTY 試過
  失敗)。每種模式都串流列檔:用 `fd` 的走訪序、不排序,首批近乎立即、載入中就能濾;
  沒有 `fd` 時退回純 Go walk。以 `/` 或 `~/` 開頭的 query 會錨定到該絕對路徑,對整條
  路徑 fuzzy、深度限錨點下數層。Goto 一律掃隱藏目錄,噪音靠 `ignore_dirs` 黑名單擋。
- **Preview** — 讀 magic bytes 判型別:目錄 → 內層 tree、壓縮包 → 內容清單、圖片 →
  base64 `data:` URI、SVG → 高亮 XML、文字 → Chroma(catppuccin-mocha)高亮 + 行號、
  二進位 → hex + ASCII、PDF → 抽出的文字 + 頁數。
- **Yank** — 走 OSC 52,所以能穿 tmux / SSH。
- **刪除** — 移到 OS 垃圾桶:macOS 用 `osascript`(不需 cgo)、Linux 寫 XDG
  `.trashinfo`。
- **分頁標記** — 第一個分頁掛啟動 glyph(它永遠開在啟動目錄,是 cd-on-quit picker 的
  固定參照),其餘各掛一隻動物(cat / dog / paw / egg)。路徑交給麵包屑列,分頁列只標
  位置與哪個 active。上限五個,到上限 toast 提示。
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
  框線不會破。

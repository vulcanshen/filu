# filu

> 用 ZLC(Zero Learning Curve)設計原則打造的終端機檔案管理器 —— 不看文件,第一次開就能用到底。

filu 是一個 TUI 檔案管理器,定位對標 yazi / superfile,但把重心放在 **零學習曲線**:靠一套跨面板不變的基礎操作(`Tab` / `Enter` / `Esc` / `Space`)+ 一個 `?` 全域入口,不需要背 hotkey 就能走完整個 app。它是 `kbu` `u`-family 的成員,共用 kbu 的整套 ZLC 設計系統(structural/user/popup 三層色階、powerline 分頁、popup taxonomy)。

> ⚠️ **狀態:開發中,已可跑。** 四面板骨架、預覽、carry-bucket 搬檔、async 落地、session 持久化都已實作;垃圾桶 / chmod / 壓縮 / 排序 / 即時刷新等仍在路上(見 `.forge/meta/IDEA.md`)。

## 需求

- **Nerd Font**:filu 用 Nerd Font glyph 當視覺語彙,是設計的一部分、不是 optional。
  - CJK Nerd Font(如 `Maple Mono NF CN`)會把 icon 畫成全形(2 格),filu 啟動時用 CPR 自動偵測 icon 實際格寬並據以排版,框線不會破。
- **平台**:macOS 或 Linux。不提供原生 Windows build(`GOOS=windows` 刻意編譯失敗);Windows 請走 WSL。
- **Go**:1.26+。`CGO_ENABLED=0` 可靜態編譯。
- **Search / Find 用到的外部工具**:`ripgrep`(`f` Find 內容搜尋必需)、`fd`(列檔案清單;缺了退回純 Go walk)。`$EDITOR`(`e` edit,預設 `vi`)。

## 介面

四個面板 + 上下兩條 content row。版面 grid 為 `[1][2][3] / [1][2][3] / [4][4][3]`:

```
 [] ~/Documents/sideproj/filu                         ← 路徑 bar([]=folder glyph)
┌──────────┬──────────────────┬──────────────────────┐
│ [1] pin  │ [2] 檔案清單     │ [3] Preview │ Meta    │
│  Local   │  （1–3 個分頁）  │                       │
│  Pinned  │                  │   （active tab 全高） │
├──────────┴──────────────────┤                       │
│ [4] Carries │ Tasks         │                       │
└─────────────────────────────┴──────────────────────┘
 space menu   ? help   tab/1-4 panels   q quit         ← footer
```

- **[1] pin**:系統 Places(CWD / Home / 根目錄)+ 使用者釘選的目錄(`Pinned` 標題為 lavender)。釘選項目以**縮減 path** 呈現(`~/Documents/sideproj/filu` → `~/D/s/filu`,過長則前綴變 `…`)。純導覽 —— 選一個,`[2]` 就跳過去。
- **[2] 清單**:當前目錄的檔案,1–3 個目錄分頁(各自獨立 CWD + cursor,標籤為羅馬數字 `Ⅰ`/`Ⅱ`/`Ⅲ`;預設開一個,`t` 在當前目錄開新分頁、`w` 關閉當前分頁)。每列 `<icon> <檔名>`,icon 與配色照 eza(見下方[配色](#功能))。檔案操作主戰場。
- **[3] detail**:`Preview` / `Meta` 兩個 tab。Preview 依型別分五類呈現;Meta 是 `stat` 等級的詳細資訊。這是參考視角,失焦也不 dim。按 `y` 開 yank 視窗(vim 式 cursor + `v` visual selection):有選取時 `y` 複製選取內容,沒選取則複製全部。
- **[4] carry**:`Carries`(搬運暫存區)/ `Tasks`(複製/搬移任務,含 running / done / pending / error 狀態)兩個 tab。

## 功能

- **Preview 五分類**(讀 magic bytes 判定):
  - 目錄 → 內層 tree(eza `-T` 風格)
  - 壓縮包(zip / tar / tar.gz…)→ 列包內清單
  - 圖片 → base64 `data:` URI(SVG 例外,當 XML 語法highlight)
  - 文字 → chroma 語法highlight + 行號(catppuccin-mocha)
  - 二進位 → hex + ASCII(xxd 風)+ 行號
  - PDF → 純 Go 抽文字 + 頁數
- **Icon + 配色(照 eza)**:
  - **Icon**:檔案型別 glyph 取自 eza 原始碼的完整對照表(`src/output/icons.rs`,~760 個)—— 目錄名 / 檔名走 exact-case、副檔名走小寫,涵蓋各語言、設定檔、壓縮/影音/圖片與 `README` / `Dockerfile` / `go.mod` 等特例。
  - **顏色**:把一份 `vivid generate catppuccin-mocha` 的 `LS_COLORS`(該 palette 每個副檔名一個色)**烘進 binary**,依 eza 的優先序上色(目錄 → symlink → executable → 整名 suffix → 副檔名 → normal;executable 蓋過副檔名)。結果與使用者終端機的 `eza` / `ls` **完全一致**,且**不需要**環境有 `LS_COLORS` —— 別人裝了也是同一套配色。
  - 面板層級另有 structural 藍 / user lavender / popup 層色的三層階層。
- **Zoom**:每個有分頁的面板都能 `z` 展開佔滿,把分頁攤成等寬並排欄。
- **Carry-bucket 搬檔**:`p` pick 把檔案拿進 bucket(panel 2 會**打綠勾**標記,等同 multi-select)→ 切到目標目錄 → `c` copy / `m` move 落地;Carries tab 可 `p` 只落地子集。落地跑 goroutine,進度在 Tasks tab 即時更新。
- **Search(`/`)/ Find(`f`),僅 panel 2**:filu 原生的 file finder(snacks / Telescope 形式,非 fzf binary),兩種模式共用同一個 picker —— snacks 樣式輸入列(peach chevron + blinking block cursor)、打字即時過濾、`jkud` 選、Enter 把 tab **潛進**選中檔案的所在子目錄。都以當前 tab 的**子樹**為範圍(遞迴)。
  - **`/` Search**:對子樹**檔名**做 **fuzzy** 比對(純記憶體、依匹配品質排序),單欄、無預覽。
  - **`f` Find**:用 `ripgrep` 對**內容**過濾出含關鍵字的檔案(去重),分割 popup(左清單 + 右預覽,窄寬改上下),預覽自動 scroll 到命中行並 lavender 反白。
- **Popup**(全走 kbu form:line→expand 動畫 + 層色 border):`Space` 情境選單(§A.1)、`?` 全域說明(§A.2)、刪除確認、Rename / Add 輸入框(chevron prompt + 閃爍游標,rename 描述帶型別 icon + 顏色)。
- **Session 持久化**:多開的分頁(dir + cursor)、focus、detail tab、carry bucket、pinned、task log 都存進 `state.yaml`,下次啟動接續。第一個分頁固定在啟動當下的 CWD、且開機時為 active(所以每次啟動都從當前目錄開始)。

## 操作

核心鍵(跨面板不變):

| 鍵 | 作用 |
|---|---|
| `Enter` | 進入目錄 / 開檔(交給系統) |
| `Esc` | 返回上層 / 關閉浮層 |
| `Tab`、`1`–`4` | 切換面板 |
| `h` / `l` | 切換當前面板的分頁 |
| `Space` | 開「當前面板此刻能做什麼」的選單 |
| `?` | 全域說明 |
| `q` | 離開:跳選單選一個目錄 cd 過去(1–4 或 j/k+Enter) |

面板 `[2]` 的字母 hotkey(皆列在 `Space` 選單裡):`p` pick(拿進 carries bucket)、`y` yank(複製 full path 到剪貼簿)、`e` edit(文字檔在內嵌 `$EDITOR` 編輯,非文字走系統開啟)、`c` copy(落地複製)、`m` move(落地搬移)、`P` pin、`r` rename、`a` add、`D` delete、`S` sort、`/` search(fuzzy 檔名)、`f` find(內容 grep + 預覽)、`t` 開新分頁、`w` 關分頁、`.` 顯示隱藏檔、`z` zoom。

## cd-on-quit(離開時切換 shell 目錄)

按 `q` 會跳出選單:從 **panel 1 的起始目錄**與**各分頁的當前目錄**裡挑一個(重複的目錄只列一次),離開 filu 時把 shell 的 cwd 切過去(對標 superfile 的 `cd_on_quit`)。

### 為什麼需要一行 shell 設定(不是 filu 偷懶,是 OS 限制)

一個程式只能改自己的工作目錄;**沒有任何 syscall 能改「父程序」(你的 shell)的 cwd** —— 這是 POSIX 鐵則。filu 是 shell 的子程序,自己 `cd` 不會影響 shell,離開後 shell 還在原地。唯一能原生改 shell cwd 的是 shell **內建指令**(如 `cd` 本身),而 filu 是外部 binary。所以 `yazi` / `lf` / `ranger` / `nnn` / `superfile` / `zoxide` 全都靠同一招:程式寫檔 → shell wrapper 讀檔再 `cd`。裝一行 `eval` 就是這類工具公認的「原生整合」。

### 啟用

filu 進 PATH 之後(`go install ./cmd/filu` 或 release binary),在 `~/.zshrc` / `~/.bashrc` 加一行:

```sh
eval "$(filu shell)"
```

之後用 **`filu`** 啟動(**不是 `./filu`** —— wrapper 是攔截 `filu` 這個指令名的 shell function,帶路徑呼叫不會經過它)。

原理:wrapper 給 filu 一個暫存檔路徑(`FILU_LAST_DIR_FILE`),filu 離開時把選定目錄寫進去,wrapper `cat` 出來再 `cd`。**沒裝 wrapper 也能正常用 filu**,只是離開時不會切目錄(選了 cd 目標也不會有反應,因為少了那段 shell 整合)。

## 安裝

> 發佈基建已備妥(goreleaser + Homebrew tap + `install.sh`);以下管道在第一個 GitHub release(tag `v*`)後生效。目前請走下面的[建置](#建置)從原始碼編。

- **Homebrew**(macOS / Linux)—— 會一併裝 Search 需要的 `ripgrep` + `fd`:
  ```sh
  brew install vulcanshen/tap/filu
  ```
- **install script**(抓 release binary;缺 `ripgrep` / `fd` 時印各發行版安裝提示):
  ```sh
  curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/install.sh | sh
  ```
- **go install**:
  ```sh
  go install github.com/vulcanshen/filu/cmd/filu@latest
  ```

移除:`curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/uninstall.sh | sh`。

## 建置

```sh
CGO_ENABLED=0 go build -o filu ./cmd/filu   # 靜態編譯
go test ./...                               # 測試
gofmt -l . && go vet ./...                  # 格式 / 靜態檢查
```

## License

`<TBD>`

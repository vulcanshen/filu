# filu

<p align="center"><img src="docs/icon.svg" width="128" alt="filu icon" /></p>

[![GitHub Release](https://img.shields.io/github/v/release/vulcanshen/filu)](https://github.com/vulcanshen/filu/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/vulcanshen/filu)](https://go.dev/)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue)](LICENSE)

**語言**: [English](README.md) · 繁體中文

**不用學的終端機檔案管理器。** 四個鍵 —— `Tab`、`Enter`、`Space`、`Esc` —— 就能用到 filu 的所有功能。一眼看清重點的檔案清單、把檔案收集起來再放到別處、依檔名或內容找東西、邊走邊預覽,離開時還能把 shell 留在你最後所在的目錄。

> _遇事不決,就按_ **`Space`**。

## Demo

![demo](docs/demo-basics.gif)

在 filu 裡走動：面板、分頁、`Space` menu，還有一直看得到的路徑。

## 為什麼用 filu

- **什麼都不用背。** 在任何面板按 `Space`,filu 就列出這裡能做的事。每個快捷鍵都只是選單項目的捷徑,從來不是唯一入口。
- **一眼看懂一個目錄。** 修改時間、擁有者、權限、大小就排在檔名旁邊,配色跟你的 `eza` / `ls` 一樣。可依任一欄排序,而且每個目錄記得自己的排序。
- **像桌面一樣複製、搬移。** 邊逛邊 mark 檔案,走到目的地按 `c` 或 `v`。可以跨分頁 mark、把同一批檔案放到好幾個地方,或先打包成 zip。
- **很快就找到。** 模糊搜尋檔名、用 ripgrep 搜尋檔案內容,或跳到 home 底下任何目錄 —— 結果邊打字邊冒出來,旁邊附即時預覽。
- **打開前先看一眼。** 語法高亮的文字、壓縮檔內容、目錄樹、PDF、圖片、二進位檔的 hex。選取其中一段複製到剪貼簿 —— 連 SSH 上都行。
- **離開時留在工作的地方。** 按 `q` 離開,shell 會停在你挑的那個目錄。
- **下次接著用。** 分頁、marks、最愛、每個目錄的排序,重開後都還在。

## 安裝

### 系統需求

- 終端機要用 **Nerd Font** —— filu 的 icon 都是 Nerd Font glyph。CJK Nerd Font(如 Maple Mono NF CN)也沒問題。
- **ripgrep** 用於內容搜尋,**fd** 讓 finder 更快(快速安裝與 Homebrew 都會幫你裝好)。
- **macOS 或 Linux** —— Windows 請用 WSL。

### 快速安裝

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/install.sh | sh
```

安裝 filu,缺 `ripgrep`、`fd` 的話也一併裝好 —— 不需要 sudo。

### Homebrew

```bash
brew install vulcanshen/tap/filu
```

`ripgrep` 與 `fd` 會作為依賴一起裝。

### Go

```bash
go install github.com/vulcanshen/filu/cmd/filu@latest
```

需要 Go 1.26+。內容搜尋與快速 finder 需要的 `ripgrep`、`fd` 請自行安裝。

### 移除

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/uninstall.sh | sh
```

## 開始使用

```bash
filu              # 開在當前目錄
filu ~/proj       # 開某個目錄
filu ~/notes.md   # 開它所在的目錄,游標停在這個檔案
```

filu 開啟時 focus 在檔案清單。剩下的交給五個鍵:

| 鍵 | 作用 |
|---|---|
| **`Tab`** | 移到下一個面板(或按 `1`–`3` 直達) |
| **`Enter`** | 進入目錄 / 確認選擇 |
| **`Space`** | *這裡能做什麼?* —— 當下所在位置的選單 |
| **`Esc`** | 退出 —— 回上層目錄,或關掉 popup |
| **`?`** | 說明 —— 所有全域動作一次列出 |

### 讓 `q` 切換 shell 的目錄

在 `~/.zshrc` 或 `~/.bashrc` 加上這一行:

```sh
eval "$(filu shell)"
```

之後用 **`filu`**(不是 `./filu`)啟動。按 `q` 會列出啟動目錄與每個分頁的目錄;挑一個,filu 關掉時 shell 就在那裡。沒加這行 filu 一樣能用 —— 只是離開時 shell 留在原地。

## 畫面

三個面板:

- **`[1]` Files** —— 主要的檔案清單,頂端是路徑列。最多五個分頁,各自瀏覽自己的目錄。
- **`[2]` Preview** —— 游標所在項目的預覽。
- **`[3]` Marks | Tasks | Favorites** —— 你收集的檔案、複製搬移的紀錄、你存下的目錄。

`Tab`(或 `1`–`3`)切換面板,`h` / `l` 切換面板內的分頁,`z` 把當前面板放大到全螢幕。

## 能做什麼

### 瀏覽

- `Enter` 進入、`Esc` 回上層;`j` / `k`、`u` / `d` 捲半頁、`gg` / `G` 到頂到底。
- `b` 跳到任一層上層目錄。`.` 顯示或隱藏隱藏檔。
- `S` 依名稱、修改時間、擁有者、權限或大小排序 —— 可疊多層決定平手時的順序。排序只套用在那個目錄。
- `t` 開新分頁(同目錄、某個最愛,或搜尋),`w` 關掉。
- 其他程式改動了檔案,清單會自己更新。

### 收集、複製、搬移

- `m` mark 一個檔案。四處移動時 marks 都會留著,可以從好幾個目錄、分頁收集。
- 走到檔案該去的地方,按 `c` 複製、`v` 搬移過去。複製後 marks 還在,可以再放到別的地方。
- 在 **Marks** 分頁:`p` 只挑其中一部分落地、`m` 取消某個 mark、`C` 全部清空、`Z` 把挑中的打包成 zip,再用 `c` / `v` 放過去。
- 複製與搬移在背景進行;**Tasks** 分頁用白話紀錄進度。離開時沒跑完的任務,下次開啟會回來。

### 尋找

- **`/` Search** —— 依**檔名**(模糊比對,範圍是當前目錄以下)或依**內容**(ripgrep;預覽會跳到命中那一行)。query 以 `/` 或 `~/` 開頭,就改從那個路徑搜尋,磁碟上哪裡都行。
- **`go` Goto** —— 跳到某個最愛,或模糊搜尋 home 底下所有目錄(含隱藏目錄)。`Enter` 把分頁帶過去。
- 結果邊找邊出現 —— 直接開始打字就好。

### 最愛

- `f` 把游標上的目錄加星號;`F` 把你所在的目錄加星號。加星的目錄在清單上會標出來。
- **Favorites** 分頁列出所有最愛:`o` 開在新的或既有的分頁、`D` 移除。也可以用 Goto → Favorites 跳過去。

### 預覽與複製

- 預覽支援:語法高亮加行號的文字、目錄樹、壓縮檔內容、PDF、圖片、SVG 原始碼、二進位檔的 hex。
- 在預覽按 `y` 開啟可捲動的檢視:`v` 開始選取、`y` 複製(沒選取就複製全部)。
- 在檔案上按 `y` 複製它的完整路徑。複製可以穿過 tmux 與 SSH。

### 開啟、編輯,以及其他

- `o` 用預設 app 開啟檔案或目錄。`O` 讓你挑 app —— 可以在設定裡加上自己的(VSCode、IntelliJ IDEA…)。
- `s` 在當前目錄開你的 shell;打 `exit` 回來。
- `r` 改名、`a` 新增檔案(名稱以 `/` 結尾就是目錄)、`D` 移到垃圾桶 —— 都會先確認。

## 按鍵一覽

```
 游標      j k        u d         gg G        h l(切本面板分頁)
 清單      o open     O open-with  m mark     c copy    v move    f favorite
           y yank     r rename     a add      s shell   D delete  S sort   . hidden   z zoom
 finder    / search   go goto      b breadcrumb
 分頁      t 開分頁   w 關分頁
```

| 鍵 | 任何地方 |
|---|---|
| `?` | 說明 |
| `q` | 離開,並選擇 shell 要停在哪裡(打字時 `q` 就是一個字母) |
| `Ctrl+C` | 同 `q`,打字時也有效;在離開畫面上再按一次就立即離開 |

各面板的 `Space` 選單依序列出：對游標項目能做的事、對整個面板能做的事，最後一列是 **Global operation**,打開全 app 的動作選單(目前只有 Quit `q`)。暫時不能做的項目(例如只開一個分頁時的 Close tab)會變暗顯示。

| Focus | 選單項目 |
|---|---|
| **`[1]` Files** | Open `o`、Open with `O`、Mark `m`、Yank `y`、Rename `r`、Delete `D`、Favorite `f` · Copy `c`、Move `v`、Search `/`、Goto `go`、Favorite dir `F`、Breadcrumb `b`、Switch tab `l`、Tab `t`、Close tab `w`、Add `a`、Sort `S`、Shell `s`、Hidden `.`、Zoom `z` |
| **`[2]` Preview** | Yank `y`、Zoom `z` |
| **`[3]` Marks** | Pick `p`、Yank `y`、Unmark `m` · Zip `Z`、Clear `C`、Switch tab `l`、Zoom `z` |
| **`[3]` Tasks** | Delete `D` · Switch tab `l`、Zoom `z` |
| **`[3]` Favorites** | Open in `o`、Delete `D` · Switch tab `l`、Zoom `z` |

## 設定

設定檔是 `config.yaml`:

| OS | 目錄 |
|---|---|
| Linux | `$XDG_CONFIG_HOME/filu/` 或 `~/.config/filu/` |
| macOS | `~/Library/Application Support/filu/`(設了 `$XDG_CONFIG_HOME` 時用 `$XDG_CONFIG_HOME/filu/`) |

filu 第一次執行時會寫一份帶註解的設定檔,之後絕不覆蓋你的版本。旁邊的 `state.yaml` 存的是你的 session —— 交給 filu 管就好。

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

## 限制

刻意不做的：
- **原生 Windows** —— filu 支援 macOS 與 Linux；Windows 請在 WSL 裡使用
- **滑鼠** —— 所有操作都在鍵盤上
- **目錄大小** —— 目錄的大小欄顯示 `-`，filu 不會把整棵樹加總

## 相關連結

- [CHANGELOG.md](CHANGELOG.md) —— 每個版本改了什麼
- [`docs/dev-remarks.md`](docs/dev-remarks.md) —— 開發者備忘：運作方式、設計理由、從原始碼建置、發布

## terminu family

filu 遵循 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.7/principle)：跟家族其他成員一樣的按鍵、一樣的 menu —— [kbu](https://github.com/vulcanshen/kbu)（Kubernetes）、[sshu](https://github.com/vulcanshen/sshu)（ssh）、[webu](https://github.com/vulcanshen/webu)（網頁）與 [locku](https://github.com/vulcanshen/locku)（螢幕鎖）。

## License

[GPL-3.0](LICENSE)

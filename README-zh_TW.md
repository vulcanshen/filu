# filu

<p align="center"><img src="docs/icon.svg" width="128" alt="filu icon" /></p>

[![GitHub Release](https://img.shields.io/github/v/release/vulcanshen/filu)](https://github.com/vulcanshen/filu/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/vulcanshen/filu)](https://go.dev/)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue)](LICENSE)

**Language**: [English](README.md) · 繁體中文

**不用學的終端機檔案管理器。** 不管在哪裡，按 `Space` 列出能做的事，按 `?` 列出能按的鍵。一眼看清重點的檔案清單、把檔案收集起來再放到別處、依檔名或內容找東西、邊走邊預覽，離開時還能把 shell 留在你最後所在的目錄。

> _遇事不決，就按_ **`Space`**。

## Demo

![demo](docs/demo-basics.gif)

## 為什麼用 filu

- **什麼都不用背。** 在任何 panel 按 `Space`，filu 就列出這裡能做的事。每個熱鍵都只是那個選單裡某一列的捷徑，從來不是唯一入口。
- **一眼看懂一個目錄。** 修改時間、擁有者、權限、大小就排在檔名旁邊，依類型上色，跟 eza 的 catppuccin-mocha 配色一樣。可依任一欄排序、疊好幾層，而且每個目錄記得自己的排序。
- **像桌面一樣複製、搬移。** 邊逛邊 mark 檔案，走到目的地按 `c` 或 `v`。可以跨目錄、跨 tab mark，把同一批檔案放到好幾個地方，或先打包成 zip。
- **很快就找到。** 搜尋檔名、用 ripgrep 搜尋檔案內容，或跳到 home 底下任何目錄 —— 結果找到一筆就列一筆，旁邊附預覽。
- **打開前先看一眼。** 語法上色的文字、壓縮檔內容、目錄樹、PDF 的文字、圖片的 data: URI、二進位檔的 hex。選取其中一段複製到剪貼簿 —— 連 SSH 上都行。
- **離開時留在工作的地方。** 離開時選 shell 停在哪裡：你啟動 filu 的目錄，或任何一個 tab 的目錄。
- **下次接著用。** tab、marks、最愛、tasks、每個目錄的排序，重開後都還在。

## 安裝

### 系統需求

- 終端機要用 **Nerd Font** —— filu 的 icon 都是 Nerd Font glyph。
  filu 啟動時會問終端機一個 icon 佔幾格，以及一個 powerline 符號（panel 標籤的兩端、cursor 尾端的箭頭）佔幾格。框線還是歪的話，`filu iconwidth` 會印出量到的格數，也可以自己指定：icon 用 `FILU__ICON_WIDTH=1` 或 `2`，powerline 符號用 `FILU__POWERLINE_WIDTH=1` 或 `2`。`TERMINU__ICON_WIDTH` 與 `TERMINU__POWERLINE_WIDTH` 是 terminu 家族共用的同一組設定：設一次，家族每個 app 都讀得到（`FILU__` 的仍然優先）。在家族 app 的終端機裡跑 —— kbu 的 Alterm、filu 自己的 shell —— 外層 app 會替你把兩個都設好。
- 終端機要支援 **truecolor（24-bit 色）** —— filu 的配色與 popup 底下的淡化都需要它；256 色下深淺會糊在一起。
- **ripgrep** 用於內容搜尋，**fd** 讓 finder 更快（快速安裝與 Homebrew 都會幫你裝好）。
- **macOS 或 Linux** —— Windows 請用 WSL。

### 快速安裝

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/install.sh | sh
```

安裝 filu，缺 `ripgrep`、`fd` 的話也一併裝好 —— 不需要 sudo。

### Homebrew

```bash
brew install vulcanshen/tap/filu
```

`ripgrep` 與 `fd` 會作為依賴一起裝。

### Go

```bash
go install github.com/vulcanshen/filu/cmd/filu@latest
```

需要 Go 1.26+。內容搜尋與快速的 finder 要自己裝 `ripgrep`、`fd`。

### 第一次啟動

filu 會在設定目錄（見[設定](#設定)）寫一份帶註解的 `config.yaml`，然後打開你啟動它的目錄。沒有別的要設定。

### 移除

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/uninstall.sh | sh
```

刪除你的設定前會先問。

## 快速開始

```bash
filu              # 打開目前的目錄
filu ~/proj       # 打開一個目錄
filu ~/notes.md   # 打開它所在的目錄，cursor 停在這個檔上
filu help         # 命令列的用法
```

filu 打開時 focus 在檔案清單上。其他的事靠這幾個鍵：

| 鍵 | 做什麼 |
|---|---|
| **`Space`** | *這裡能做什麼？* —— 你所在 panel 的選單；最後一列打開整個 app 的動作 |
| **`?`** | 這裡能按的鍵（在 popup 裡就是那個 popup 的鍵） |
| **`Enter`** | 進入目錄、看檔案的資訊，或做你所在那一列的事 |
| **`Esc`** | 退一步：關掉最上層的 popup，或回上一層目錄 |
| **`Tab`** | 下一個 panel（`Shift-Tab` 上一個，`1`–`3` 直接跳過去） |
| **`q`** | 離開（先問 —— 加了下面那一行時，問 shell 要停在哪裡） |

### 讓離開 filu 時 shell 換目錄

在 `~/.zshrc` 或 `~/.bashrc` 加這一行：

```sh
eval "$(filu shell)"
```

之後用 **`filu`**（不是 `./filu`）啟動。按 `q` 時 filu 會問 shell 要停在哪裡：你啟動 filu 的目錄，或任何一個 tab 的目錄 —— 按它的數字，或移到它上面按 `Enter`。還有複製或搬移在跑時，filu 會再問一次，因為離開會中斷它。沒加這一行，filu 照樣能用 —— 只是離開前先問，shell 停在原地。

## 使用方式

### 畫面

三個 panel：

- **`[1]` 檔案** —— 清單，最上面是目錄的路徑。最多五個 tab，各自在自己的目錄。
- **`[2]` 預覽** —— `[1]` 的 cursor 所在的那一項。
- **`[3]` Marks | Tasks | Favorites** —— 收集起來的檔案、複製／搬移／打包的紀錄，以及加星號的目錄。

`Tab`（或 `1`–`3`）在 panel 之間移動，`h|l` 切換 panel 的 tab，`z` 把你所在的 panel 放大到整個畫面 —— 放大時 `[1]` 的每個 tab 並排。

### 能做什麼

#### 瀏覽

- `Enter` 進入目錄；在檔案上會顯示資訊：完整路徑、類型、大小、時間、權限、擁有者。`Esc` 回上一層；`j|k`，`u|d` 半頁，`gg|G` 到頭、到尾。
- `b` 跳到這個目錄所在的任何一層。`.` 顯示或隱藏隱藏檔。
- `S` 依名稱、修改時間、權限、擁有者或大小排序 —— 先選欄位、再選方向。欄位留著、寫出目前的排序，可以馬上再疊一層比較；`Esc` 關掉。目錄一律在前面。排序只記在那個目錄；`[r]eset` 回到照名稱排。
- 很長的檔名會被截短，其他欄照樣留著。panel 再窄下去，就整欄讓給檔名 —— 先收擁有者，再來是大小、修改時間、權限。
- `t` 開新 tab（同一個目錄、一個最愛，或搜尋），`w` 關掉你所在的 tab。
- 別的程式改了檔案，清單會自己更新。

#### 收集、複製、搬移

- `m` mark 一個檔。移動時 marks 留著，可以從好幾個目錄、好幾個 tab 收集。
- 走到檔案要去的地方，按 `c` 複製或 `v` 搬過去。複製會保留 marks，同一批可以放到不只一個地方。
- 在 **Marks** tab 裡：`Enter` 在 `[1]` 顯示那個檔（在已經開著它目錄的 tab，或開一個新的）並帶你過去，`p` pick 其中幾個、只有 pick 的會落地，`m` 取消一個 mark，`C` 全部清掉，`y` 複製路徑，`Z` 把會落地的（有 pick 就是 pick 的，否則全部 marks）打包成 zip，再用 `c` 或 `v` 落地。zip 的名字 filu 會用灰字建議一個：按 `Enter` 就用它，或打你自己的。
- 複製、搬移、打包在背景跑；**Tasks** tab 用白話記下每一筆，還在跑的有轉圈的圖示與進度，在一筆上按 `Enter` 帶你到它落地的地方。離開時沒做完的工作，下次會標成中斷。

#### 尋找

- **`[/] Search`** —— `[f]ile names`：這個目錄底下的全部，邊打字邊縮小（打的每個詞都要出現，順序不拘）。`[c]ontents`（ripgrep）：含有你打的字的檔案，邊打邊重新找，預覽從第一個找到的地方開始。
- **`[go]to`** —— 跳到一個最愛，或搜尋 home 底下的每個目錄（包括隱藏的）。
- 在 File names 與 Goto 裡，打 `/` 或 `~/` 開頭的路徑就改從那裡找：它寫到的最深那個目錄底下幾層，用你打的其餘部分縮小 —— 磁碟上哪裡都行（`/usr/lo` 找到 `/usr/local`；`~/proj/` 列出 `~/proj`）。
- 結果找到一筆就列一筆 —— 馬上就能開始打字。打字時 `↑|↓` 在結果之間移動；`Enter` 或 `Tab` 到清單，在清單上 `j|k` 移動、`Enter` 打開所在那一筆；`Tab` 或 `/` 回去打字；`Esc` 關掉。每一筆的預覽在清單旁邊，附類型、大小、修改時間與權限。

#### 最愛

- `f` 替 cursor 所在的目錄加星號；`F` 替你所在的目錄加。加了星號的目錄，名稱後面有一顆星；你在那個目錄裡時，路徑後面也有。
- **Favorites** tab 列出它們：`Enter` 到那裡（它的 tab，或一個新的），`o` 自己選 tab，`D` 拿掉星號（先問）。`[go]to` → `[f]avorite` 也能跳過去。

#### 預覽與複製

- 預覽顯示語法上色加行號的文字、目錄樹、壓縮檔內容、PDF 的文字、圖片的 data: URI、SVG 原始碼，二進位檔顯示 hex。
- 在 `[2]` 按 `Enter` 用可捲動的檢視打開預覽：`v` 叫出游標，再按一次 `v` 開始選，用 `h|j|k|l` 移動、`w|b|e` 一個字一個字、`0|$` 到行首行尾，`y` 複製選起來的部分。沒有游標時，`y` 複製全部 —— 跟在 `[2]` 上直接按 `y` 一樣。
- 在 `[1]` 的檔案上按 `y` 複製它的完整路徑。複製能穿過 tmux 與 SSH。

#### 開啟、編輯與其他

- `o` 用預設 app 開檔案或目錄（先問）。`O` 自己選 app —— 在設定裡加你自己的（VSCode、IntelliJ IDEA……）。
- `s` 在目前的目錄打開你的 shell（先問）；打 `exit` 回來，或按 `Alt-Esc`：filu 會先問，`Enter` 結束 shell（連同裡面還在跑的東西），`Esc` 回到 shell。
- `r` 重新命名（舊名稱是灰字：`Tab` 收進來改，或直接打新的），`a` 新增檔案（名稱以 `/` 結尾就是目錄），`D` 問過之後移到垃圾桶。名稱是空的、已經有了、在目前目錄外面，或含控制字元，當場就會擋下 —— 框留著，說明原因。重新命名或新增還是失敗時（例如沒有權限），filu 在框上面說明原因，關掉後回到框、打的字還在。

### 按鍵

| 鍵 | `[1]` 檔案 |
|---|---|
| `Enter` | 進入目錄 · 檔案的資訊 |
| `o` · `O` | 開啟 · 選 app 開啟 |
| `m` · `c` · `v` | mark · 把 marks 複製到這裡 · 搬到這裡 |
| `y` · `r` · `D` | 複製路徑 · 重新命名 · 刪除 |
| `f` · `F` | 替 cursor 所在的目錄加星號 · 替你所在的目錄加 |
| `/` · `go` · `b` | 搜尋 · goto · 上層目錄 |
| `t` · `w` · `h\|l` | 新 tab · 關掉 tab · 切換 tab |
| `a` · `S` · `s` · `.` · `z` | 新增 · 排序 · shell · 隱藏檔 · 放大 |

| 鍵 | `[2]` 預覽 | `[3]` Marks | `[3]` Tasks | `[3]` Favorites |
|---|---|---|---|---|
| `Enter` | 檢視 | 在 `[1]` 顯示 | 到它落地的地方 | 到那裡 |
| 其他 | `y` 全部複製 · `z` 放大 | `p` pick · `m` 取消 mark · `y` 複製路徑 · `Z` 打包 · `C` 清掉 · `z` 放大 | `D` 從紀錄拿掉 · `z` 放大 | `o` 在某個 tab 開 · `D` 拿掉星號 · `z` 放大 |

| 鍵 | 任何地方 |
|---|---|
| `?` | 這裡能按的鍵；`?` 或 `Esc` 關掉 |
| `q` · `Ctrl-C` | 離開（先問，或問 shell 要停在哪裡）；`Ctrl-C` 打字時也有效，再按一次立刻離開 |

每個 panel 的 `Space` menu 先列出對 cursor 所在項目能做的事，再列出對整個 panel 能做的事，最後一列是 **Global operation**：整個 app 的動作（`[q]uit`）。現在不能做的列 —— 例如只有一個 tab 時的 `[w] Close tab` —— 會變暗。

## 設定

設定在 `config.yaml`，放在 `$XDG_CONFIG_HOME/filu/`（或 `~/.config/filu/`）；`FILU__CONFIG` 可以指定另一個目錄。filu 第一次執行時會寫一份帶註解的，之後不會蓋掉你的。旁邊的 `state.yaml` 存你上次在哪裡 —— tab、marks、每個目錄的排序 —— 交給 filu 就好（`FILU__STATE` 把它放到另一個目錄）。最愛與 task 紀錄在 `~/.filu/data.yaml`（`FILU__DATA` 可以指定另一個目錄）：要備份的是這一份。

macOS 上，0.4.0 以前的版本把全部放在 `~/Library/Application Support/filu/`。這一版第一次啟動時，會把裡面的東西搬到 `~/.config/filu/` 並告訴你，搬空的舊目錄刪掉。`~/.config/filu/` 已經有的檔兩邊都不動，filu 寫出兩個路徑，讓你自己把兩份合起來。搬不動時照舊用舊的，並寫出兩個路徑讓你自己搬。最愛與 task 紀錄也同樣從 `state.yaml` 搬到 `~/.filu/data.yaml`：`data.yaml` 已經有自己的時，`state.yaml` 裡的留著，filu 會告訴你。

```yaml
# finder 掃描幾筆就停。Goto 會走遍整個 $HOME，靠這個設上限。
finder_cap: 50000

# finder 略過的目錄 —— 快取、建置產物、IDE 的資料、容器的資料，都是不會 cd
# 進去的地方。只寫名字的在任何深度都符合；帶斜線的（例：go/pkg）比對路徑。
# 設成 [] 就什麼都不略過。
ignore_dirs:
  - node_modules
  - .git
  - Library
  - go/pkg
  # …

# Open with 選單裡的 app（在檔案或目錄上按 [O]；[o] 用系統預設的開）。
# filu 執行 `<cmd> <path>`。
open_with:
  - name: VSCode
    cmd: code
  - name: IntelliJ IDEA
    cmd: idea

# 預覽與檢視裡，一個 Tab 佔幾格。
input_tab_width: 4

# true 時，finder 的預覽不顯示私鑰，也不顯示只有擁有者能讀的檔。
hide_secrets_in_preview: false
```

`input_tab_width` 與 `hide_secrets_in_preview` 在 terminu 家族每個 app 都是同一個設定。

## 限制

刻意不做的：
- **原生 Windows** —— filu 在 macOS 與 Linux 上跑；Windows 請在 WSL 裡用
- **滑鼠** —— 全部用鍵盤
- **目錄大小** —— 目錄的大小顯示成 `-`；filu 不會把整棵樹加總
- **畫出圖片** —— 圖片顯示成 data: URI，不畫出來
- **比 60×24 小的終端機** —— filu 只寫出需要多大的終端機；把視窗拉大，畫面就回到原本的樣子

## 相關連結

- [CHANGELOG.md](CHANGELOG.md) —— 每一版改了什麼
- [`docs/dev-remarks.md`](docs/dev-remarks.md) —— 開發者備忘：怎麼運作、為什麼、從原始碼建置、發布

## terminu family

filu 遵循 [terminu design](https://github.com/vulcanshen/terminu)，建在 [terminu-go](https://github.com/vulcanshen/terminu-go) 上：跟家族其他成員一樣的按鍵、選單與 popup —— [kbu](https://github.com/vulcanshen/kbu)（Kubernetes）、[sshu](https://github.com/vulcanshen/sshu)（ssh）、[webu](https://github.com/vulcanshen/webu)（網頁）、[locku](https://github.com/vulcanshen/locku)（鎖定畫面）、[kamonu](https://github.com/vulcanshen/kamonu)（畫家族 mark 的編輯器）與 [ccdrawu](https://github.com/vulcanshen/ccdrawu)（Claude Code 的側邊欄）。

## License

[GPL-3.0](LICENSE)

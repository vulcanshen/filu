# filu

> 用 ZLC(Zero Learning Curve)設計原則打造的終端機檔案管理器 —— 不看文件,第一次開就能用到底。

filu 是一個 TUI 檔案管理器,定位對標 yazi / superfile,但把重心放在 **零學習曲線**:靠一套跨面板不變的基礎操作(`Tab` / `Enter` / `Esc` / `Space`)+ 一個 `?` 全域入口,不需要背 hotkey 就能走完整個 app。它是 `kbu` `u`-family 的成員,共用 kbu 的整套 ZLC 設計系統(structural/user/popup 三層色階、powerline 分頁、popup taxonomy)。

> ⚠️ **狀態:開發中,已可跑。** 四面板骨架、預覽、carry-bucket 搬檔、async 落地、session 持久化都已實作;垃圾桶 / chmod / 壓縮 / 排序 / 即時刷新等仍在路上(見 `.forge/meta/IDEA.md`)。

## 需求

- **Nerd Font**:filu 用 Nerd Font glyph 當視覺語彙,是設計的一部分、不是 optional。
  - CJK Nerd Font(如 `Maple Mono NF CN`)會把 icon 畫成全形(2 格),filu 啟動時用 CPR 自動偵測 icon 實際格寬並據以排版,框線不會破。
- **平台**:macOS 或 Linux。不提供原生 Windows build(`GOOS=windows` 刻意編譯失敗);Windows 請走 WSL。
- **Go**:1.26+。`CGO_ENABLED=0` 可靜態編譯。

## 介面

四個面板 + 上下兩條 content row。版面 grid 為 `[1][2][3] / [1][2][3] / [4][4][3]`:

```
 [] ~/Documents/sideproj/filu                         ← 路徑 bar([]=folder glyph)
┌──────────┬──────────────────┬──────────────────────┐
│ [1] pin  │ [2] 檔案清單     │ [3] Preview │ Meta    │
│  Local   │  （3 個目錄分頁）│                       │
│  Pinned  │                  │   （active tab 全高） │
├──────────┴──────────────────┤                       │
│ [4] Carries │ Tasks         │                       │
└─────────────────────────────┴──────────────────────┘
 space menu   ? help   tab/1-4 panels   q quit         ← footer
```

- **[1] pin**:系統 Places(CWD / Home / 根目錄)+ 使用者釘選的目錄。純導覽 —— 選一個,`[2]` 就跳過去。
- **[2] 清單**:當前目錄的檔案,固定 3 個目錄分頁(各自獨立 CWD + cursor)。檔案操作主戰場。
- **[3] detail**:`Preview` / `Meta` 兩個 tab。Preview 依型別分五類呈現;Meta 是 `stat` 等級的詳細資訊。這是參考視角,失焦也不 dim。
- **[4] carry**:`Carries`(搬運暫存區)/ `Tasks`(複製/搬移任務,含 running / done / pending / error 狀態)兩個 tab。

## 功能

- **Preview 五分類**(讀 magic bytes 判定):
  - 目錄 → 內層 tree(eza `-T` 風格)
  - 壓縮包(zip / tar / tar.gz…)→ 列包內清單
  - 圖片 → base64 `data:` URI(SVG 例外,當 XML 語法highlight)
  - 文字 → chroma 語法highlight + 行號(catppuccin-mocha)
  - 二進位 → hex + ASCII(xxd 風)+ 行號
  - PDF → 純 Go 抽文字 + 頁數
- **顏色**:eza catppuccin-mocha 檔案型別色 + executable-bit 綠。structural 藍 / user lavender / popup 層色的三層階層。
- **Zoom**:每個有分頁的面板都能 `z` 展開佔滿,把分頁攤成等寬並排欄。
- **Carry-bucket 搬檔**:`Carry` 把檔案拿進 bucket → 切到目標目錄 → `Land`(落地)才決定複製(`p`)或搬移(`m`);可 `Pick` 只落地子集。落地跑 goroutine,進度在 Tasks tab 即時更新。
- **Popup**(全走 kbu form:line→expand 動畫 + 層色 border):`Space` 情境選單(§A.1)、`?` 全域說明(§A.2)、刪除確認、Rename / Add 輸入框。
- **Session 持久化**:分頁位置 + cursor、active tab、focus、detail tab、carry bucket、pinned、task log 都存進 `state.yaml`,下次啟動接續。

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
| `q` | 離開(存檔) |

面板 `[2]` 的字母 hotkey(皆列在 `Space` 選單裡):`C` carry、`y` yank(複製 full path 到剪貼簿)、`p` 貼上落地、`m` 搬移落地、`P` pin、`R` rename、`a` add、`D` delete、`S` sort、`.` 顯示隱藏檔、`z` zoom。

## 建置

```sh
CGO_ENABLED=0 go build -o filu ./cmd/filu   # 靜態編譯
go test ./...                               # 測試
gofmt -l . && go vet ./...                  # 格式 / 靜態檢查
```

## License

`<TBD>`

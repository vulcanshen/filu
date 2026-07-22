# filu

> 用 ZLC(Zero Learning Curve)設計原則打造的終端機檔案管理器 —— 不看文件,第一次開就能用到底。

filu 是一個 TUI 檔案管理器,定位對標 yazi / superfile,但把重心放在 **零學習曲線**:靠一套跨面板不變的基礎操作(`Tab` / `Enter` / `Esc` / `Space`)+ 一個 `?` 全域入口,不需要背 hotkey 就能走完整個 app。它是 `kbu` `u`-family 的成員,共用 kbu 的整套 ZLC 設計系統。

> ⚠️ **狀態:早期開發中。** 設計已收斂(見下與 `.forge/meta/IDEA.md`),實作尚未開始。

## 需求

- **Nerd Font**:filu 用 Nerd Font glyph 當視覺語彙,是設計的一部分、不是 optional。未安裝者的體驗不在支援範圍。
- **平台**:macOS 或 Linux。Windows 走 WSL(不提供原生 Windows build)。

## 介面

四個面板 + 上下兩條 content row:

```
 ~ › Documents › sideproj › filu                      ← 路徑 bar
┌──────────┬──────────────────┬──────────────────────┐
│ [1] pin  │ [2] 檔案清單     │ [3] Preview │ Info    │
│  Places  │  （3 個目錄分頁）│                       │
│  Pinned  │                  │                       │
├──────────┤                  │                       │
│ [4] carry│                  │                       │
│  bucket  │                  │                       │
└──────────┴──────────────────┴──────────────────────┘
 Space 動作   ? 選單   Tab 切面板                      ← footer
```

- **[1] pin**:系統捷徑(垃圾桶 / Home / 根目錄 / 卷宗)+ 使用者釘選的目錄
- **[2] 清單**:當前目錄的檔案,支援 3 個目錄分頁
- **[3] detail**:Preview / Info 兩個 tab
- **[4] carry**:檔案搬運暫存區 + 操作進度 / 歷史

## 操作

| 鍵 | 作用 |
|---|---|
| `Enter` | 進入目錄 / 開檔(交給系統) |
| `Esc` | 返回上層 / 關閉浮層 |
| `Tab`、`1`–`4` | 切換面板 |
| `h` / `l` | 切換當前面板的 tab |
| `Space` | 開當前能做什麼的選單 |
| `?` | 全域動作 |

**搬檔**:對檔案 `Carry` 把它拿進 bucket → 切到目標目錄 → 落地(land)時才決定複製還是搬移。

## 建置

（Go 骨架建立後）

```sh
go build ./cmd/...
go test ./...
```

預編譯版本:`<URL_TBD>`

## License

`<TBD>`

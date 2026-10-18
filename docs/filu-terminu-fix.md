# filu：還沒符合 tdp 的地方

對照 tdp v0.7.4 與它的 checklist（2026-10-18），還沒符合、filu 自己修不了的：都在等 terminu-go 或規範（已回報 terminu，編號是
回報的 FL）。全部修完就刪掉這個檔（tdp E7）。

| 違反 | 在哪裡 | 現況 | 該怎麼改 |
|---|---|---|---|
| tdp E3（checklist E3-1）：設定在 `$XDG_CONFIG_HOME/<app>` 或 `~/.config/<app>` | `internal/ui/config.go` 的 `filuConfigDir` | macOS 上 `~/.config/filu` 不存在、舊的 `~/Library/Application Support/filu` 存在時，繼續用舊的 | 等 FL5 定案：自動搬過去，或維持讓使用者自己搬 |
| tdp K6、M3（checklist K6-5）：key reference 列出這裡有作用的 core key | lib 的 panel key reference | 開了 `Shift-Tab`，`?` 裡沒有 | 等 FL7：lib 在 `ShiftTab` 開著時列出它 |
| tdp E5（checklist E5-10、E5-13）：疊 popup 不能當掉；畫面測試涵蓋每一種 popup 比畫面大 | lib 的 finder（`Finder.Files`）；`internal/ui/screen_test.go` 跳過的那一格 | 檔名搜尋或 Goto 開著時把終端機縮到 24×6，lib panic；測試先跳過 | 等 FL8：lib 修好後拿掉 `TestE5PopupLargerThanTheScreen` 裡的跳過 |
| tdp components/input/finder 的「預覽」→ file-picker 的預覽（內容下面的四列資訊） | `internal/ui/finders.go` 的 `hitPreview`（Search → Contents） | 預覽從找到的那一行開始，是 filu 自己畫的，下面沒有 TYPE／SIZE／MODIFIED／MODE | 等 FL16：lib 的檔案預覽能從某一行開始，改用 `Finder.Files` |

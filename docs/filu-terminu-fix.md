# filu 對齊 tdp v0.1.10 的待修清單

filu 已符合 tdp v0.1.7(見 `dev-remarks.md`「對照 tdp 時確認過的」)。v0.1.8、v0.1.9 新增 F1 六類、F7 尺寸、F8 dim,v0.1.10 改寫 K11(模式),
這份列出 filu 還不符合的地方。修完一條就從這裡刪掉;全部修完刪掉整份檔。

## 待修

6. **F8 dim**:有 popup 開著時,底下的 popup 與 base 畫面都照常亮度畫。規則是最上層以外全部 dim;
   底下 popup 的邊框畫成自己層色的 dim 版本;toast 不觸發 dim。
7. **F1 finder 打字時的 `Enter`**:finder 打字時按 `Enter` 目前是把 focus 交給結果清單(跟 `Tab` 一樣)。
   user 裁定(2026-09-28):打字階段是 input,`Enter` 就是 submit,直接選取清單目前那一筆(預設第一筆);
   打字時方向鍵在候選之間移動;`Tab` 把 focus 交給清單,清單裡用 `j/k` 選。
8. **K11 模式裡不開按鍵清單**(tdp v0.1.10,原本的待確認已定案):選取模式的 `modeList`(模式裡按 `Space` 開、
   可執行、只用方向鍵移動的 menu)整個拿掉。規則是模式裡 `Space` **不開任何 menu、不作用**;`?` 是這個模式的
   key reference(唯讀,列出模式裡能按的鍵與作用);模式自己的鍵直接按,揭露在 `?` 與下框 hint / footer;footer 照樣
   顯示 `?`,`Space` 不必列。`Esc`、`q` / `Ctrl-C`、`Tab` 照舊(離開模式 / 離開流程 / 暫停但要回應)。
   守 `modeList` 的測試改寫成「模式裡 `Space` 不作用、`?` 開模式的 key reference」,不是刪掉。

## 已確認不用改

- F7 terminal 類「terminal 寬 − 2 × 高 − 2」指內容區(user 2026-09-28):filu 的 shell popup 外框貼滿畫面,
  內容區正好是 W−2 × H−2,已符合。

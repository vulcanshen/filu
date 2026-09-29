# filu — terminu fix

filu 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.23/principle)（tdp v0.1.23）的地方，逐條待修。
修好一條就刪掉一條。

盤點日期：2026-09-29。依據 `main` 的 `5d7bdd4`（已對齊 v0.1.22，工作區乾淨）。**這一輪只對 v0.1.22 → v0.1.23 的改動**：

- D7：README 講 icon 寬度的地方（通常在前置需求的 Nerd Font 那一段），寫出 `<APP>__ICON_WIDTH` 與 `TERMINU__ICON_WIDTH` 兩個變數，
  並說明後者是全家族共用的：設一次，家族每個 app 都讀到；在家族 app 的 PTY 裡跑時，外層 app 會替它設好（D6）。寫法不限。不舉「一定
  佔兩格」的字型當例子（filu 實測：Maple Mono NF CN 的 icon 看起來兩格，游標只前進一格）。kbu 的 README 環境變數表是例子。

tdp 連結（README 兩份、docs、`.claude/rules`）已由 terminu session 從 v0.1.22 改成 v0.1.23，只改網址，跟這份清單一起留在工作樹，
還沒 commit。


## 先看

- 這一輪只改文件：清單、改釘的連結與 README 兩份可以同一個 commit；CHANGELOG `[Unreleased]` 記一條文件改動（要不要記由 filu 定）。
- 修完拿 v0.1.23 全文再對一次（這一輪的 diff 只有 D7），刪掉這份清單。不 push、不發版；把這一輪寫進 terminu
  `.local/family-fix/filu/README.md`。


## 1. README 沒寫 icon 寬度的兩個變數 —— D7（v0.1.23）

**現況**：README 兩份的前置需求（第 35 行）只寫「終端機要用 Nerd Font，CJK Nerd Font（如 Maple Mono NF CN）也沒問題」，沒提 filu 會在啟動時
探測 icon 寬度、`FILU__ICON_WIDTH`、`TERMINU__ICON_WIDTH`、`filu iconwidth`。「也沒問題」沒有說它佔兩格，這句可以留。

**規則**：D7（v0.1.23）—— README 講 icon 寬度的地方寫出 `<APP>__ICON_WIDTH` 與 `TERMINU__ICON_WIDTH`，說明後者全家族共用、設一次每個
app 都讀到、在家族 app 的 PTY 裡跑時外層會替它設好；不舉「一定佔兩格」的字型當例子。

**怎麼改**：在那一條後面補一句（兩份 README 對齊），例：filu 啟動時會問終端機一個 icon 佔幾格；框線還是歪的話，`filu iconwidth` 看
量到多少，設 `FILU__ICON_WIDTH=1` 或 `2`；`TERMINU__ICON_WIDTH` 是 terminu 家族共用的同一個設定，設一次家族每個 app 都讀到，在家族
app 的終端機裡跑時外層 app 會替 filu 設好，filu 也會替它的 shell 裡跑的程式設好。參考 kbu README 的環境變數表（`KBU__ICON_WIDTH`、
`TERMINU__ICON_WIDTH` 兩列）。


## 待確認

沒有。

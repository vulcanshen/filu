# filu — terminu fix

filu 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.22/principle)（tdp v0.1.22）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與 `docs/dev-remarks.md` 裡描述該行為的段落。

盤點日期：2026-09-29。依據 `main` 的 `02524f9`（已對齊 v0.1.21，工作區乾淨）。**這一輪只對 v0.1.21 → v0.1.22 的改動**
（`git -C ~/Documents/sideproj/terminu diff v0.1.21 v0.1.22 -- principle/`）：

- D6：`TERMINU__ICON_WIDTH` —— 有 PTY 的 app 開子程序時，在子程序的環境設 `TERMINU__ICON_WIDTH=<自己用的格數>`；每個 app 取 icon
  寬度的順序是 `<APP>__ICON_WIDTH` → `TERMINU__ICON_WIDTH` → 探測（sshu 回報：在別的 app 的 PTY 裡，探測由外層的終端模擬器回答，
  它把 icon 當一格；user 選「家族共用一個變數，所有 app 互通」）。
- D6：疊 popup 時寬、高兩邊都比畫面大也一樣切，不可以把整個框原樣交出去；測試的參考改指 filu 的 `TestD6CompositeDispOversized`。

tdp 連結（README 兩份、docs、`.claude/rules`）已由 terminu session 從 v0.1.21 改成 v0.1.22，只改網址，跟這份清單一起留在工作樹，
還沒 commit。


## 先看

- 清單與改釘的連結先一起 commit，再動程式；commit 只加自己改的路徑。
- 每修一處補 model test、做 mutation；同一個 commit 同步 README 兩份與 dev-remarks；CHANGELOG 記 `[Unreleased]`。
- **修完拿 v0.1.22 全文再逐條對一次**，修完刪掉這份清單。不 push、不發版；把這一輪寫進 terminu `.local/family-fix/filu/README.md`。

## 已經符合、不用修的（對照 v0.1.22 的改動）

- **D6 疊 popup 寬高都大也切**：`compositeDisp()` 第八輪（`b2436f3`）已經拿掉「又寬又高就原樣回傳」，`TestD6CompositeDispOversized` 就是
  v0.1.22 指的參考測試。


## 待確認

沒有。

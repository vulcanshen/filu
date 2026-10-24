# filu：還沒符合 tdp 的地方

對照 tdp v0.9.14（2026-10-24），還沒符合、filu 自己修不了的：在等 terminu-go（已回報 terminu，編號是回報的 FL）。全部修完就刪掉這個檔
（tdp E7）。

| 違反 | 在哪裡 | 現況 | 該怎麼改 |
|---|---|---|---|
| tdp layout/panel「Hint」（checklist panel-17、panel-19）：`h\|l` 會切 tab 的地方，hint 在 `Enter` 之後寫 `h\|l:tab` | `[1]`、`[3]` 的 hint（lib 組的） | 沒有 `h\|l:tab` | 等 FL27：terminu-go v0.4.17 自動加；升上去後拿掉 `filu.go` 的 `TODO(lib FL27)`、刪掉這個檔 |

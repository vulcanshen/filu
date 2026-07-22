package ui

// osRootGlyph is the OS logo shown on panel [1]'s RootDir entry. It is split by
// build tag (project rule: platform-divergent "roots" go through build tags, not
// a runtime switch). Windows would be e70f, but filu is unix-only — there is
// deliberately no _windows.go, so GOOS=windows fails to compile.
var osRootGlyph = string(rune(0xf179)) // nf-fa-apple

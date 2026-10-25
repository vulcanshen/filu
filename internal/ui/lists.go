package ui

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
	"gopkg.in/yaml.v3"
)

// userList is a list the user writes for filu: the apps of Open with, the
// directories the finders skip. They say how filu works, yet they are data
// (tdp E3): each is a file of its own next to data.yaml, named after it.
// Versions before tdp v0.9.1 kept them in config.yaml (moveLists).
type userList struct {
	key    string             // the file's name without .yaml, and its key in config.yaml before
	header string             // the comment on top of its file
	deflt  func() any         // what a new file holds
	apply  func([]byte) error // reads a file into filu's live value
}

var userLists = []userList{
	{key: "open_with", header: openWithHeader, deflt: func() any { return []openWithApp{} },
		apply: func(b []byte) error {
			var v []openWithApp
			err := yaml.Unmarshal(b, &v)
			if err == nil {
				openWithApps = v
			}
			return err
		}},
	{key: "ignore_dirs", header: ignoreDirsHeader, deflt: func() any { return defaultIgnoreDirs },
		apply: func(b []byte) error {
			var v []string
			err := yaml.Unmarshal(b, &v)
			if err == nil {
				finderIgnoreDirs = v
			}
			return err
		}},
}

const openWithHeader = `# The apps of filu's Open with ([O] on a file or a directory; [o] opens it
# with the system's default). Each has a name and a command: filu runs
# "<cmd> <path>" when you pick it. The picker offers "Default", the system's
# default app, first, then these. For example:
#   - name: VSCode
#     cmd: code
#   - name: IntelliJ IDEA
#     cmd: idea
`

const ignoreDirsHeader = `# The directories filu's finders (File names, Contents, Goto) skip:
# dependency caches, build output, IDE metadata, container data — nowhere
# you would go to. A name matches a directory of that name anywhere; a name
# with a slash (go/pkg) matches that path. An empty list skips nothing.
`

// listFilePath is where a list is kept: next to data.yaml.
func listFilePath(key string) (string, bool) {
	p, ok := dataFilePath()
	if !ok {
		return "", false
	}
	return filepath.Join(filepath.Dir(p), key+".yaml"), true
}

// writeList writes a new file for l holding its default, and reads it.
func writeList(path string, l userList) error {
	body, err := yaml.Marshal(l.deflt())
	if err != nil {
		return err
	}
	b := append([]byte(l.header), body...)
	if err := writeFileIn(path, b); err != nil {
		return err
	}
	return l.apply(b)
}

// loadLists reads the lists over what config.yaml left. A list with no file
// gets one, commented and holding the default, for the user to find and edit;
// one that cannot be read is asked about once filu runs (tdp E3: move it
// aside, or quit).
func (a *App) loadLists() {
	for _, l := range userLists {
		p, ok := listFilePath(l.key)
		if !ok {
			continue
		}
		b, err := os.ReadFile(p)
		if errors.Is(err, fs.ErrNotExist) {
			if writeList(p, l) == nil {
				a.t.Log(terminu.Info, "Wrote a commented "+l.key+".yaml with the defaults at "+shortPath(p))
			}
			continue
		}
		if err == nil {
			err = l.apply(b)
		}
		if err != nil {
			a.unread = append(a.unread, unreadFile{p, err, func() tea.Cmd {
				if err := writeList(p, l); err != nil {
					return a.t.Toast(terminu.Error, opFailedText("write a new "+l.key+".yaml", err))
				}
				a.t.Log(terminu.Info, "Wrote a new "+l.key+".yaml with the defaults at "+shortPath(p))
				return nil
			}})
		}
	}
}

// listsMove is what moveLists did.
type listsMove struct {
	from, to string   // config.yaml, and the directory the lists go to
	moved    []string // the lists moved; with err, the ones it tried
	left     []string // the lists that directory had already: left in config.yaml
	err      error    // why they could not be moved: filu goes on reading them from config.yaml
}

// moveLists moves the lists config.yaml holds to files of their own, one at a
// time (tdp E3, when the place changes): a list whose file is there already
// stays in config.yaml, for the user to put together. Each goes with the
// comments on it; what filu's template said about it goes from the comments
// left. When config.yaml cannot be written without them, the files written go,
// and filu reads the lists from config.yaml.
func moveLists() listsMove {
	cfg, ok := configFilePath()
	if !ok {
		return listsMove{}
	}
	b, err := os.ReadFile(cfg)
	var doc yaml.Node
	if err != nil || yaml.Unmarshal(b, &doc) != nil || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return listsMove{} // no config.yaml, or one loadConfig asks about
	}
	data, ok := dataFilePath()
	if !ok {
		return listsMove{}
	}
	m := listsMove{from: cfg, to: filepath.Dir(data)}
	root := doc.Content[0]
	var written []string
	for _, l := range userLists {
		i := keyIndex(root, l.key)
		if i < 0 {
			continue
		}
		to := filepath.Join(m.to, l.key+".yaml")
		if _, err := os.Lstat(to); err == nil {
			m.left = append(m.left, l.key)
			continue
		}
		m.moved = append(m.moved, l.key)
		key, val := root.Content[i], root.Content[i+1]
		if next := i + 2; i == 0 && next < len(root.Content) { // the top of config.yaml stays on top
			root.Content[next].HeadComment = strings.TrimSpace(key.HeadComment + "\n" + root.Content[next].HeadComment)
		} else if key.HeadComment != "" {
			val.HeadComment = strings.TrimSpace(key.HeadComment + "\n" + val.HeadComment)
		}
		body, err := yaml.Marshal(val)
		if err == nil {
			err = writeFileIn(to, append([]byte(l.header), body...))
		}
		if err != nil {
			m.err = err
			break
		}
		written = append(written, to)
		root.Content = slices.Delete(root.Content, i, i+2)
		for _, n := range append([]*yaml.Node{&doc, root}, root.Content...) {
			n.HeadComment = dropNotes(n.HeadComment, l.key)
		}
	}
	if m.err == nil && len(written) > 0 {
		out, err := yaml.Marshal(&doc)
		if err == nil {
			err = os.WriteFile(cfg, out, 0o644)
		}
		m.err = err
	}
	if m.err != nil {
		for _, p := range written { // back, so that filu reads one place
			_ = os.Remove(p)
		}
		return listsMove{from: m.from, to: m.to, moved: m.moved, err: m.err}
	}
	return m
}

// keyIndex is where key is among a mapping's keys, or -1.
func keyIndex(mapping *yaml.Node, key string) int {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return i
		}
	}
	return -1
}

// dropNotes takes out of a comment what filu's template said about key: the
// paragraph from "# key:" to the next "#" line.
func dropNotes(comment, key string) string {
	var out []string
	skip := false
	for _, line := range strings.Split(comment, "\n") {
		if strings.HasPrefix(line, "# "+key+":") {
			skip = true
		}
		if skip {
			skip = line != "#"
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// sayListsMove puts what moveLists did in what filu says once it runs: a
// toast for the lists moved, both paths of one it left or could not move
// (tdp E3).
func (a *App) sayListsMove(m listsMove) {
	switch {
	case m.err != nil:
		if a.moveTitle == "" {
			a.moveTitle = "Moving the lists failed"
		}
		a.moveRows = append(a.moveRows, opFailedText("move "+strings.Join(m.moved, " and ")+" from "+m.from+" to "+m.to, m.err)+".",
			"filu goes on reading them from "+m.from+". Put each in a file of its name in "+m.to+
				" and take it out of "+m.from+" yourself to switch.")
	case len(m.moved) > 0:
		a.startupNews = append(a.startupNews, "Moved "+strings.Join(m.moved, " and ")+" from "+shortPath(m.from)+" to "+shortPath(m.to))
		for _, key := range m.moved {
			a.t.Log(terminu.Info, "Moved "+key+" from "+shortPath(m.from)+" to "+shortPath(filepath.Join(m.to, key+".yaml")))
		}
	}
	if m.err == nil && len(m.left) > 0 {
		if a.moveTitle == "" {
			a.moveTitle = "Found in two places"
		}
		var both []string
		for _, key := range m.left {
			both = append(both, key+" in "+m.from+" and in "+filepath.Join(m.to, key+".yaml"))
		}
		a.moveRows = append(a.moveRows, "These were in two places: "+strings.Join(both, "; ")+".",
			"filu uses the ones in "+m.to+" and left the others in "+m.from+
				": put what you need of them into those files, then take them out of "+m.from+".")
	}
}

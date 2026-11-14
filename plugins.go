package main

import (
	"errors"
	"os"
	"sort"
	"strings"
)

const (
	settingBool   = 0
	settingEnum   = 1
	settingInt    = 2
	settingString = 3
)

var (
	pluginRoot   = "/usr/lib/cargadgetd/plugins"
	lookupPlugin = lookupPluginDefault
	lookupShell  = lookupShellDefault
)

type settingRow struct {
	label     string
	kind      int
	enums     []string
	getBool   func() int
	getEnum   func() string
	getInt    func() int
	getString func() string
	setBool   func(int) error
	setEnum   func(string) error
	setInt    func(int) error
	setString func(string) error
}

type settingSection struct {
	title string
	rows  []settingRow
}

type pluginAPI struct {
	name      string
	count     func() int
	label     func(int) string
	typ       func(int) int
	enums     func(int) string
	getBool   func(int) int
	getEnum   func(int) string
	getInt    func(int) int
	getString func(int) string
	setBool   func(int, int) int
	setEnum   func(int, string) int
	setInt    func(int, int) int
	setString func(int, string) int
	err       func() string
}

func pluginSections() []settingSection {
	var out []settingSection
	for _, id := range listIDs("libcargadgetd_app_") {
		if id == "settings" {
			continue
		}
		if sec, ok := sectionFrom(id, lookupPlugin(id)); ok {
			out = append(out, sec)
		}
	}
	for _, id := range listIDs("libcargadgetd_shell_") {
		if sec, ok := sectionFrom(id, lookupShell(id)); ok {
			out = append(out, sec)
		}
	}
	return out
}

func sectionFrom(id string, api *pluginAPI) (settingSection, bool) {
	if api == nil || api.count == nil || api.count() < 1 {
		return settingSection{}, false
	}
	title := id
	if api.name != "" {
		title = api.name
	}
	sec := settingSection{title: title}
	for i := 0; i < api.count(); i++ {
		row, ok := rowFromPlugin(api, i)
		if ok {
			sec.rows = append(sec.rows, row)
		}
	}
	if len(sec.rows) == 0 {
		return settingSection{}, false
	}
	return sec, true
}

func rowFromPlugin(api *pluginAPI, i int) (settingRow, bool) {
	if api.typ == nil || api.label == nil {
		return settingRow{}, false
	}
	row := settingRow{label: api.label(i), kind: api.typ(i)}
	switch row.kind {
	case settingBool:
		if api.getBool == nil || api.setBool == nil {
			return settingRow{}, false
		}
		row.getBool = func() int { return api.getBool(i) }
		row.setBool = func(v int) error {
			if api.setBool(i, v) == 0 {
				return nil
			}
			return pluginErr(api)
		}
	case settingEnum:
		if api.enums == nil || api.getEnum == nil || api.setEnum == nil {
			return settingRow{}, false
		}
		row.enums = splitEnums(api.enums(i))
		row.getEnum = func() string { return api.getEnum(i) }
		row.setEnum = func(v string) error {
			if api.setEnum(i, v) == 0 {
				return nil
			}
			return pluginErr(api)
		}
	case settingInt:
		if api.getInt == nil || api.setInt == nil {
			return settingRow{}, false
		}
		row.getInt = func() int { return api.getInt(i) }
		row.setInt = func(v int) error {
			if api.setInt(i, v) == 0 {
				return nil
			}
			return pluginErr(api)
		}
	case settingString:
		if api.getString == nil || api.setString == nil {
			return settingRow{}, false
		}
		row.getString = func() string { return api.getString(i) }
		row.setString = func(v string) error {
			if api.setString(i, v) == 0 {
				return nil
			}
			return pluginErr(api)
		}
	default:
		return settingRow{}, false
	}
	return row, true
}

func pluginErr(api *pluginAPI) error {
	msg := ""
	if api.err != nil {
		msg = api.err()
	}
	if msg == "" {
		msg = "rejected"
	}
	return errors.New(msg)
}

func splitEnums(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func listIDs(prefix string) []string {
	entries, err := os.ReadDir(pluginRoot)
	if err != nil {
		return nil
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		id, ok := idFromBase(e.Name(), prefix)
		if ok {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func idFromBase(name, prefix string) (string, bool) {
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".so") {
		return "", false
	}
	id := strings.TrimSuffix(name[len(prefix):], ".so")
	if !validID(id) {
		return "", false
	}
	return id, true
}

func validID(id string) bool {
	if id == "" {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if i == 0 {
			if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_') {
				return false
			}
			continue
		}
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

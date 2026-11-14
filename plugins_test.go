package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListSkipsPluginsWithoutSettings(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"libcargadgetd_app_settings.so",
		"libcargadgetd_app_hello.so",
		"libcargadgetd_app_music.so",
		"libcargadgetd_app_empty.so",
		"libcargadgetd_shell_basic.so",
		"libcargadgetd_shell_neon.so",
		"music.so",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	prevRoot, prevLook, prevShell := pluginRoot, lookupPlugin, lookupShell
	t.Cleanup(func() {
		pluginRoot, lookupPlugin, lookupShell = prevRoot, prevLook, prevShell
	})
	pluginRoot = dir
	lookupShell = func(id string) *pluginAPI {
		if id != "neon" {
			return nil
		}
		return &pluginAPI{
			name:    "Neon",
			count:   func() int { return 1 },
			label:   func(int) string { return "On-screen log" },
			typ:     func(int) int { return settingBool },
			getBool: func(int) int { return 0 },
			setBool: func(int, int) int { return 0 },
		}
	}
	lookupPlugin = func(id string) *pluginAPI {
		switch id {
		case "music":
			return &pluginAPI{
				name:    "Music",
				count:   func() int { return 1 },
				label:   func(int) string { return "Extra" },
				typ:     func(int) int { return settingBool },
				getBool: func(int) int { return 1 },
				setBool: func(int, int) int { return 0 },
			}
		case "empty":
			return &pluginAPI{name: "Empty", count: func() int { return 0 }}
		default:
			return nil
		}
	}
	secs := pluginSections()
	if len(secs) != 2 {
		t.Fatalf("sections %d", len(secs))
	}
	if secs[0].title != "Music" || len(secs[0].rows) != 1 || secs[0].rows[0].label != "Extra" {
		t.Fatalf("%+v", secs[0])
	}
	if secs[1].title != "Neon" || secs[1].rows[0].label != "On-screen log" {
		t.Fatalf("%+v", secs[1])
	}
}

func TestLookupMissingPlugin(t *testing.T) {
	if lookupPluginDefault("no_such_plugin_zz") != nil {
		t.Fatal("missing plugin resolved")
	}
	if lookupShellDefault("no_such_shell_zz") != nil {
		t.Fatal("missing shell resolved")
	}
}

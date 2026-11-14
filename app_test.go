package main

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestEmbeddedIcon(t *testing.T) {
	loadIcon()
	if iconW != 64 || iconH != 64 || len(iconPix) != 64*64*4 {
		t.Fatalf("icon %dx%d pix %d", iconW, iconH, len(iconPix))
	}
	p, w, h := iconABI()
	if p == nil || w != iconW || h != iconH {
		t.Fatalf("icon ABI %v %dx%d", p, w, h)
	}
	if uintptr(p) == uintptr(unsafe.Pointer(&iconPix[0])) {
		t.Fatal("icon must be C.malloc, not Go heap")
	}
	if !bytes.Equal(unsafe.Slice((*byte)(p), w*h*4), iconPix) {
		t.Fatal("icon pixels mismatch")
	}
}

func TestSettingsDrawEditsAndReverts(t *testing.T) {
	plug := t.TempDir()
	for _, name := range []string{"libcargadgetd_app_music.so", "libcargadgetd_app_hello.so"} {
		if err := os.WriteFile(filepath.Join(plug, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	prevRoot, prevLook, prevShell := pluginRoot, lookupPlugin, lookupShell
	t.Cleanup(func() {
		pluginRoot, lookupPlugin, lookupShell = prevRoot, prevLook, prevShell
	})
	pluginRoot = plug
	loop := 0
	lookupPlugin = func(id string) *pluginAPI {
		if id != "music" {
			return nil
		}
		return &pluginAPI{
			name:    "Music",
			count:   func() int { return 1 },
			label:   func(int) string { return "Extra" },
			typ:     func(int) int { return settingBool },
			getBool: func(int) int { return loop },
			setBool: func(_ int, v int) int {
				if v == 1 {
					return 1
				}
				loop = v
				return 0
			},
			err: func() string { return "extra must stay off" },
		}
	}

	s := newSettings()
	dst := image.NewRGBA(image.Rect(0, 0, 800, 480))
	s.draw(dst)
	if findScroll(s) == nil {
		t.Fatal("app list is not a scroll")
	}
	if findButton(s, "Hello") != nil || findCheck(s, "Audio enabled") != nil {
		t.Fatal("list shows an app with no settings, or its rows")
	}
	if findButton(s, "Music") == nil || findButton(s, "Cargadgetd") != nil {
		t.Fatal("app list")
	}
	test.Tap(findButton(s, "Music"))
	var nonempty int
	for i := 0; i < len(dst.Pix); i += 4 {
		if dst.Pix[i]|dst.Pix[i+1]|dst.Pix[i+2] != 0 {
			nonempty++
		}
	}
	if nonempty < 1000 {
		t.Fatalf("blank capture nonempty=%d", nonempty)
	}

	if findButton(s, "Back") == nil {
		t.Fatal("no back")
	}
	test.Tap(findButton(s, "Back"))
	if findButton(s, "Music") == nil {
		t.Fatal("back did not return to the list")
	}
	test.Tap(findButton(s, "Music"))
	extra := findCheck(s, "Extra")
	if extra == nil || extra.Checked {
		t.Fatal("extra check")
	}
	test.Tap(extra)
	if extra.Checked {
		t.Fatal("failed set left the check on")
	}
	if loop != 0 {
		t.Fatal("failed set stored the value")
	}
	if s.banner.Text != "extra must stay off" {
		t.Fatalf("banner %q", s.banner.Text)
	}
}

func TestEnumPickerStaysInsideAndScrolls(t *testing.T) {
	plug := t.TempDir()
	if err := os.WriteFile(filepath.Join(plug, "libcargadgetd_app_styles.so"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	prevRoot, prevLook, prevShell := pluginRoot, lookupPlugin, lookupShell
	t.Cleanup(func() {
		pluginRoot, lookupPlugin, lookupShell = prevRoot, prevLook, prevShell
	})
	pluginRoot = plug
	cur := "style-0"
	var names []string
	for i := 0; i < 40; i++ {
		names = append(names, fmt.Sprintf("style-%d", i))
	}
	joined := strings.Join(names, ",")
	lookupPlugin = func(id string) *pluginAPI {
		if id != "styles" {
			return nil
		}
		return &pluginAPI{
			name:    "Styles",
			count:   func() int { return 1 },
			label:   func(int) string { return "Look" },
			typ:     func(int) int { return settingEnum },
			enums:   func(int) string { return joined },
			getEnum: func(int) string { return cur },
			setEnum: func(_ int, v string) int {
				if v == "style-1" {
					return 1
				}
				cur = v
				return 0
			},
			err: func() string { return "style-1 is taken" },
		}
	}

	s := newSettings()
	dst := image.NewRGBA(image.Rect(0, 0, 800, 480))
	s.draw(dst)
	test.Tap(findButton(s, "Styles"))
	s.draw(dst)
	look := findButton(s, "Look: style-0")
	if look == nil {
		t.Fatal("style row")
	}
	test.Tap(look)
	s.draw(dst)
	first := findButton(s, "style-0")
	second := findButton(s, "style-1")
	if first == nil || second == nil {
		t.Fatal("style choices")
	}
	p0, ok0 := absPos(s.kit.Canvas().Content(), first, fyne.NewPos(0, 0))
	p1, ok1 := absPos(s.kit.Canvas().Content(), second, fyne.NewPos(0, 0))
	if !ok0 || !ok1 || p1.Y < p0.Y+first.Size().Height-1 || p0.X != p1.X {
		t.Fatalf("choices are not one per line: %v %v", p0, p1)
	}
	assertInside(t, s, first)
	sc := overflowingScroll(s)
	frame := s.kit.Canvas().Content().Size()
	if sc == nil || sc.Size().Width < frame.Width*0.7 || sc.Size().Height < first.Size().Height*4 {
		t.Fatalf("sheet is a single narrow row: scroll %v button %v frame %v", sc.Size(), first.Size(), frame)
	}
	if sc == nil {
		t.Fatal("long list does not scroll")
	}
	scPos, ok := absPos(s.kit.Canvas().Content(), sc, fyne.NewPos(0, 0))
	if !ok {
		t.Fatal("scroll position")
	}
	x := int(scPos.X + sc.Size().Width/2)
	y := int(scPos.Y + sc.Size().Height/2)
	s.touch(x, y, 1)
	s.touch(x, y-120, 2)
	s.touch(x, y-120, 0)
	if sc.Offset.Y < 40 {
		t.Fatalf("scroll offset %v", sc.Offset)
	}

	test.Tap(findButton(s, "style-1"))
	s.draw(dst)
	if cur != "style-0" {
		t.Fatal("rejected value was stored")
	}
	if s.banner.Text != "style-1 is taken" {
		t.Fatalf("banner %q", s.banner.Text)
	}
	if findButton(s, "Cancel") != nil || findButton(s, "Look: style-0") == nil {
		t.Fatal("failed choice left the picker open")
	}

	test.Tap(findButton(s, "Look: style-0"))
	s.draw(dst)
	card := s.picker.(*fyne.Container).Objects[2]
	cardPos, ok := absPos(s.kit.Canvas().Content(), card, fyne.NewPos(0, 0))
	if !ok || cardPos.X < 4 {
		t.Fatalf("card %v", cardPos)
	}
	s.touch(int(cardPos.X)-2, int(cardPos.Y+card.Size().Height/2), 1)
	s.touch(int(cardPos.X)-2, int(cardPos.Y+card.Size().Height/2), 0)
	if s.picker != nil {
		t.Fatal("tap outside the sheet left it open")
	}
	if cur != "style-0" {
		t.Fatal("outside tap changed the value")
	}
}

func TestEnumWriteRebuildsSection(t *testing.T) {
	plug := t.TempDir()
	if err := os.WriteFile(filepath.Join(plug, "libcargadgetd_app_grow.so"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	prevRoot, prevLook, prevShell := pluginRoot, lookupPlugin, lookupShell
	t.Cleanup(func() {
		pluginRoot, lookupPlugin, lookupShell = prevRoot, prevLook, prevShell
	})
	pluginRoot = plug
	n := 1
	lookupPlugin = func(id string) *pluginAPI {
		if id != "grow" {
			return nil
		}
		return &pluginAPI{
			name:  "Grow",
			count: func() int { return n },
			label: func(i int) string {
				if i == 0 {
					return "Slot"
				}
				return "Extra"
			},
			typ:   func(int) int { return settingEnum },
			enums: func(int) string { return "Off,On" },
			getEnum: func(i int) string {
				if i == 0 {
					return "On"
				}
				return "Off"
			},
			setEnum: func(int, string) int {
				n = 2
				return 0
			},
			err: func() string { return "" },
		}
	}

	s := newSettings()
	dst := image.NewRGBA(image.Rect(0, 0, 800, 480))
	s.draw(dst)
	if findButton(s, "Grow") == nil {
		t.Fatal("grow missing")
	}
	test.Tap(findButton(s, "Grow"))
	if findButton(s, "Extra: Off") != nil || findButton(s, "Slot: On") == nil {
		t.Fatal("section before the write")
	}
	test.Tap(findButton(s, "Slot: On"))
	off := findButton(s, "Off")
	if off == nil {
		t.Fatal("picker")
	}
	test.Tap(off)
	if n != 2 || findButton(s, "Extra: Off") == nil || findButton(s, "Cancel") != nil {
		t.Fatalf("section did not grow n=%d", n)
	}
}

func findScroll(s *settingsApp) *container.Scroll {
	var found *container.Scroll
	walk(s.kit.Canvas().Content(), func(o fyne.CanvasObject) {
		if sc, ok := o.(*container.Scroll); ok {
			found = sc
		}
	})
	return found
}

func findButton(s *settingsApp, text string) *widget.Button {
	var found *widget.Button
	walk(s.kit.Canvas().Content(), func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok && b.Text == text {
			found = b
		}
	})
	return found
}

func findCheck(s *settingsApp, text string) *widget.Check {
	var found *widget.Check
	walk(s.kit.Canvas().Content(), func(o fyne.CanvasObject) {
		if c, ok := o.(*widget.Check); ok && c.Text == text {
			found = c
		}
	})
	return found
}

func findRadio(s *settingsApp) *widget.RadioGroup {
	var found *widget.RadioGroup
	walk(s.kit.Canvas().Content(), func(o fyne.CanvasObject) {
		if r, ok := o.(*widget.RadioGroup); ok {
			found = r
		}
	})
	return found
}

func overflowingScroll(s *settingsApp) *container.Scroll {
	var found *container.Scroll
	walk(s.kit.Canvas().Content(), func(o fyne.CanvasObject) {
		sc, ok := o.(*container.Scroll)
		if !ok || sc.Content == nil {
			return
		}
		if sc.Content.MinSize().Height > sc.Size().Height+1 {
			found = sc
		}
	})
	return found
}

func assertInside(t *testing.T, s *settingsApp, o fyne.CanvasObject) {
	t.Helper()
	if o == nil {
		t.Fatal("missing object")
	}
	pos, ok := absPos(s.kit.Canvas().Content(), o, fyne.NewPos(0, 0))
	if !ok {
		t.Fatal("object not in the canvas")
	}
	bounds := s.kit.Canvas().Content().Size()
	sz := o.Size()
	if pos.X < -1 || pos.Y < -1 || pos.X+sz.Width > bounds.Width+1 || pos.Y+sz.Height > bounds.Height+1 {
		t.Fatalf("draws outside the frame at %v size %v frame %v", pos, sz, bounds)
	}
}

func absPos(o, target fyne.CanvasObject, acc fyne.Position) (fyne.Position, bool) {
	if o == nil {
		return fyne.Position{}, false
	}
	if o == target {
		return acc.Add(o.Position()), true
	}
	switch w := o.(type) {
	case *fyne.Container:
		for _, ch := range w.Objects {
			if p, ok := absPos(ch, target, acc.Add(o.Position())); ok {
				return p, true
			}
		}
	case *container.Scroll:
		if p, ok := absPos(w.Content, target, acc.Add(o.Position())); ok {
			return p, true
		}
	}
	return fyne.Position{}, false
}

func walk(o fyne.CanvasObject, fn func(fyne.CanvasObject)) {
	if o == nil {
		return
	}
	fn(o)
	switch w := o.(type) {
	case *fyne.Container:
		for _, ch := range w.Objects {
			walk(ch, fn)
		}
	case *container.Scroll:
		walk(w.Content, fn)
	}
}

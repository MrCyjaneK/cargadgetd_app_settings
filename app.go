package main

import (
	"image"
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/mrcyjanek/cargadgetd_sdk/fynekit"
)

type settingsApp struct {
	kit      *fynekit.Kit
	banner   *widget.Label
	body     *fyne.Container
	page     fyne.CanvasObject
	picker   fyne.CanvasObject
	sections []settingSection
	built    bool
	section  int
}

func newSettings() *settingsApp {
	return &settingsApp{kit: fynekit.New(), section: -1}
}

func (a *settingsApp) ensure() {
	a.kit.Ensure(a.ensureUI)
	a.built = true
}

func (a *settingsApp) ensureUI() fyne.CanvasObject {
	a.banner = widget.NewLabel("")
	a.banner.Wrapping = fyne.TextWrapWord
	a.sections = pluginSections()
	a.body = container.NewStack()
	a.showList()
	return container.NewBorder(a.banner, nil, nil, nil, a.body)
}

func (a *settingsApp) showList() {
	a.section = -1
	var rows []fyne.CanvasObject
	for i, sec := range a.sections {
		i := i
		title := sec.title
		rows = append(rows, widget.NewButton(title, func() { a.showSection(i) }))
	}
	a.fill(container.NewVScroll(container.NewPadded(container.NewVBox(rows...))))
}

func (a *settingsApp) showSection(i int) {
	if i < 0 || i >= len(a.sections) {
		a.showList()
		return
	}
	a.section = i
	sec := a.sections[i]
	objs := []fyne.CanvasObject{
		widget.NewButton("Back", a.showList),
	}
	title := widget.NewLabel(sec.title)
	title.TextStyle = fyne.TextStyle{Bold: true}
	objs = append(objs, title)
	for _, row := range sec.rows {
		objs = append(objs, a.rowWidgets(row)...)
	}
	a.fill(container.NewVScroll(container.NewPadded(container.NewVBox(objs...))))
}

func (a *settingsApp) fill(page fyne.CanvasObject) {
	a.page = page
	a.picker = nil
	a.compose()
}

func (a *settingsApp) compose() {
	if a.body == nil {
		return
	}
	var objs []fyne.CanvasObject
	if a.page != nil {
		objs = append(objs, a.page)
	}
	if a.picker != nil {
		objs = append(objs, a.picker)
	}
	a.body.Objects = objs
	if !a.built {
		return
	}
	a.body.Refresh()
	a.kit.MarkDirty()
}

func (a *settingsApp) rowWidgets(row settingRow) []fyne.CanvasObject {
	switch row.kind {
	case settingBool:
		chk := widget.NewCheck(row.label, nil)
		if row.getBool != nil {
			chk.SetChecked(row.getBool() != 0)
		}
		var guard bool
		chk.OnChanged = func(on bool) {
			if guard || row.setBool == nil {
				return
			}
			v := 0
			if on {
				v = 1
			}
			if err := row.setBool(v); err != nil {
				a.fail(err.Error())
				guard = true
				if row.getBool != nil {
					chk.SetChecked(row.getBool() != 0)
				} else {
					chk.SetChecked(!on)
				}
				guard = false
				return
			}
			a.clearErr()
		}
		return []fyne.CanvasObject{chk}
	case settingEnum:
		cur := ""
		if row.getEnum != nil {
			cur = row.getEnum()
		}
		btn := widget.NewButton(enumCaption(row.label, cur), nil)
		btn.Alignment = widget.ButtonAlignLeading
		btn.OnTapped = func() { a.openEnum(row, btn) }
		return []fyne.CanvasObject{btn}
	case settingInt:
		label := widget.NewLabel(row.label)
		cur := 0
		if row.getInt != nil {
			cur = row.getInt()
		}
		value := widget.NewLabel(strconv.Itoa(cur))
		step := func(delta int) {
			if row.getInt == nil || row.setInt == nil {
				return
			}
			if err := row.setInt(row.getInt() + delta); err != nil {
				a.fail(err.Error())
				return
			}
			value.SetText(strconv.Itoa(row.getInt()))
			a.clearErr()
		}
		minus := widget.NewButton("-", func() { step(-1) })
		plus := widget.NewButton("+", func() { step(1) })
		return []fyne.CanvasObject{label, container.NewHBox(minus, value, plus)}
	case settingString:
		label := widget.NewLabel(row.label)
		cur := ""
		if row.getString != nil {
			cur = row.getString()
		}
		entry := widget.NewEntry()
		entry.SetText(cur)
		var guard bool
		entry.OnChanged = func(text string) {
			if guard || row.setString == nil || (row.getString != nil && text == row.getString()) {
				return
			}
			if err := row.setString(text); err != nil {
				a.fail(err.Error())
				guard = true
				if row.getString != nil {
					entry.SetText(row.getString())
				}
				guard = false
				return
			}
			a.clearErr()
		}
		return []fyne.CanvasObject{label, entry}
	default:
		return nil
	}
}

func (a *settingsApp) fail(msg string) {
	if a.banner != nil {
		a.banner.SetText(msg)
	}
}

func (a *settingsApp) clearErr() {
	if a.banner != nil && a.banner.Text != "" {
		a.banner.SetText("")
	}
}

func enumCaption(label, value string) string {
	if value == "" {
		return label
	}
	return label + ": " + value
}

// openEnum covers the page with a dimmed sheet. The choices are a vertical
// scroll, one row each, so a long list stays inside the head-unit frame.
func (a *settingsApp) openEnum(row settingRow, btn *widget.Button) {
	cur := ""
	if row.getEnum != nil {
		cur = row.getEnum()
	}
	items := make([]fyne.CanvasObject, 0, len(row.enums))
	for _, opt := range row.enums {
		opt := opt
		b := widget.NewButton(opt, func() { a.chooseEnum(row, btn, opt) })
		b.Alignment = widget.ButtonAlignLeading
		if opt == cur {
			b.Importance = widget.HighImportance
		}
		items = append(items, b)
	}
	title := widget.NewLabel(row.label)
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(container.NewVBox(items...))
	panel := container.NewBorder(
		container.NewPadded(title),
		widget.NewButton("Cancel", a.closeEnum),
		nil, nil,
		scroll,
	)
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameMenuBackground))
	card := container.NewStack(bg, container.NewPadded(panel))
	dim := canvas.NewRectangle(theme.Color(theme.ColorNameShadow))
	a.picker = container.New(modalLayout{}, dim, newDismiss(a.closeEnum), card)
	a.compose()
}

func (a *settingsApp) chooseEnum(row settingRow, btn *widget.Button, sel string) {
	changed := false
	if row.setEnum != nil && (row.getEnum == nil || sel != row.getEnum()) {
		if err := row.setEnum(sel); err != nil {
			a.fail(err.Error())
			a.closeEnum()
			return
		}
		changed = true
	}
	a.clearErr()
	if changed && a.section >= 0 {
		a.sections = pluginSections()
		if a.section < len(a.sections) {
			a.showSection(a.section)
			return
		}
	}
	shown := sel
	if row.getEnum != nil {
		shown = row.getEnum()
	}
	btn.SetText(enumCaption(row.label, shown))
	a.closeEnum()
}

func (a *settingsApp) closeEnum() {
	if a.picker == nil {
		return
	}
	a.picker = nil
	a.compose()
}

func (a *settingsApp) draw(dst *image.RGBA) {
	a.ensure()
	a.kit.Draw(dst)
}

func (a *settingsApp) touch(x, y, action int) {
	a.ensure()
	a.kit.Touch(x, y, action)
}

func (a *settingsApp) pause()  { a.kit.Pause() }
func (a *settingsApp) resume() { a.kit.Resume() }

// modalLayout fills the page with the scrim and places the sheet a few
// pixels in from the frame. A Fyne vertical scroll reports a 32px minimum,
// so sizing the sheet from that minimum left one row and a drag target
// that was hard to hit.
type modalLayout struct{}

func (modalLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var s fyne.Size
	for _, o := range objs {
		s = s.Max(o.MinSize())
	}
	return s
}

func (modalLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	if len(objs) == 0 {
		return
	}
	for _, o := range objs[:len(objs)-1] {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
	card := objs[len(objs)-1]
	margin := float32(12)
	w := size.Width - margin*2
	h := size.Height - margin*2
	if w < 1 {
		w = size.Width
		margin = 0
	}
	if h < 1 {
		h = size.Height
	}
	card.Resize(fyne.NewSize(w, h))
	card.Move(fyne.NewPos((size.Width-w)/2, (size.Height-h)/2))
}

// dismissLayer takes taps and drags that miss the sheet, so the page under
// the scrim does not scroll.
type dismissLayer struct {
	widget.BaseWidget
	onTap func()
}

func newDismiss(fn func()) *dismissLayer {
	d := &dismissLayer{onTap: fn}
	d.ExtendBaseWidget(d)
	return d
}

func (d *dismissLayer) Tapped(*fyne.PointEvent) {
	if d.onTap != nil {
		d.onTap()
	}
}

func (d *dismissLayer) Dragged(*fyne.DragEvent) {}
func (d *dismissLayer) DragEnd()                {}

func (d *dismissLayer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.NRGBA{}))
}

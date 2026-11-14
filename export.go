package main

/*
#cgo CFLAGS: -I${SRCDIR}/include
#cgo LDFLAGS: -ldl
#define _GNU_SOURCE
#include <dlfcn.h>
#include <stdlib.h>
#include "cargadgetd/cg_app.h"
#include <stdint.h>
#include <string.h>

static void *cg_dlsym(const char *name) {
	if (!name)
		return 0;
	return dlsym(RTLD_DEFAULT, name);
}

static int cg_settings_count(void *fn) {
	return ((int (*)(void))fn)();
}

static const char *cg_settings_label(void *fn, int i) {
	const char *s = ((const char *(*)(int))fn)(i);
	return s ? s : "";
}

static int cg_settings_type(void *fn, int i) {
	return ((int (*)(int))fn)(i);
}

static const char *cg_settings_enums(void *fn, int i) {
	const char *s = ((const char *(*)(int))fn)(i);
	return s ? s : "";
}

static int cg_settings_get_bool(void *fn, int i) {
	return ((int (*)(int))fn)(i);
}

static const char *cg_settings_get_enum(void *fn, int i) {
	const char *s = ((const char *(*)(int))fn)(i);
	return s ? s : "";
}

static int cg_settings_get_int(void *fn, int i) {
	return ((int (*)(int))fn)(i);
}

static int cg_settings_set_bool(void *fn, int i, int v) {
	return ((int (*)(int, int))fn)(i, v);
}

static int cg_settings_set_enum(void *fn, int i, const char *v) {
	return ((int (*)(int, const char *))fn)(i, v);
}

static int cg_settings_set_int(void *fn, int i, int v) {
	return ((int (*)(int, int))fn)(i, v);
}

static const char *cg_settings_get_string(void *fn, int i) {
	const char *s = ((const char *(*)(int))fn)(i);
	return s ? s : "";
}

static int cg_settings_set_string(void *fn, int i, const char *v) {
	return ((int (*)(int, const char *))fn)(i, v);
}

static const char *cg_settings_error(void *fn) {
	const char *s = ((const char *(*)(void))fn)();
	return s ? s : "";
}

static const char *cg_settings_name(void *fn) {
	const char *s = ((const char *(*)(void))fn)();
	return s ? s : "";
}
*/
import "C"

import (
	"image"
	"unsafe"
)

var (
	app          = newSettings()
	settingsName = C.CString("Settings")
	iconC        *C.uint8_t
)

func ensureIconC() {
	if iconC != nil {
		return
	}
	loadIcon()
	if len(iconPix) == 0 {
		return
	}
	n := C.size_t(len(iconPix))
	iconC = (*C.uint8_t)(C.malloc(n))
	if iconC == nil {
		return
	}
	C.memcpy(unsafe.Pointer(iconC), unsafe.Pointer(&iconPix[0]), n)
}

//export cargadgetd_app_settings_icon
func cargadgetd_app_settings_icon(w, h *C.int) *C.uint8_t {
	ensureIconC()
	if w != nil {
		*w = C.int(iconW)
	}
	if h != nil {
		*h = C.int(iconH)
	}
	return iconC
}

func iconABI() (unsafe.Pointer, int, int) {
	var w, h C.int
	return unsafe.Pointer(cargadgetd_app_settings_icon(&w, &h)), int(w), int(h)
}

//export cargadgetd_app_settings_init
func cargadgetd_app_settings_init(*C.struct_cargadgetd_host) {}

//export cargadgetd_app_settings_draw
func cargadgetd_app_settings_draw(rgba *C.uint32_t, w, h, stride C.int) {
	if rgba == nil || w < 1 || h < 1 || stride < w*4 {
		return
	}
	pix := unsafe.Slice((*uint8)(unsafe.Pointer(rgba)), int(h)*int(stride))
	dst := &image.RGBA{
		Pix:    pix,
		Stride: int(stride),
		Rect:   image.Rect(0, 0, int(w), int(h)),
	}
	app.draw(dst)
}

//export cargadgetd_app_settings_touch
func cargadgetd_app_settings_touch(x, y, action C.int) {
	app.touch(int(x), int(y), int(action))
}

//export cargadgetd_app_settings_pause
func cargadgetd_app_settings_pause() { app.pause() }

//export cargadgetd_app_settings_resume
func cargadgetd_app_settings_resume() { app.resume() }

//export cargadgetd_app_settings_name
func cargadgetd_app_settings_name() *C.char { return settingsName }

func mustSym(name string) unsafe.Pointer {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	return unsafe.Pointer(C.cg_dlsym(cs))
}

func goCString(p *C.char) string {
	if p == nil {
		return ""
	}
	return C.GoString(p)
}

func lookupPluginDefault(id string) *pluginAPI {
	return lookupPrefixed("cargadgetd_app_"+id+"_", id)
}

func lookupShellDefault(id string) *pluginAPI {
	return lookupPrefixed("cargadgetd_shell_"+id+"_", id)
}

func lookupPrefixed(prefix, id string) *pluginAPI {
	if !validID(id) {
		return nil
	}
	count := mustSym(prefix + "settings_count")
	label := mustSym(prefix + "settings_label")
	typ := mustSym(prefix + "settings_type")
	enums := mustSym(prefix + "settings_enum_values")
	getBool := mustSym(prefix + "settings_get_bool")
	getEnum := mustSym(prefix + "settings_get_enum")
	getInt := mustSym(prefix + "settings_get_int")
	setBool := mustSym(prefix + "settings_set_bool")
	setEnum := mustSym(prefix + "settings_set_enum")
	setInt := mustSym(prefix + "settings_set_int")
	errFn := mustSym(prefix + "settings_error")
	getString := mustSym(prefix + "settings_get_string")
	setString := mustSym(prefix + "settings_set_string")
	if count == nil || label == nil || typ == nil || enums == nil ||
		getBool == nil || getEnum == nil || getInt == nil ||
		setBool == nil || setEnum == nil || setInt == nil || errFn == nil {
		return nil
	}
	name := id
	if nameFn := mustSym(prefix + "name"); nameFn != nil {
		if n := goCString(C.cg_settings_name(nameFn)); n != "" {
			name = n
		}
	}
	api := &pluginAPI{
		name:  name,
		count: func() int { return int(C.cg_settings_count(count)) },
		label: func(i int) string { return goCString(C.cg_settings_label(label, C.int(i))) },
		typ:   func(i int) int { return int(C.cg_settings_type(typ, C.int(i))) },
		enums: func(i int) string { return goCString(C.cg_settings_enums(enums, C.int(i))) },
		getBool: func(i int) int {
			return int(C.cg_settings_get_bool(getBool, C.int(i)))
		},
		getEnum: func(i int) string {
			return goCString(C.cg_settings_get_enum(getEnum, C.int(i)))
		},
		getInt: func(i int) int { return int(C.cg_settings_get_int(getInt, C.int(i))) },
		setBool: func(i, v int) int {
			return int(C.cg_settings_set_bool(setBool, C.int(i), C.int(v)))
		},
		setEnum: func(i int, v string) int {
			cs := C.CString(v)
			defer C.free(unsafe.Pointer(cs))
			return int(C.cg_settings_set_enum(setEnum, C.int(i), cs))
		},
		setInt: func(i, v int) int {
			return int(C.cg_settings_set_int(setInt, C.int(i), C.int(v)))
		},
		err: func() string { return goCString(C.cg_settings_error(errFn)) },
	}
	if getString != nil && setString != nil {
		api.getString = func(i int) string {
			return goCString(C.cg_settings_get_string(getString, C.int(i)))
		}
		api.setString = func(i int, v string) int {
			cs := C.CString(v)
			defer C.free(unsafe.Pointer(cs))
			return int(C.cg_settings_set_string(setString, C.int(i), cs))
		}
	}
	return api
}

func main() {}

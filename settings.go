package fluidsynth2

/*
#cgo pkg-config: fluidsynth
#include <fluidsynth.h>
#include <stdlib.h>

*/
import "C"
import (
	"strings"
	"unsafe"
)

var settingNames map[string]*C.char
var nSettings = 0

type Settings struct {
	ptr *C.fluid_settings_t
}

func NewSettings() Settings {
	if settingNames == nil {
		settingNames = make(map[string]*C.char)
	}
	nSettings++
	return Settings{ptr: C.new_fluid_settings()}
}

func (s *Settings) Close() {
	C.delete_fluid_settings(s.ptr)
}

func (s *Settings) SetInt(name string, val int) bool {
	return C.fluid_settings_setint(s.ptr, cname(name), C.int(val)) == 1
}

func (s *Settings) SetNum(name string, val float64) bool {
	return C.fluid_settings_setnum(s.ptr, cname(name), C.double(val)) == 1
}

func (s *Settings) SetString(name, val string) bool {
	cval := C.CString(val)
	defer C.free(unsafe.Pointer(cval))
	return C.fluid_settings_setstr(s.ptr, cname(name), cval) == 1

}

func (s *Settings) GetInt(name string, val *int) bool {
	return C.fluid_settings_getint(s.ptr, cname(name), (*C.int)(unsafe.Pointer(val))) == 1
}

func (s *Settings) GetNum(name string, val *float64) bool {
	return C.fluid_settings_getnum(s.ptr, cname(name), (*C.double)(unsafe.Pointer(val))) == 1
}

func (s *Settings) GetStringDefault(name string, val *string) bool {
	var cstr *C.char
	ok := (C.fluid_settings_getstr_default(s.ptr, cname(name), &cstr) == 1)
	if ok {
		*val = C.GoString(cstr)
	}
	return ok
}

//GetOptions returns the list of available options for a given setting.
//For example: "audio.driver" returns a coreaudio on OSX and alsa on Linux if compiled with support.
func (s *Settings) GetOptions(name string) []string {
	options := C.fluid_settings_option_concat(s.ptr, cname(name), cname(", "))
	optionsString := C.GoString(options)
	C.free(unsafe.Pointer(options))
	return strings.Split(optionsString, ", ")
}

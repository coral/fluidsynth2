package fluidsynth2

// #cgo pkg-config: fluidsynth
// #include <fluidsynth.h>
// #include <stdlib.h>
import "C"

func cbool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

func cname(name string) *C.char {
	if cname, ok := settingNames[name]; ok {
		return cname
	}
	cname := C.CString(name)
	settingNames[name] = cname
	return cname
}

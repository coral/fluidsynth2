package fluidsynth2

// #cgo pkg-config: fluidsynth
// #include <fluidsynth.h>
// #include <stdlib.h>
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

const (
	FLUID_OK     = C.FLUID_OK
	FLUID_FAILED = C.FLUID_FAILED

	MAX_MIDI_CHANNEL  = 16
	MAX_MIDI_NOTE     = 127
	MAX_MIDI_VELOCITY = 127
)

var (
	settingNames     map[string]*C.char
	settingNamesMu   sync.Mutex
	settingsRefCount int32 // Track active Settings instances
)

func fluidStatus(i C.int) error {
	if i == FLUID_FAILED {
		return fmt.Errorf("Fail")
	}

	return nil
}

func cbool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

// cleanupSettingNames frees all cached C strings when the last Settings is deleted
func cleanupSettingNames() {
	settingNamesMu.Lock()
	defer settingNamesMu.Unlock()

	for _, cstr := range settingNames {
		C.free(unsafe.Pointer(cstr))
	}
	settingNames = nil
}

func cname(name string) *C.char {
	settingNamesMu.Lock()
	defer settingNamesMu.Unlock()

	if settingNames == nil {
		settingNames = make(map[string]*C.char)
	}

	if cname, ok := settingNames[name]; ok {
		return cname
	}

	cname := C.CString(name)
	settingNames[name] = cname
	return cname
}

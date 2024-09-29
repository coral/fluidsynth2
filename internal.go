package fluidsynth2

// #cgo pkg-config: fluidsynth
// #include <fluidsynth.h>
// #include <stdlib.h>
import "C"
import "fmt"

const (
	FLUID_OK     = C.FLUID_OK
	FLUID_FAILED = C.FLUID_FAILED

	MAX_MIDI_CHANNEL  = 16
	MAX_MIDI_NOTE     = 127
	MAX_MIDI_VELOCITY = 127
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

func cname(name string) *C.char {
	if cname, ok := settingNames[name]; ok {
		return cname
	}
	cname := C.CString(name)
	settingNames[name] = cname
	return cname
}

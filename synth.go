package fluidsynth2

// #cgo pkg-config: fluidsynth
// #include <fluidsynth.h>
// #include <stdlib.h>
import "C"
import "unsafe"

type Synth struct {
	ptr *C.fluid_synth_t
}

func cbool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

func NewSynth(settings Settings) Synth {
	return Synth{C.new_fluid_synth(settings.ptr)}
}

func (s *Synth) Delete() {
	C.delete_fluid_synth(s.ptr)
}

func (s *Synth) SFLoad(path string, resetPresets bool) int {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	creset := cbool(resetPresets)
	cfont_id, _ := C.fluid_synth_sfload(s.ptr, cpath, creset)
	return int(cfont_id)
}

package fluidsynth2

// #cgo pkg-config: fluidsynth
// #include <fluidsynth.h>
// #include <stdlib.h>
import "C"
import "unsafe"

type Player struct {
	ptr *C.fluid_player_t
}

func NewPlayer(synth Synth) Player {
	return Player{C.new_fluid_player(synth.ptr)}
}

func (p *Player) Add(filename string) int {
	cpath := C.CString(filename)
	defer C.free(unsafe.Pointer(cpath))
	return int(C.fluid_player_add(p.ptr, cpath))
}

func (p *Player) AddMem(data []byte) int {
	cb := C.CBytes(data)
	defer C.free(unsafe.Pointer(cb))
	return int(C.fluid_player_add_mem(p.ptr, cb, C.size_t(cap(data))))
}

func (p *Player) Play() int {
	return int(C.fluid_player_play(p.ptr))
}

func (p *Player) Join() {
	C.fluid_player_join(p.ptr)
}

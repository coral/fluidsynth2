package fluidsynth2

// #cgo pkg-config: fluidsynth
// #include <fluidsynth.h>
// #include <stdlib.h>
import "C"
import (
	"unsafe"
)

type Player struct {
	ptr  *C.fluid_player_t
	open bool
}

func NewPlayer(synth Synth) Player {
	return Player{
		ptr:  C.new_fluid_player(synth.ptr),
		open: true,
	}
}

//Close deletes the fluid player
func (p *Player) Close() {
	if p.open {
		C.delete_fluid_player(p.ptr)
		p.open = false
	}
}

//Add plays files from disk
func (p *Player) Add(filename string) error {
	cpath := C.CString(filename)
	defer C.free(unsafe.Pointer(cpath))
	return fluidStatus(C.fluid_player_add(p.ptr, cpath))
}

//AddMem plays back MIDI data from a byte slice.
func (p *Player) AddMem(data []byte) error {
	cb := C.CBytes(data)
	defer C.free(unsafe.Pointer(cb))
	return fluidStatus(C.fluid_player_add_mem(p.ptr, cb, C.size_t(cap(data))))
}

func (p *Player) Play() error {
	return fluidStatus(C.fluid_player_play(p.ptr))
}

func (p *Player) Stop() {
	C.fluid_player_stop(p.ptr)
}

//SetLoop enables the MIDI player to loop the playlist. -1 means loop infinitely
func (p *Player) SetLoop(loops int) {
	C.fluid_player_set_loop(p.ptr, C.int(loops))
}

func (p *Player) Seek(ticks int) error {
	return fluidStatus(C.fluid_player_seek(p.ptr, C.int(ticks)))
}

//Join blocks until playback has finished
func (p *Player) Join() {
	C.fluid_player_join(p.ptr)
}

//GetBPM returns the beats per minute of the MIDI player
func (p *Player) GetBPM() int {
	return int(C.fluid_player_get_bpm(p.ptr))
}

//GetTempo returns the tempo of the MIDI player (in microseconds per quarter note)
func (p *Player) GetTempo() int {
	return int(C.fluid_player_get_midi_tempo(p.ptr))
}

type TempoType int

const (
	TEMPO_INTERNAL      = 0
	TEMPO_EXTERNAL_BPM  = 1
	TEMPO_EXTERNAL_MIDI = 2
)

//SetTempo sets the tempo of the MIDI player (in microseconds per quarter note)
func (p *Player) SetTempo(t TempoType, bpm float64) {
	if t < 0 || t > 2 {
		t = 0
	}
	C.fluid_player_set_tempo(p.ptr, C.int(t), C.double(bpm))
}

//GetCurrentTick returns the number of tempo ticks passed
func (p *Player) GetCurrentTick() int {
	return int(C.fluid_player_get_current_tick(p.ptr))
}

//GetTotalTicks returns the total tick count of the sequence
func (p *Player) GetTotalTicks() int {
	return int(C.fluid_player_get_total_ticks(p.ptr))
}

//GetStatus returns the current status of the player
func (p *Player) GetStatus() string {
	status := int(C.fluid_player_get_status(p.ptr))

	//Codes documented here http://www.fluidsynth.org/api/midi_8h.html#a5ec93766f61465dedbbac9bdb76ced83

	switch status {
	case 0:
		return "READY"
	case 1:
		return "PLAYING"
	case 2:
		return "DONE"
	default:
		return "UNKNOWN"
	}
}

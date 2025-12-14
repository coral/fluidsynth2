package fluidsynth2

// #cgo pkg-config: fluidsynth
// #include <fluidsynth.h>
// #include <stdlib.h>
import "C"
import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Player represents a MIDI file player that reads MIDI files and sends events to a Synth.
// It can load files from disk or memory and provides playback control.
type Player struct {
	ptr    *C.fluid_player_t
	synth  *Synth // Keep reference to prevent GC
	closed atomic.Bool
	mu     sync.Mutex
}

// Player status constants returned by GetStatus().
const (
	FLUID_PLAYER_READY    = "READY"    // Player is ready to play
	FLUID_PLAYER_PLAYING  = "PLAYING"  // Player is currently playing
	FLUID_PLAYER_STOPPING = "STOPPING" // Player is stopping
	FLUID_PLAYER_DONE     = "DONE"     // Player has finished playback
)

// NewPlayer creates a new MIDI file player connected to the given synthesizer.
// At least one MIDI file must be loaded via Add() or AddMem() before playback.
// The returned Player must be closed with Close() when no longer needed.
//
// Example:
//
//	player, err := fluidsynth2.NewPlayer(synth)
//	if err != nil {
//	    return err
//	}
//	defer player.Close()
//
//	player.Add("song.mid")
//	player.Play()
//	player.Join() // Wait for playback to finish
func NewPlayer(synth *Synth) (*Player, error) {
	if synth == nil {
		return nil, fmt.Errorf("synth cannot be nil")
	}
	if synth.closed.Load() {
		return nil, fmt.Errorf("synth is closed")
	}
	if synth.ptr == nil {
		return nil, fmt.Errorf("synth pointer is nil")
	}

	ptr := C.new_fluid_player(synth.ptr)
	if ptr == nil {
		return nil, fmt.Errorf("failed to create FluidSynth player")
	}

	p := &Player{
		ptr:   ptr,
		synth: synth,
	}

	runtime.SetFinalizer(p, func(p *Player) {
		p.Close()
	})

	return p, nil
}

// Close deletes the fluid player
func (p *Player) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed.Load() {
		return nil
	}

	p.closed.Store(true)

	if p.ptr != nil {
		C.delete_fluid_player(p.ptr)
		p.ptr = nil
	}

	p.synth = nil // Release reference

	runtime.SetFinalizer(p, nil)

	return nil
}

// validate checks if Player is in a valid state for method calls
func (p *Player) validate() error {
	if p.closed.Load() {
		return fmt.Errorf("player is closed")
	}
	if p.ptr == nil {
		return fmt.Errorf("player pointer is nil")
	}
	return nil
}

// Add loads a MIDI file from disk into the player's playlist.
// Multiple files can be added and will play in sequence.
//
// Parameters:
//   - filename: Path to the MIDI file (.mid or .midi)
//
// The file is loaded but playback doesn't start until Play() is called.
func (p *Player) Add(filename string) error {
	if err := p.validate(); err != nil {
		return err
	}

	cpath := C.CString(filename)
	defer C.free(unsafe.Pointer(cpath))
	if status := C.fluid_player_add(p.ptr, cpath); status == C.FLUID_FAILED {
		return fmt.Errorf("failed to add file to player: %s", filename)
	}
	return nil
}

// AddMem loads MIDI data from memory into the player's playlist.
// This is useful for playing MIDI data that's embedded in the application
// or loaded from a non-filesystem source.
//
// Parameters:
//   - data: Raw MIDI file data as a byte slice
//
// The data is loaded but playback doesn't start until Play() is called.
func (p *Player) AddMem(data []byte) error {
	if err := p.validate(); err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("empty MIDI data")
	}

	cb := C.CBytes(data)
	defer C.free(unsafe.Pointer(cb))
	return fluidStatus(C.fluid_player_add_mem(p.ptr, cb, C.size_t(len(data))))
}

// Play starts playback of the loaded MIDI file(s).
// Playback happens asynchronously. Use Join() to wait for completion.
//
// Example:
//
//	player.Add("song.mid")
//	player.Play()
//	// Do other work while playing...
//	player.Join() // Wait for playback to finish
func (p *Player) Play() error {
	if err := p.validate(); err != nil {
		return err
	}

	return fluidStatus(C.fluid_player_play(p.ptr))
}

func (p *Player) Stop() error {
	if err := p.validate(); err != nil {
		return err
	}

	result := C.fluid_player_stop(p.ptr)
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to stop player")
	}
	return nil
}

// SetLoop configures playlist looping behavior.
//
// Parameters:
//   - loops: Number of times to loop (0=no loop, -1=infinite loop, N=loop N times)
//
// Example:
//
//	player.SetLoop(-1) // Loop forever
//	player.SetLoop(3)  // Play through playlist 3 times
func (p *Player) SetLoop(loops int) error {
	if err := p.validate(); err != nil {
		return err
	}

	result := C.fluid_player_set_loop(p.ptr, C.int(loops))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set loop")
	}
	return nil
}

func (p *Player) Seek(ticks int) error {
	if err := p.validate(); err != nil {
		return err
	}

	return fluidStatus(C.fluid_player_seek(p.ptr, C.int(ticks)))
}

// Join blocks the calling goroutine until playback has finished.
// This is typically called after Play() to wait for the MIDI file to complete.
//
// Example:
//
//	player.Play()
//	player.Join() // Blocks until the song finishes
func (p *Player) Join() error {
	if err := p.validate(); err != nil {
		return err
	}

	result := C.fluid_player_join(p.ptr)
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to join player")
	}
	return nil
}

// GetBPM returns the beats per minute of the MIDI player
func (p *Player) GetBPM() (int, error) {
	if err := p.validate(); err != nil {
		return 0, err
	}

	return int(C.fluid_player_get_bpm(p.ptr)), nil
}

// GetTempo returns the tempo of the MIDI player (in microseconds per quarter note)
func (p *Player) GetTempo() (int, error) {
	if err := p.validate(); err != nil {
		return 0, err
	}

	return int(C.fluid_player_get_midi_tempo(p.ptr)), nil
}

type TempoType int

const (
	TEMPO_INTERNAL      = 0
	TEMPO_EXTERNAL_BPM  = 1
	TEMPO_EXTERNAL_MIDI = 2
)

// SetTempo sets the playback tempo using one of three tempo modes.
//
// Parameters:
//   - t: Tempo type (TEMPO_INTERNAL, TEMPO_EXTERNAL_BPM, or TEMPO_EXTERNAL_MIDI)
//   - bpm: Tempo value (interpretation depends on tempo type)
//
// Tempo types:
//   - TEMPO_INTERNAL: Use MIDI file tempo, multiplied by bpm factor (1.0=normal, 2.0=double speed)
//   - TEMPO_EXTERNAL_BPM: Override MIDI tempo with explicit BPM value
//   - TEMPO_EXTERNAL_MIDI: Override with microseconds per quarter note
//
// Example:
//
//	// Play at double speed
//	player.SetTempo(fluidsynth2.TEMPO_INTERNAL, 2.0)
//	// Force 120 BPM
//	player.SetTempo(fluidsynth2.TEMPO_EXTERNAL_BPM, 120.0)
func (p *Player) SetTempo(t TempoType, bpm float64) error {
	if err := p.validate(); err != nil {
		return err
	}
	if t < TEMPO_INTERNAL || t > TEMPO_EXTERNAL_MIDI {
		return fmt.Errorf("invalid tempo type: %d", t)
	}

	result := C.fluid_player_set_tempo(p.ptr, C.int(t), C.double(bpm))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set tempo")
	}
	return nil
}

// GetCurrentTick returns the number of tempo ticks passed
func (p *Player) GetCurrentTick() (int, error) {
	if err := p.validate(); err != nil {
		return 0, err
	}

	return int(C.fluid_player_get_current_tick(p.ptr)), nil
}

// GetTotalTicks returns the total tick count of the sequence
func (p *Player) GetTotalTicks() (int, error) {
	if err := p.validate(); err != nil {
		return 0, err
	}

	return int(C.fluid_player_get_total_ticks(p.ptr)), nil
}

// GetDivision returns the MIDI division (ticks per quarter note) of the loaded MIDI file.
// This value defines the timing resolution of the MIDI file.
// Typical values are 96, 192, 384, or 480 ticks per quarter note.
func (p *Player) GetDivision() (int, error) {
	if err := p.validate(); err != nil {
		return 0, err
	}

	return int(C.fluid_player_get_division(p.ptr)), nil
}

// GetStatus returns the current status of the player
func (p *Player) GetStatus() (string, error) {
	if err := p.validate(); err != nil {
		return "", err
	}

	status := int(C.fluid_player_get_status(p.ptr))

	//Codes documented here http://www.fluidsynth.org/api/midi_8h.html#a5ec93766f61465dedbbac9bdb76ced83

	switch status {
	case C.FLUID_PLAYER_READY:
		return FLUID_PLAYER_READY, nil
	case C.FLUID_PLAYER_PLAYING:
		return FLUID_PLAYER_PLAYING, nil
	case C.FLUID_PLAYER_STOPPING:
		return FLUID_PLAYER_STOPPING, nil
	case C.FLUID_PLAYER_DONE:
		return FLUID_PLAYER_DONE, nil
	default:
		return "UNKNOWN", fmt.Errorf("unknown status code: %d", status)
	}
}

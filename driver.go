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
)

type AudioDriver struct {
	ptr      *C.fluid_audio_driver_t
	settings *Settings // Keep reference
	synth    *Synth    // Keep reference
	closed   atomic.Bool
	mu       sync.Mutex
}

func NewAudioDriver(settings *Settings, synth *Synth) (*AudioDriver, error) {
	if settings == nil {
		return nil, fmt.Errorf("settings cannot be nil")
	}
	if synth == nil {
		return nil, fmt.Errorf("synth cannot be nil")
	}
	if settings.closed.Load() {
		return nil, fmt.Errorf("settings is closed")
	}
	if synth.closed.Load() {
		return nil, fmt.Errorf("synth is closed")
	}
	if settings.ptr == nil {
		return nil, fmt.Errorf("settings pointer is nil")
	}
	if synth.ptr == nil {
		return nil, fmt.Errorf("synth pointer is nil")
	}

	ptr := C.new_fluid_audio_driver(settings.ptr, synth.ptr)
	if ptr == nil {
		return nil, fmt.Errorf("failed to create audio driver")
	}

	d := &AudioDriver{
		ptr:      ptr,
		settings: settings,
		synth:    synth,
	}

	runtime.SetFinalizer(d, func(d *AudioDriver) {
		d.Close()
	})

	return d, nil
}

func (d *AudioDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed.Load() {
		return nil
	}

	d.closed.Store(true)

	if d.ptr != nil {
		C.delete_fluid_audio_driver(d.ptr)
		d.ptr = nil
	}

	d.settings = nil
	d.synth = nil

	runtime.SetFinalizer(d, nil)

	return nil
}

// validate checks if AudioDriver is in a valid state for method calls
func (d *AudioDriver) validate() error {
	if d.closed.Load() {
		return fmt.Errorf("AudioDriver is closed")
	}
	if d.ptr == nil {
		return fmt.Errorf("AudioDriver pointer is nil")
	}
	return nil
}

type FileRenderer struct {
	ptr    *C.fluid_file_renderer_t
	synth  *Synth
	closed atomic.Bool
	mu     sync.Mutex
}

func NewFileRenderer(synth *Synth) (*FileRenderer, error) {
	if synth == nil {
		return nil, fmt.Errorf("synth cannot be nil")
	}
	if synth.closed.Load() {
		return nil, fmt.Errorf("synth is closed")
	}
	if synth.ptr == nil {
		return nil, fmt.Errorf("synth pointer is nil")
	}

	ptr := C.new_fluid_file_renderer(synth.ptr)
	if ptr == nil {
		return nil, fmt.Errorf("failed to create file renderer")
	}

	r := &FileRenderer{
		ptr:   ptr,
		synth: synth,
	}

	runtime.SetFinalizer(r, func(r *FileRenderer) {
		r.Close()
	})

	return r, nil
}

func (r *FileRenderer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed.Load() {
		return nil
	}

	r.closed.Store(true)

	if r.ptr != nil {
		C.delete_fluid_file_renderer(r.ptr)
		r.ptr = nil
	}

	r.synth = nil

	runtime.SetFinalizer(r, nil)

	return nil
}

// validate checks if FileRenderer is in a valid state for method calls
func (r *FileRenderer) validate() error {
	if r.closed.Load() {
		return fmt.Errorf("FileRenderer is closed")
	}
	if r.ptr == nil {
		return fmt.Errorf("FileRenderer pointer is nil")
	}
	return nil
}

func (r *FileRenderer) ProcessBlock() (bool, error) {
	if err := r.validate(); err != nil {
		return false, err
	}

	return C.fluid_file_renderer_process_block(r.ptr) == C.FLUID_OK, nil
}

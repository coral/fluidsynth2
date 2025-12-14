package fluidsynth2

/*
#cgo pkg-config: fluidsynth
#include <fluidsynth.h>
#include <stdlib.h>

*/
import "C"
import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Settings represents FluidSynth configuration settings.
// Settings objects can be shared by multiple Synths and must be kept alive
// until all dependent Synths are closed.
type Settings struct {
	ptr      *C.fluid_settings_t
	closed   atomic.Bool
	refCount atomic.Int32 // Number of child Synths
	mu       sync.Mutex   // Protects Close() operations
}

// NewSettings creates a new FluidSynth settings object with default values.
// The returned Settings must be closed with Close() when no longer needed.
// A finalizer is registered as a safety net, but explicit cleanup is recommended.
//
// Example:
//
//	settings, err := fluidsynth2.NewSettings()
//	if err != nil {
//	    return err
//	}
//	defer settings.Close()
func NewSettings() (*Settings, error) {
	ptr := C.new_fluid_settings()
	if ptr == nil {
		return nil, fmt.Errorf("failed to create FluidSynth settings")
	}

	s := &Settings{
		ptr: ptr,
	}
	s.refCount.Store(0)

	// Increment global refcount for cname() cleanup
	atomic.AddInt32(&settingsRefCount, 1)

	// Set finalizer as safety net
	runtime.SetFinalizer(s, func(s *Settings) {
		s.Close()
	})

	return s, nil
}

// Close releases the Settings resources.
// Returns an error if any Synth instances still reference this Settings.
// Safe to call multiple times (subsequent calls are no-ops).
// After closing, the Settings cannot be used for any operations.
func (s *Settings) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Already closed?
	if s.closed.Load() {
		return nil
	}

	// Check if still in use
	if count := s.refCount.Load(); count > 0 {
		return fmt.Errorf("cannot close Settings: still referenced by %d Synth instance(s)", count)
	}

	// Mark as closed
	s.closed.Store(true)

	// Delete C object
	if s.ptr != nil {
		C.delete_fluid_settings(s.ptr)
		s.ptr = nil
	}

	// Clear finalizer since we've cleaned up manually
	runtime.SetFinalizer(s, nil)

	// Decrement global refcount and cleanup if last
	if atomic.AddInt32(&settingsRefCount, -1) == 0 {
		cleanupSettingNames()
	}

	return nil
}

// incRef increments the reference count (called by Synth constructor)
func (s *Settings) incRef() {
	s.refCount.Add(1)
}

// decRef decrements the reference count (called by Synth.Close())
func (s *Settings) decRef() {
	s.refCount.Add(-1)
}

// validate checks if Settings is in a valid state for method calls
func (s *Settings) validate() error {
	if s.closed.Load() {
		return fmt.Errorf("Settings is closed")
	}
	if s.ptr == nil {
		return fmt.Errorf("Settings pointer is nil")
	}
	return nil
}

// SetInt sets an integer setting value.
//
// Common settings:
//   - "synth.polyphony": Maximum number of voices (default: 256)
//   - "synth.midi-channels": Number of MIDI channels (default: 16)
//   - "audio.periods": Number of audio buffers (default: 16)
//   - "audio.period-size": Size of each audio buffer (default: 64)
//
// Returns an error if the Settings is closed or the setting doesn't exist.
func (s *Settings) SetInt(name string, val int) error {
	if err := s.validate(); err != nil {
		return err
	}

	if C.fluid_settings_setint(s.ptr, cname(name), C.int(val)) != 1 {
		return fmt.Errorf("failed to set int setting %s", name)
	}
	return nil
}

// SetNum sets a floating-point setting value.
//
// Common settings:
//   - "synth.gain": Master gain (0.0-10.0, default: 0.2)
//   - "synth.sample-rate": Sample rate in Hz (default: 44100.0)
//
// Returns an error if the Settings is closed or the setting doesn't exist.
func (s *Settings) SetNum(name string, val float64) error {
	if err := s.validate(); err != nil {
		return err
	}

	if C.fluid_settings_setnum(s.ptr, cname(name), C.double(val)) != 1 {
		return fmt.Errorf("failed to set num setting %s", name)
	}
	return nil
}

// SetString sets a string setting value.
//
// Common settings:
//   - "audio.driver": Audio driver ("alsa", "coreaudio", "jack", "pulseaudio", etc.)
//   - "audio.file.name": Output file path for file renderer
//   - "audio.file.type": Output file type ("raw", "wav", "flac", etc.)
//   - "midi.driver": MIDI driver name
//
// Use GetOptions() to discover available values for a setting.
// Returns an error if the Settings is closed or the setting doesn't exist.
func (s *Settings) SetString(name, val string) error {
	if err := s.validate(); err != nil {
		return err
	}

	cval := C.CString(val)
	defer C.free(unsafe.Pointer(cval))

	if C.fluid_settings_setstr(s.ptr, cname(name), cval) != 1 {
		return fmt.Errorf("failed to set string setting %s", name)
	}
	return nil
}

// GetInt retrieves an integer setting value.
// The value is stored in the provided pointer.
// Returns an error if the Settings is closed or the setting doesn't exist.
func (s *Settings) GetInt(name string, val *int) error {
	if err := s.validate(); err != nil {
		return err
	}

	if C.fluid_settings_getint(s.ptr, cname(name), (*C.int)(unsafe.Pointer(val))) != 1 {
		return fmt.Errorf("failed to get int setting %s", name)
	}
	return nil
}

// GetNum retrieves a floating-point setting value.
// The value is stored in the provided pointer.
// Returns an error if the Settings is closed or the setting doesn't exist.
func (s *Settings) GetNum(name string, val *float64) error {
	if err := s.validate(); err != nil {
		return err
	}

	if C.fluid_settings_getnum(s.ptr, cname(name), (*C.double)(unsafe.Pointer(val))) != 1 {
		return fmt.Errorf("failed to get num setting %s", name)
	}
	return nil
}

// GetStringDefault retrieves the default string value for a setting.
// The value is stored in the provided pointer.
// Returns an error if the Settings is closed or the setting doesn't exist.
func (s *Settings) GetStringDefault(name string, val *string) error {
	if err := s.validate(); err != nil {
		return err
	}

	var cstr *C.char
	if C.fluid_settings_getstr_default(s.ptr, cname(name), &cstr) != 1 {
		return fmt.Errorf("failed to get string default for setting %s", name)
	}
	*val = C.GoString(cstr)
	return nil
}

// GetOptions returns the list of available options for a string setting.
//
// Example usage:
//
//	drivers, err := settings.GetOptions("audio.driver")
//	// Returns: ["alsa", "jack", "pulseaudio"] on Linux
//	// Returns: ["coreaudio", "jack"] on macOS
//
// Common settings with options:
//   - "audio.driver": Available audio drivers
//   - "audio.file.type": Available file formats ("raw", "wav", "flac", etc.)
//   - "midi.driver": Available MIDI drivers
//
// Returns an error if the Settings is closed or the setting doesn't exist.
func (s *Settings) GetOptions(name string) ([]string, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}

	options := C.fluid_settings_option_concat(s.ptr, cname(name), cname(", "))
	if options == nil {
		return nil, fmt.Errorf("failed to get options for setting %s", name)
	}
	optionsString := C.GoString(options)
	C.free(unsafe.Pointer(options))
	return strings.Split(optionsString, ", "), nil
}

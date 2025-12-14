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

// Synth represents a FluidSynth software synthesizer instance.
// It processes MIDI events and generates audio output.
// A Synth requires a Settings object and at least one loaded soundfont to produce sound.
type Synth struct {
	ptr      *C.fluid_synth_t
	settings *Settings // Keep reference to Settings
	closed   atomic.Bool
	mu       sync.Mutex
}

// NewSynth creates a new synthesizer instance with the given settings.
// The Settings object must remain alive for the lifetime of the Synth.
// At least one soundfont must be loaded via SFLoad() before the synth can produce sound.
// The returned Synth must be closed with Close() when no longer needed.
//
// Example:
//
//	settings, _ := fluidsynth2.NewSettings()
//	defer settings.Close()
//
//	synth, err := fluidsynth2.NewSynth(settings)
//	if err != nil {
//	    return err
//	}
//	defer synth.Close()
//
//	synth.SFLoad("soundfont.sf2", false)
func NewSynth(settings *Settings) (*Synth, error) {
	if settings == nil {
		return nil, fmt.Errorf("settings cannot be nil")
	}
	if settings.closed.Load() {
		return nil, fmt.Errorf("settings is closed")
	}
	if settings.ptr == nil {
		return nil, fmt.Errorf("settings pointer is nil")
	}

	ptr := C.new_fluid_synth(settings.ptr)
	if ptr == nil {
		return nil, fmt.Errorf("failed to create FluidSynth synthesizer")
	}

	s := &Synth{
		ptr:      ptr,
		settings: settings,
	}

	// Increment Settings refcount
	settings.incRef()

	// Set finalizer as safety net
	runtime.SetFinalizer(s, func(s *Synth) {
		s.Close()
	})

	return s, nil
}

// Close releases the Synth resources and decrements the Settings reference count.
// Safe to call multiple times (subsequent calls are no-ops).
// After closing, the Synth cannot be used for any operations.
func (s *Synth) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.Load() {
		return nil
	}

	s.closed.Store(true)

	// Delete C object
	if s.ptr != nil {
		C.delete_fluid_synth(s.ptr)
		s.ptr = nil
	}

	// Decrement Settings refcount
	if s.settings != nil {
		s.settings.decRef()
		s.settings = nil
	}

	// Clear finalizer
	runtime.SetFinalizer(s, nil)

	return nil
}

// validate checks if Synth is in a valid state for method calls
func (s *Synth) validate() error {
	if s.closed.Load() {
		return fmt.Errorf("Synth is closed")
	}
	if s.ptr == nil {
		return fmt.Errorf("Synth pointer is nil")
	}
	return nil
}

// SFLoad loads a SoundFont file (.sf2) into the synthesizer.
// Returns the soundfont ID on success, which can be used with SFUnload() and other soundfont management functions.
//
// Parameters:
//   - path: Filesystem path to the soundfont file
//   - resetPresets: If true, reset all presets to the soundfont's defaults
//
// Example:
//
//	sfID, err := synth.SFLoad("/path/to/soundfont.sf2", false)
//	if err != nil {
//	    return err
//	}
//	// Later: synth.SFUnload(sfID, true)
func (s *Synth) SFLoad(path string, resetPresets bool) (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	creset := cbool(resetPresets)
	cfont_id := C.fluid_synth_sfload(s.ptr, cpath, creset)
	if cfont_id == C.FLUID_FAILED {
		return 0, fmt.Errorf("could not load soundfont: %s", path)
	}
	return int(cfont_id), nil
}

func (s *Synth) SFReload(sfid int) (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	cfont_id := C.fluid_synth_sfreload(s.ptr, C.int(sfid))
	if cfont_id == C.FLUID_FAILED {
		return 0, fmt.Errorf("could not reload soundfont with ID: %d", sfid)
	}
	return int(cfont_id), nil
}

func (s *Synth) SFUnload(sfid int, reset bool) error {
	if err := s.validate(); err != nil {
		return err
	}

	status := C.fluid_synth_sfunload(s.ptr, C.int(sfid), cbool(reset))
	if status == C.FLUID_FAILED {
		return fmt.Errorf("could not unload soundfont with ID: %d", sfid)
	}
	return nil
}

// NoteOn sends a MIDI note-on event to start playing a note.
//
// Parameters:
//   - channel: MIDI channel (0-15)
//   - note: MIDI note number (0-127, where 60=middle C)
//   - velocity: Note velocity (0-127, where 0 is silent and 127 is maximum)
//
// Example:
//
//	// Play middle C (note 60) at medium velocity on channel 0
//	synth.NoteOn(0, 60, 64)
func (s *Synth) NoteOn(channel, note, velocity uint8) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_noteon(s.ptr, C.int(channel), C.int(note), C.int(velocity))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to turn note on: channel=%d, note=%d, velocity=%d", channel, note, velocity)
	}
	return nil
}

// NoteOff sends a MIDI note-off event to stop playing a note.
//
// Parameters:
//   - channel: MIDI channel (0-15)
//   - note: MIDI note number (0-127) to stop
//
// The note will decay according to the soundfont's release envelope.
func (s *Synth) NoteOff(channel, note uint8) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_noteoff(s.ptr, C.int(channel), C.int(note))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to turn note off: channel=%d, note=%d", channel, note)
	}
	return nil
}

func (s *Synth) ProgramChange(channel, program uint8) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_program_change(s.ptr, C.int(channel), C.int(program))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to change program: channel=%d, program=%d", channel, program)
	}
	return nil
}

// CC sends a MIDI control change (CC) message to modify controller parameters.
//
// Parameters:
//   - channel: MIDI channel (0-15)
//   - ctrl: Controller number (0-127)
//   - val: Controller value (0-127)
//
// Common controllers:
//   - 1: Modulation wheel
//   - 7: Volume
//   - 10: Pan (0=left, 64=center, 127=right)
//   - 11: Expression
//   - 64: Sustain pedal (0-63=off, 64-127=on)
//   - 91: Reverb level
//   - 93: Chorus level
//
// Example:
//
//	// Set volume to maximum on channel 0
//	synth.CC(0, 7, 127)
//	// Enable sustain pedal on channel 0
//	synth.CC(0, 64, 127)
func (s *Synth) CC(channel uint8, ctrl, val int) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_cc(s.ptr, C.int(channel), C.int(ctrl), C.int(val))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to send CC: channel=%d, ctrl=%d, val=%d", channel, ctrl, val)
	}
	return nil
}

// GetCC retrieves the current value of a MIDI control change parameter
func (s *Synth) GetCC(channel uint8, ctrl int) (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	var val C.int
	result := C.fluid_synth_get_cc(s.ptr, C.int(channel), C.int(ctrl), &val)
	if result == C.FLUID_FAILED {
		return 0, fmt.Errorf("failed to get CC: channel=%d, ctrl=%d", channel, ctrl)
	}
	return int(val), nil
}

// PitchBend sends a pitch bend message to bend the pitch of all notes on a channel.
//
// Parameters:
//   - channel: MIDI channel (0-15)
//   - val: Pitch bend value (0-16383, where 8192 is center/no bend)
//
// The bend range in semitones can be configured with PitchWheelSens().
// Values below 8192 bend down, values above 8192 bend up.
//
// Example:
//
//	// Bend up by half the range
//	synth.PitchBend(0, 12288)
//	// Return to center (no bend)
//	synth.PitchBend(0, 8192)
func (s *Synth) PitchBend(channel uint8, val int) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_pitch_bend(s.ptr, C.int(channel), C.int(val))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to send pitch bend: channel=%d, val=%d", channel, val)
	}
	return nil
}

// GetPitchBend retrieves the current pitch bend value
func (s *Synth) GetPitchBend(channel uint8) (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	var val C.int
	result := C.fluid_synth_get_pitch_bend(s.ptr, C.int(channel), &val)
	if result == C.FLUID_FAILED {
		return 0, fmt.Errorf("failed to get pitch bend: channel=%d", channel)
	}
	return int(val), nil
}

// PitchWheelSens sets the pitch wheel sensitivity in semitones
func (s *Synth) PitchWheelSens(channel uint8, val int) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_pitch_wheel_sens(s.ptr, C.int(channel), C.int(val))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set pitch wheel sensitivity: channel=%d, val=%d", channel, val)
	}
	return nil
}

// GetPitchWheelSens retrieves the pitch wheel sensitivity
func (s *Synth) GetPitchWheelSens(channel uint8) (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	var val C.int
	result := C.fluid_synth_get_pitch_wheel_sens(s.ptr, C.int(channel), &val)
	if result == C.FLUID_FAILED {
		return 0, fmt.Errorf("failed to get pitch wheel sensitivity: channel=%d", channel)
	}
	return int(val), nil
}

// BankSelect selects a bank on a MIDI channel
func (s *Synth) BankSelect(channel uint8, bank int) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_bank_select(s.ptr, C.int(channel), C.int(bank))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to select bank: channel=%d, bank=%d", channel, bank)
	}
	return nil
}

// ChannelPressure sends a channel pressure (aftertouch) message
func (s *Synth) ChannelPressure(channel uint8, val int) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_channel_pressure(s.ptr, C.int(channel), C.int(val))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to send channel pressure: channel=%d, val=%d", channel, val)
	}
	return nil
}

// KeyPressure sends a polyphonic key pressure (aftertouch) message
func (s *Synth) KeyPressure(channel, key uint8, val int) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_key_pressure(s.ptr, C.int(channel), C.int(key), C.int(val))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to send key pressure: channel=%d, key=%d, val=%d", channel, key, val)
	}
	return nil
}

// AllNotesOff turns off all sounding notes on a channel.
// Notes will decay according to their release envelopes.
// Use AllSoundsOff() for immediate silence.
//
// Parameters:
//   - channel: MIDI channel (0-15, or -1 for all channels)
func (s *Synth) AllNotesOff(channel int) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_all_notes_off(s.ptr, C.int(channel))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to turn off all notes: channel=%d", channel)
	}
	return nil
}

// AllSoundsOff immediately silences all sounds on a channel without waiting for release.
// This stops all notes abruptly, ignoring release envelopes.
//
// Parameters:
//   - channel: MIDI channel (0-15, or -1 for all channels)
func (s *Synth) AllSoundsOff(channel int) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_all_sounds_off(s.ptr, C.int(channel))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to turn off all sounds: channel=%d", channel)
	}
	return nil
}

// SystemReset resets the synth to its initial state
func (s *Synth) SystemReset() error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_system_reset(s.ptr)
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to reset system")
	}
	return nil
}

func (s *Synth) GetGain() (float32, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	return float32(C.fluid_synth_get_gain(s.ptr)), nil
}

func (s *Synth) SetGain(g float32) error {
	if err := s.validate(); err != nil {
		return err
	}

	C.fluid_synth_set_gain(s.ptr, C.float(g))
	return nil
}

// GetPolyphony returns the maximum number of voices that can be played simultaneously
func (s *Synth) GetPolyphony() (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	return int(C.fluid_synth_get_polyphony(s.ptr)), nil
}

// SetPolyphony sets the maximum number of voices that can be played simultaneously
func (s *Synth) SetPolyphony(polyphony int) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_set_polyphony(s.ptr, C.int(polyphony))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set polyphony: %d", polyphony)
	}
	return nil
}

// GetActiveVoiceCount returns the number of voices currently playing
func (s *Synth) GetActiveVoiceCount() (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	return int(C.fluid_synth_get_active_voice_count(s.ptr)), nil
}

// GetCPULoad returns the CPU load in percent (0-100)
func (s *Synth) GetCPULoad() (float64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	return float64(C.fluid_synth_get_cpu_load(s.ptr)), nil
}

// CountMIDIChannels returns the number of MIDI channels
func (s *Synth) CountMIDIChannels() (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	return int(C.fluid_synth_count_midi_channels(s.ptr)), nil
}

// CountAudioChannels returns the number of audio channels (stereo pairs)
func (s *Synth) CountAudioChannels() (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	return int(C.fluid_synth_count_audio_channels(s.ptr)), nil
}

// CountAudioGroups returns the number of audio groups
func (s *Synth) CountAudioGroups() (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	return int(C.fluid_synth_count_audio_groups(s.ptr)), nil
}

// CountEffectsChannels returns the number of effects channels
func (s *Synth) CountEffectsChannels() (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	return int(C.fluid_synth_count_effects_channels(s.ptr)), nil
}

// CountEffectsGroups returns the number of effects groups
func (s *Synth) CountEffectsGroups() (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	return int(C.fluid_synth_count_effects_groups(s.ptr)), nil
}

// SFCount returns the number of loaded soundfonts
func (s *Synth) SFCount() (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	return int(C.fluid_synth_sfcount(s.ptr)), nil
}

// GetBankOffset returns the bank offset for a soundfont
func (s *Synth) GetBankOffset(sfontID int) (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	return int(C.fluid_synth_get_bank_offset(s.ptr, C.int(sfontID))), nil
}

// SetBankOffset sets the bank offset for a soundfont
func (s *Synth) SetBankOffset(sfontID, offset int) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_set_bank_offset(s.ptr, C.int(sfontID), C.int(offset))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set bank offset: sfont=%d, offset=%d", sfontID, offset)
	}
	return nil
}

// ReverbOn enables or disables the reverb effect for an effects group.
//
// Parameters:
//   - fxGroup: Effects group ID (typically 0 for the main effects group)
//   - on: true to enable, false to disable
//
// Use SetReverbRoomSize, SetReverbDamp, SetReverbWidth, and SetReverbLevel
// to configure reverb parameters.
func (s *Synth) ReverbOn(fxGroup int, on bool) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_reverb_on(s.ptr, C.int(fxGroup), cbool(on))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set reverb on/off: fx_group=%d, on=%v", fxGroup, on)
	}
	return nil
}

// SetReverbRoomSize sets the reverb room size parameter.
//
// Parameters:
//   - fxGroup: Effects group ID (typically 0)
//   - roomsize: Room size (0.0-1.0, typical range 0.0-1.2)
//
// Larger values create longer reverb tails.
func (s *Synth) SetReverbRoomSize(fxGroup int, roomsize float64) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_set_reverb_group_roomsize(s.ptr, C.int(fxGroup), C.double(roomsize))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set reverb roomsize: fx_group=%d", fxGroup)
	}
	return nil
}

// SetReverbDamp sets the reverb damping parameter
func (s *Synth) SetReverbDamp(fxGroup int, damping float64) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_set_reverb_group_damp(s.ptr, C.int(fxGroup), C.double(damping))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set reverb damp: fx_group=%d", fxGroup)
	}
	return nil
}

// SetReverbWidth sets the reverb width parameter
func (s *Synth) SetReverbWidth(fxGroup int, width float64) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_set_reverb_group_width(s.ptr, C.int(fxGroup), C.double(width))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set reverb width: fx_group=%d", fxGroup)
	}
	return nil
}

// SetReverbLevel sets the reverb output level
func (s *Synth) SetReverbLevel(fxGroup int, level float64) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_set_reverb_group_level(s.ptr, C.int(fxGroup), C.double(level))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set reverb level: fx_group=%d", fxGroup)
	}
	return nil
}

// GetReverbRoomSize retrieves the reverb room size parameter
func (s *Synth) GetReverbRoomSize(fxGroup int) (float64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	var roomsize C.double
	result := C.fluid_synth_get_reverb_group_roomsize(s.ptr, C.int(fxGroup), &roomsize)
	if result == C.FLUID_FAILED {
		return 0, fmt.Errorf("failed to get reverb roomsize: fx_group=%d", fxGroup)
	}
	return float64(roomsize), nil
}

// GetReverbDamp retrieves the reverb damping parameter
func (s *Synth) GetReverbDamp(fxGroup int) (float64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	var damping C.double
	result := C.fluid_synth_get_reverb_group_damp(s.ptr, C.int(fxGroup), &damping)
	if result == C.FLUID_FAILED {
		return 0, fmt.Errorf("failed to get reverb damp: fx_group=%d", fxGroup)
	}
	return float64(damping), nil
}

// GetReverbWidth retrieves the reverb width parameter
func (s *Synth) GetReverbWidth(fxGroup int) (float64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	var width C.double
	result := C.fluid_synth_get_reverb_group_width(s.ptr, C.int(fxGroup), &width)
	if result == C.FLUID_FAILED {
		return 0, fmt.Errorf("failed to get reverb width: fx_group=%d", fxGroup)
	}
	return float64(width), nil
}

// GetReverbLevel retrieves the reverb output level
func (s *Synth) GetReverbLevel(fxGroup int) (float64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	var level C.double
	result := C.fluid_synth_get_reverb_group_level(s.ptr, C.int(fxGroup), &level)
	if result == C.FLUID_FAILED {
		return 0, fmt.Errorf("failed to get reverb level: fx_group=%d", fxGroup)
	}
	return float64(level), nil
}

// ChorusOn enables or disables the chorus effect for an effects group.
//
// Parameters:
//   - fxGroup: Effects group ID (typically 0 for the main effects group)
//   - on: true to enable, false to disable
//
// Use SetChorusNr, SetChorusLevel, SetChorusSpeed, SetChorusDepth, and SetChorusType
// to configure chorus parameters.
func (s *Synth) ChorusOn(fxGroup int, on bool) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_chorus_on(s.ptr, C.int(fxGroup), cbool(on))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set chorus on/off: fx_group=%d, on=%v", fxGroup, on)
	}
	return nil
}

// SetChorusNr sets the number of chorus voices
func (s *Synth) SetChorusNr(fxGroup, nr int) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_set_chorus_group_nr(s.ptr, C.int(fxGroup), C.int(nr))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set chorus nr: fx_group=%d", fxGroup)
	}
	return nil
}

// SetChorusLevel sets the chorus output level
func (s *Synth) SetChorusLevel(fxGroup int, level float64) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_set_chorus_group_level(s.ptr, C.int(fxGroup), C.double(level))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set chorus level: fx_group=%d", fxGroup)
	}
	return nil
}

// SetChorusSpeed sets the chorus modulation speed (Hz)
func (s *Synth) SetChorusSpeed(fxGroup int, speed float64) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_set_chorus_group_speed(s.ptr, C.int(fxGroup), C.double(speed))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set chorus speed: fx_group=%d", fxGroup)
	}
	return nil
}

// SetChorusDepth sets the chorus modulation depth (ms)
func (s *Synth) SetChorusDepth(fxGroup int, depthMs float64) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_set_chorus_group_depth(s.ptr, C.int(fxGroup), C.double(depthMs))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set chorus depth: fx_group=%d", fxGroup)
	}
	return nil
}

// ChorusType represents the chorus waveform type
type ChorusType int

const (
	ChorusSine     ChorusType = 0 // Sine wave
	ChorusTriangle ChorusType = 1 // Triangle wave
)

// SetChorusType sets the chorus modulation waveform type
func (s *Synth) SetChorusType(fxGroup int, chorusType ChorusType) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_set_chorus_group_type(s.ptr, C.int(fxGroup), C.int(chorusType))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to set chorus type: fx_group=%d", fxGroup)
	}
	return nil
}

// GetChorusNr retrieves the number of chorus voices
func (s *Synth) GetChorusNr(fxGroup int) (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	var nr C.int
	result := C.fluid_synth_get_chorus_group_nr(s.ptr, C.int(fxGroup), &nr)
	if result == C.FLUID_FAILED {
		return 0, fmt.Errorf("failed to get chorus nr: fx_group=%d", fxGroup)
	}
	return int(nr), nil
}

// GetChorusLevel retrieves the chorus output level
func (s *Synth) GetChorusLevel(fxGroup int) (float64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	var level C.double
	result := C.fluid_synth_get_chorus_group_level(s.ptr, C.int(fxGroup), &level)
	if result == C.FLUID_FAILED {
		return 0, fmt.Errorf("failed to get chorus level: fx_group=%d", fxGroup)
	}
	return float64(level), nil
}

// GetChorusSpeed retrieves the chorus modulation speed (Hz)
func (s *Synth) GetChorusSpeed(fxGroup int) (float64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	var speed C.double
	result := C.fluid_synth_get_chorus_group_speed(s.ptr, C.int(fxGroup), &speed)
	if result == C.FLUID_FAILED {
		return 0, fmt.Errorf("failed to get chorus speed: fx_group=%d", fxGroup)
	}
	return float64(speed), nil
}

// GetChorusDepth retrieves the chorus modulation depth (ms)
func (s *Synth) GetChorusDepth(fxGroup int) (float64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	var depth C.double
	result := C.fluid_synth_get_chorus_group_depth(s.ptr, C.int(fxGroup), &depth)
	if result == C.FLUID_FAILED {
		return 0, fmt.Errorf("failed to get chorus depth: fx_group=%d", fxGroup)
	}
	return float64(depth), nil
}

// GetChorusType retrieves the chorus modulation waveform type
func (s *Synth) GetChorusType(fxGroup int) (ChorusType, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}

	var chorusType C.int
	result := C.fluid_synth_get_chorus_group_type(s.ptr, C.int(fxGroup), &chorusType)
	if result == C.FLUID_FAILED {
		return 0, fmt.Errorf("failed to get chorus type: fx_group=%d", fxGroup)
	}
	return ChorusType(chorusType), nil
}

/*
	WriteS16 synthesizes signed 16-bit samples. It will fill as much of the provided

slices as it can without overflowing 'left' or 'right'. For interleaved stereo, have both
'left' and 'right' share a backing array and use lstride = rstride = 2. ie:

	synth.WriteS16(samples, samples[1:], 2, 2)
*/
func (s *Synth) WriteS16(left, right []int16, lstride, rstride int) error {
	if err := s.validate(); err != nil {
		return err
	}

	// Validate slices are not empty
	if len(left) == 0 || len(right) == 0 {
		return fmt.Errorf("left and right slices must not be empty")
	}

	nframes := (len(left) + lstride - 1) / lstride
	rframes := (len(right) + rstride - 1) / rstride
	if rframes < nframes {
		nframes = rframes
	}
	if nframes == 0 {
		return fmt.Errorf("no frames to write")
	}
	C.fluid_synth_write_s16(s.ptr, C.int(nframes), unsafe.Pointer(&left[0]), 0, C.int(lstride), unsafe.Pointer(&right[0]), 0, C.int(rstride))
	return nil
}

func (s *Synth) WriteFloat(left, right []float32, lstride, rstride int) error {
	if err := s.validate(); err != nil {
		return err
	}

	// Validate slices are not empty
	if len(left) == 0 || len(right) == 0 {
		return fmt.Errorf("left and right slices must not be empty")
	}

	nframes := (len(left) + lstride - 1) / lstride
	rframes := (len(right) + rstride - 1) / rstride
	if rframes < nframes {
		nframes = rframes
	}
	if nframes == 0 {
		return fmt.Errorf("no frames to write")
	}
	C.fluid_synth_write_float(s.ptr, C.int(nframes), unsafe.Pointer(&left[0]), 0, C.int(lstride), unsafe.Pointer(&right[0]), 0, C.int(rstride))
	return nil
}

type TuningId struct {
	Bank, Program uint8
}

/* ActivateKeyTuning creates/modifies a specific tuning bank/program */
func (s *Synth) ActivateKeyTuning(id TuningId, name string, tuning [128]float64, apply bool) error {
	if err := s.validate(); err != nil {
		return err
	}

	n := C.CString(name)
	defer C.free(unsafe.Pointer(n))
	result := C.fluid_synth_activate_key_tuning(s.ptr, C.int(id.Bank), C.int(id.Program), n, (*C.double)(&tuning[0]), cbool(apply))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to activate key tuning")
	}
	return nil
}

/* ActivateTuning switches a midi channel onto the specified tuning bank/program */
func (s *Synth) ActivateTuning(channel uint8, id TuningId, apply bool) error {
	if err := s.validate(); err != nil {
		return err
	}

	result := C.fluid_synth_activate_tuning(s.ptr, C.int(channel), C.int(id.Bank), C.int(id.Program), cbool(apply))
	if result == C.FLUID_FAILED {
		return fmt.Errorf("failed to activate tuning on channel %d", channel)
	}
	return nil
}

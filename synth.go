package fluidsynth2

// #cgo pkg-config: fluidsynth
// #include <fluidsynth.h>
// #include <stdlib.h>
import "C"
import "unsafe"

type Synth struct {
	ptr *C.fluid_synth_t
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

func (s *Synth) NoteOn(channel, note, velocity uint8) {
	C.fluid_synth_noteon(s.ptr, C.int(channel), C.int(note), C.int(velocity))
}

func (s *Synth) NoteOff(channel, note uint8) {
	C.fluid_synth_noteoff(s.ptr, C.int(channel), C.int(note))
}

func (s *Synth) ProgramChange(channel, program uint8) {
	C.fluid_synth_program_change(s.ptr, C.int(channel), C.int(program))
}

/* WriteS16 synthesizes signed 16-bit samples. It will fill as much of the provided
slices as it can without overflowing 'left' or 'right'. For interleaved stereo, have both
'left' and 'right' share a backing array and use lstride = rstride = 2. ie:
    synth.WriteS16(samples, samples[1:], 2, 2)
*/
func (s *Synth) WriteS16(left, right []int16, lstride, rstride int) {
	nframes := (len(left) + lstride - 1) / lstride
	rframes := (len(right) + rstride - 1) / rstride
	if rframes < nframes {
		nframes = rframes
	}
	C.fluid_synth_write_s16(s.ptr, C.int(nframes), unsafe.Pointer(&left[0]), 0, C.int(lstride), unsafe.Pointer(&right[0]), 0, C.int(rstride))
}

func (s *Synth) WriteFloat(left, right []float32, lstride, rstride int) {
	nframes := (len(left) + lstride - 1) / lstride
	rframes := (len(right) + rstride - 1) / rstride
	if rframes < nframes {
		nframes = rframes
	}
	C.fluid_synth_write_float(s.ptr, C.int(nframes), unsafe.Pointer(&left[0]), 0, C.int(lstride), unsafe.Pointer(&right[0]), 0, C.int(rstride))
}

type TuningId struct {
	Bank, Program uint8
}

/* ActivateKeyTuning creates/modifies a specific tuning bank/program */
func (s *Synth) ActivateKeyTuning(id TuningId, name string, tuning [128]float64, apply bool) {
	n := C.CString(name)
	defer C.free(unsafe.Pointer(n))
	C.fluid_synth_activate_key_tuning(s.ptr, C.int(id.Bank), C.int(id.Program), n, (*C.double)(&tuning[0]), cbool(apply))
}

/* ActivateTuning switches a midi channel onto the specified tuning bank/program */
func (s *Synth) ActivateTuning(channel uint8, id TuningId, apply bool) {
	C.fluid_synth_activate_tuning(s.ptr, C.int(channel), C.int(id.Bank), C.int(id.Program), cbool(apply))
}

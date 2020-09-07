package main

import (
	"fmt"

	"github.com/coral/fluidsynth2"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("Starting SC-420")

	s := fluidsynth2.NewSettings()
	synth := fluidsynth2.NewSynth(s)
	i := synth.SFLoad("soundfont.sf2", false)
	fmt.Println(i)

	player := fluidsynth2.NewPlayer(synth)
	player.Add("files/be_sharp_bw_redfarn.mid")

	// Example of how to play from memory
	// dat, err := ioutil.ReadFile("midifile.mid")
	// if err != nil {
	// 	panic(err)
	// }

	// player.AddMem(dat)

	s.SetString("audio.driver", "coreaudio")

	adriver := fluidsynth2.NewAudioDriver(s, synth)
	_ = adriver

	player.Play()
	player.Join()

}

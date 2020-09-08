package main

import (
	"fmt"

	"github.com/coral/fluidsynth2"
)

func main() {

	s := fluidsynth2.NewSettings()
	fmt.Println("Avaliable audio drivers:")
	for _, driver := range s.GetOptions("audio.driver") {
		fmt.Print(driver + " ")
	}
	// Easy way to set audio backend
	//s.SetString("audio.driver", "coreaudio")

	synth := fluidsynth2.NewSynth(s)
	synth.SFLoad("files/soundfont.sf2", false)

	player := fluidsynth2.NewPlayer(synth)
	player.Add("files/song.mid")

	// Example of how to play from memory
	// dat, err := ioutil.ReadFile("midifile.mid")
	// if err != nil {
	// 	panic(err)
	// }

	// player.AddMem(dat)

	// Easy way to set audio backend
	//s.SetString("audio.driver", "coreaudio")

	fluidsynth2.NewAudioDriver(s, synth)

	player.Play()
	player.Join()

}

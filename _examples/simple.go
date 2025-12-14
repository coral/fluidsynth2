// Simple example demonstrating basic FluidSynth usage.
//
// Usage:
//   go run simple.go                                         # Use default files
//   go run simple.go -sf2 path/to/font.sf2                   # Custom soundfont
//   go run simple.go -midi path/to/song.mid                  # Custom MIDI file
//   go run simple.go -sf2 font.sf2 -midi song.mid            # Both custom
//   go run simple.go -h                                       # Show help
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/coral/fluidsynth2"
)

func main() {
	// Parse command-line arguments
	sf2Path := flag.String("sf2", "files/soundfont.sf2", "Path to the SoundFont (.sf2) file")
	midiPath := flag.String("midi", "files/song.mid", "Path to the MIDI (.mid) file")
	flag.Parse()

	// Create settings
	s, err := fluidsynth2.NewSettings()
	if err != nil {
		log.Fatalf("Failed to create settings: %v", err)
	}
	defer s.Close()

	// Get available audio drivers
	fmt.Println("Available audio drivers:")
	drivers, err := s.GetOptions("audio.driver")
	if err != nil {
		log.Fatalf("Failed to get audio drivers: %v", err)
	}
	for _, driver := range drivers {
		fmt.Print(driver + " ")
	}
	fmt.Println()

	// Easy way to set audio backend
	// if err := s.SetString("audio.driver", "coreaudio"); err != nil {
	// 	log.Fatalf("Failed to set audio driver: %v", err)
	// }

	// Create synth
	synth, err := fluidsynth2.NewSynth(s)
	if err != nil {
		log.Fatalf("Failed to create synth: %v", err)
	}
	defer synth.Close()

	// Load soundfont
	fmt.Printf("Loading soundfont: %s\n", *sf2Path)
	_, err = synth.SFLoad(*sf2Path, false)
	if err != nil {
		log.Fatalf("Failed to load soundfont: %v", err)
	}

	// Create player
	player, err := fluidsynth2.NewPlayer(synth)
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}
	defer player.Close()

	// Add MIDI file
	fmt.Printf("Loading MIDI file: %s\n", *midiPath)
	if err := player.Add(*midiPath); err != nil {
		log.Fatalf("Failed to add MIDI file: %v", err)
	}

	// Example of how to play from memory
	// dat, err := ioutil.ReadFile("midifile.mid")
	// if err != nil {
	// 	log.Fatalf("Failed to read MIDI file: %v", err)
	// }
	// if err := player.AddMem(dat); err != nil {
	// 	log.Fatalf("Failed to add MIDI from memory: %v", err)
	// }

	// Create audio driver (starts audio output)
	driver, err := fluidsynth2.NewAudioDriver(s, synth)
	if err != nil {
		log.Fatalf("Failed to create audio driver: %v", err)
	}
	defer driver.Close()

	// Start playback
	if err := player.Play(); err != nil {
		log.Fatalf("Failed to start playback: %v", err)
	}

	// Example: Increase tempo
	// if err := player.SetTempo(fluidsynth2.TEMPO_INTERNAL, 2); err != nil {
	// 	log.Fatalf("Failed to set tempo: %v", err)
	// }

	// Wait for playback to finish
	if err := player.Join(); err != nil {
		log.Fatalf("Failed to join player: %v", err)
	}
}

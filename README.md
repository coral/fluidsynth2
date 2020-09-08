# Fluidsynth2 bindings for Go

This package allows you to interface with Fluidsynth using Go.

It's mostly based on the repo by [sqweek](https://github.com/sqweek/fluidsynth) with updates for Fluidsynth2 and additions to the API that allows for playback and not only note sending.

Check in examples to get the general gist on how to play a MIDI file.

## Installation

1. Get the bindings:

```sh
$ go get -u github.com/coral/fluidsynth2
```

2. Import it in your code:

```go
import "github.com/coral/fluidsynth2"
```

## Simple Example

This example will play a MIDI file from disk.
You need a MIDI file and a Soundfont in order for audio to play.

```go
s := fluidsynth2.NewSettings()
synth := fluidsynth2.NewSynth(s)
i := synth.SFLoad("soundfont.sf2", false)

player := fluidsynth2.NewPlayer(synth)
player.Add("song.mid")

fluidsynth2.NewAudioDriver(s, synth)

player.Play()
player.Join()
```

## Configuring FluidSynth

Most of the methods needed to configure FluidSynth are exposed. Here is an example of how you can query FluidSynth for avaliable audio drivers.

```go
s := fluidsynth2.NewSettings()

audioDrivers := s.GetOptions("audio.driver")

for _, driver := range audioDrivers {
	fmt.Print(driver + " ")
}

//Perform logic here to decide what driver to use.
//In this case we are going to use coreaudio

s.SetString("audio.driver", "coreaudio")
```

## Playing MIDI from a buffer

Sometimes you want to load files through Go rather than FluidSynth, the bindings provide a simple way to play back byte slices of MIDI.

```go
s := fluidsynth2.NewSettings()

synth := fluidsynth2.NewSynth(s)
synth.SFLoad("files/soundfont.sf2", false)

dat, err := ioutil.ReadFile("midifile.mid")
if err != nil {
	panic(err)
}

player.AddMem(dat)

fluidsynth2.NewAudioDriver(s, synth)

player.Play()
player.Join()
```

![KADSBUGGEL](https://raw.githubusercontent.com/coral/fluidsynth2/master/kadsbuggel.png)

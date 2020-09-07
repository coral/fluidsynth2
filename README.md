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

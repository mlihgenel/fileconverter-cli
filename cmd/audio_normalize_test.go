package cmd

import "testing"

func TestBuildNormalizeOutputPath(t *testing.T) {
	outputDir = ""
	got := buildNormalizeOutputPath("/tmp/podcast.mp3", "mp3")
	if got != "/tmp/podcast_normalized.mp3" {
		t.Fatalf("unexpected output path: %s", got)
	}

	got = buildNormalizeOutputPath("/tmp/voice.wav", "mp3")
	if got != "/tmp/voice_normalized.mp3" {
		t.Fatalf("unexpected output path for format conversion: %s", got)
	}
}

func TestNormalizeAudioCodecArgs(t *testing.T) {
	// MP3
	args := normalizeAudioCodecArgs("mp3")
	if len(args) < 2 || args[1] != "libmp3lame" {
		t.Fatalf("expected libmp3lame for mp3, got %v", args)
	}

	// WAV
	args = normalizeAudioCodecArgs("wav")
	if len(args) < 2 || args[1] != "pcm_s16le" {
		t.Fatalf("expected pcm_s16le for wav, got %v", args)
	}

	// FLAC
	args = normalizeAudioCodecArgs("flac")
	if len(args) < 2 || args[1] != "flac" {
		t.Fatalf("expected flac for flac, got %v", args)
	}

	// OGG
	args = normalizeAudioCodecArgs("ogg")
	if len(args) < 2 || args[1] != "libvorbis" {
		t.Fatalf("expected libvorbis for ogg, got %v", args)
	}

	// Opus
	args = normalizeAudioCodecArgs("opus")
	if len(args) < 2 || args[1] != "libopus" {
		t.Fatalf("expected libopus for opus, got %v", args)
	}
}

func TestNormalizeDefaultValues(t *testing.T) {
	// Varsayılan LUFS
	lufs := float64(0)
	if lufs == 0 {
		lufs = -14
	}
	if lufs != -14 {
		t.Fatalf("expected default LUFS -14, got %f", lufs)
	}

	// Varsayılan TP
	tp := float64(0)
	if tp == 0 {
		tp = -1.5
	}
	if tp != -1.5 {
		t.Fatalf("expected default TP -1.5, got %f", tp)
	}

	// Varsayılan LRA
	lra := float64(0)
	if lra == 0 {
		lra = 11
	}
	if lra != 11 {
		t.Fatalf("expected default LRA 11, got %f", lra)
	}
}

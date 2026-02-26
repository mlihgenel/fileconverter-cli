package cmd

import "testing"

func TestBuildExtractAudioOutputPath(t *testing.T) {
	outputDir = ""
	got := buildExtractAudioOutputPath("/tmp/video.mp4", "mp3", "")
	if got != "/tmp/video.mp3" {
		t.Fatalf("unexpected output path: %s", got)
	}

	got = buildExtractAudioOutputPath("/tmp/video.mp4", "wav", "soundtrack")
	if got != "/tmp/soundtrack.wav" {
		t.Fatalf("unexpected output path with custom name: %s", got)
	}

	got = buildExtractAudioOutputPath("/tmp/video.mp4", "flac", "")
	if got != "/tmp/video.flac" {
		t.Fatalf("unexpected output path for flac: %s", got)
	}
}

func TestIsValidExtractAudioFormat(t *testing.T) {
	valid := []string{"mp3", "wav", "ogg", "flac", "aac", "m4a", "opus"}
	for _, f := range valid {
		if !isValidExtractAudioFormat(f) {
			t.Fatalf("expected %s to be valid", f)
		}
	}
	invalid := []string{"mp4", "avi", "jpg", "pdf", ""}
	for _, f := range invalid {
		if isValidExtractAudioFormat(f) {
			t.Fatalf("expected %s to be invalid", f)
		}
	}
}

func TestExtractAudioCodecArgs(t *testing.T) {
	// Copy mode
	args := extractAudioCodecArgs("mp3", 80, true)
	if len(args) != 2 || args[0] != "-c:a" || args[1] != "copy" {
		t.Fatalf("expected copy args, got %v", args)
	}

	// MP3 encoding
	args = extractAudioCodecArgs("mp3", 80, false)
	if len(args) < 2 || args[1] != "libmp3lame" {
		t.Fatalf("expected libmp3lame for mp3, got %v", args)
	}

	// WAV encoding
	args = extractAudioCodecArgs("wav", 0, false)
	if len(args) < 2 || args[1] != "pcm_s16le" {
		t.Fatalf("expected pcm_s16le for wav, got %v", args)
	}

	// FLAC encoding
	args = extractAudioCodecArgs("flac", 0, false)
	if len(args) < 2 || args[1] != "flac" {
		t.Fatalf("expected flac codec, got %v", args)
	}

	// Quality-based bitrate (low)
	args = extractAudioCodecArgs("mp3", 20, false)
	foundBitrate := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-b:a" && args[i+1] == "96k" {
			foundBitrate = true
		}
	}
	if !foundBitrate {
		t.Fatalf("expected 96k bitrate for quality=20, got %v", args)
	}

	// Quality-based bitrate (high)
	args = extractAudioCodecArgs("ogg", 100, false)
	foundBitrate = false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-b:a" && args[i+1] == "320k" {
			foundBitrate = true
		}
	}
	if !foundBitrate {
		t.Fatalf("expected 320k bitrate for quality=100, got %v", args)
	}
}

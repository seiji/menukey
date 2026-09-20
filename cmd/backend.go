package cmd

import "github.com/seiji/menukey/internal/prefs"

// backend is a variable so that tests can substitute prefs.MemoryBackend.
var backend prefs.Backend = prefs.DefaultsBackend{}

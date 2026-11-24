package main

import (
	_ "embed"
	"encoding/json"
)

//go:embed app.json
var appManifest []byte

// Read the version once from the same manifest used by the launcher.
var appVersionLabel = func() string {
	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(appManifest, &manifest); err != nil {
		panic(err)
	}
	return "v" + manifest.Version + buildVersionSuffix
}()

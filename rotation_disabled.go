//go:build norotation

package main

// Compile out rotation rendering while retaining the normal simulation.
const birdRotationEnabled = false
const buildVersionSuffix = "-norotate"

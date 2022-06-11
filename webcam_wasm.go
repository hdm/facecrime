//go:build wasm
// +build wasm

package main

import (
	"fmt"
	"github.com/hdm/facecrime/pigo/wasm/canvas"
)

func setupCamera() {
	c := canvas.NewCanvas(processFaces)
	webcam, err := c.StartWebcam()
	if err != nil {
		c.Log(fmt.Sprintf("no webcam available: %v", err))
		return

	}

	err = webcam.Render()
	if err != nil {
		c.Log(fmt.Sprintf("webcam render failed: %v", err))
		return
	}
	isCameraAvailable = true
}

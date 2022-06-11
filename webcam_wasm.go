//go:build wasm
// +build wasm

package main

import (
	"fmt"
	"github.com/hdm/facecrime/pigo/wasm/canvas"
	"github.com/hdm/facecrime/static"
	"log"
)

func setupCamera() {

	styleBytes, err := static.Files.ReadFile("style.css")
	if err != nil {
		log.Fatalf("failed to load styles: %v", err)
	}

	c := canvas.NewCanvas(processFaces, string(styleBytes))
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

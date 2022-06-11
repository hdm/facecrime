//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/hdm/facecrime/pigo/wasm/canvas"
)

func main() {
	c := canvas.NewCanvas()
	webcam, err := c.StartWebcam()
	if err != nil {
		c.Alert("Webcam not detected!")
	} else {
		err := webcam.Render()
		if err != nil {
			c.Log(fmt.Sprint(err))
		}
	}
}

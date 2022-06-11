package main

import (
	"fmt"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	pigo "github.com/hdm/facecrime/pigo/core"
	"github.com/hdm/facecrime/static"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

var camWidth = 640
var camHeight = 480

// TODO: translate cam coords from 1.777 to 1.333 ratio to use 1920x1080
var gameWidth = camWidth * 3
var gameHeight = camHeight * 3

var debug = false

var (
	mplusNormalFont font.Face
	mplusBigFont    font.Face
)

type Game struct{}

func (g *Game) Update() error {
	return nil
}

var ticks uint64
var ticksSinceFace uint64

func (g *Game) Draw(screen *ebiten.Image) {
	ticks++
	if ticks < uint64(ebiten.MaxTPS()) {
		displaySplashEE(screen)
		return
	}

	screen.Fill(color.RGBA{0, 0, 0, 0xff})

	if !isCameraAvailable {
		screen.Fill(color.RGBA{0x00, 0x00, 0x00, 0xff})
		text.Draw(screen, "Hi! Turn on your camera. Don't be shy.", mplusBigFont, mplusBigFont.Metrics().Height.Round()*3, 0, color.White)
		return
	}

	f := getFace()
	if f == nil {
		screen.Fill(color.RGBA{0x00, 0x00, 0x00, 0xff})
		text.Draw(screen, "Hi! I can't see your face. Get closer.", mplusBigFont, 0, mplusBigFont.Metrics().Height.Round()*3, color.White)
		return
	}

	ticksSinceFace++
	if ticksSinceFace < (uint64(ebiten.MaxTPS()) * 3) {
		text.Draw(screen, "There you are. Lets play a game.", mplusBigFont, 0, mplusBigFont.Metrics().Height.Round()*3, color.RGBA{0xff, 0x00, 0x00, 0xff})
	}

	faceClr := color.RGBA{255, 0, 0, 255}
	leftClr := color.RGBA{0, 0, 255, 255}
	rightClr := color.RGBA{255, 0, 255, 255}
	markClr := color.RGBA{255, 255, 255, 255}

	// drawface
	_ = faceClr

	row, col, scale := f.area[1], f.area[0], float32(f.area[2])

	//g.drawCircle(screen, row+(scale/2), col, 256, faceClr)

	if f.left != nil {
		col, row, scale = f.left.Row, f.left.Col, f.left.Scale // These ARE flipped (row vs col) intentionally
		// g.drawCircle(screen, col+(scale/2), row, 32, leftClr)
		text.Draw(screen, fmt.Sprintf("%d", f.left.Perturbs), mplusNormalFont, row+int(scale/2), col, leftClr)
	}
	if f.right != nil {
		col, row, scale = f.right.Row, f.right.Col, f.right.Scale // These ARE flipped (row vs col) intentionally
		// g.drawCircle(screen, col+(scale/2), row, 32, rightClr)
		_ = rightClr
		text.Draw(screen, fmt.Sprintf("%d", f.right.Perturbs), mplusNormalFont, row+int(scale/2), col, rightClr)
	}

	for i, m := range f.marks {
		col, row, scale = m[1], m[0], float32(m[2]) // These ARE flipped (row vs col) intentionally
		// g.drawCircle(screen, col+(scale/2), row, 4, markClr)
		msg := fmt.Sprintf("%.2d", i)
		text.Draw(screen, msg, mplusNormalFont, row+int(scale/2), col, markClr)
	}

	if debug {
		ebitenutil.DebugPrint(screen, fmt.Sprintf(
			"FACES: %v\nAREA: %v [%v/%v/%v]\nLEFT: %v\nRIGHT: %v\nMARKS: %v\nLAST: %s\nDIM:%d/%d\nX: %d-%d\nY: %d/%d\n",
			f.total, f.area, row, col, scale, f.left, f.right, f.marks, time.Since(time.Unix(0, f.ts)),
			f.width, f.height,
			f.minX, f.maxX, f.minY, f.maxY,
		))
	}
}

func (g *Game) drawCircle(screen *ebiten.Image, x, y, radius int, clr color.Color) {
	radius64 := float64(radius)
	minAngle := math.Acos(1 - 1/radius64)

	for angle := float64(0); angle <= 360; angle += minAngle {
		xDelta := radius64 * math.Cos(angle)
		yDelta := radius64 * math.Sin(angle)

		x1 := int(math.Round(float64(x) + xDelta))
		y1 := int(math.Round(float64(y) + yDelta))

		screen.Set(x1, y1, clr)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return gameWidth, gameHeight
}

func translateXFromCam(v int) int {
	return int((float64(v) / float64(camWidth)) * float64(gameWidth))
}

func translateYFromCam(v int) int {
	return int((float64(v) / float64(camHeight)) * float64(gameHeight))
}

func translateScaleFromCam(v int) int {
	return int(float64(gameWidth) / float64(camWidth) / float64(v))
}

var faceLock sync.Mutex

var lastFace *Face

func getFace() *Face {
	faceLock.Lock()
	defer faceLock.Unlock()
	return lastFace
}

func setFace(f *Face) {
	faceLock.Lock()
	defer faceLock.Unlock()
	lastFace = f
}

type Face struct {
	left   *pigo.Puploc
	right  *pigo.Puploc
	marks  [][]int
	total  int
	index  int
	area   []int
	ts     int64
	width  int
	height int
	minX   int
	maxX   int
	minY   int
	maxY   int
}

func processFaces(cnt int, idx int, area []int, left *pigo.Puploc, right *pigo.Puploc, landmarks [][]int) {
	if cnt > 0 {

		minX := area[1]
		maxX := area[1]
		minY := area[0]
		maxY := area[0]

		for _, m := range landmarks {
			if m[0] < minX {
				minX = m[0]
			}
			if m[0] > maxX {
				maxX = m[0]
			}
			if m[1] < minY {
				minY = m[1]
			}
			if m[1] > maxY {
				maxY = m[1]
			}
		}

		width := maxX - minX
		height := maxY - minY

		// Scale the coordinates based on the min width/height
		// Use up to X% of the screen
		xScale := (float64(gameWidth) * 0.50) / float64(width)
		yScale := (float64(gameHeight) * 0.50) / float64(height)

		xShift := int((float64(width) / 2) * xScale)
		yShift := int((float64(height) / 2) * yScale)

		// Scale the face box
		area[0] = int(float64(area[0]-minY) * yScale) // Y
		area[1] = int(float64(area[1]-minX) * xScale) // X

		// Pupal locations have rows/cols backwards
		if right != nil {
			right.Row = int(float64(right.Row-minY)*yScale) + yShift // Actually column
			right.Col = int(float64(right.Col-minX)*xScale) + xShift // Actually row
		}
		if left != nil {
			left.Row = int(float64(left.Row-minY)*yScale) + yShift // Actually column
			left.Col = int(float64(left.Col-minX)*xScale) + xShift // Actually row
		}

		for _, m := range landmarks {
			m[1] = int(float64(m[1]-minY)*yScale) + yShift
			m[0] = int(float64(m[0]-minX)*xScale) + xShift
		}

		f := &Face{
			total:  cnt,
			index:  idx,
			left:   left,
			right:  right,
			marks:  landmarks,
			area:   area,
			ts:     time.Now().UnixNano(),
			width:  width,
			height: height,
			minX:   minX,
			maxX:   maxX,
			minY:   minY,
			maxY:   maxY,
		}
		setFace(f)
	}
}

var splashEE *ebiten.Image
var splashUS *ebiten.Image

func loadImage(path string) (*ebiten.Image, error) {
	fd, err := static.Files.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	img, _, err := ebitenutil.NewImageFromReader(fd)
	return img, err
}

func displaySplashEE(screen *ebiten.Image) {
	sw, sh := screen.Size()
	w, h := splashEE.Size()

	op := &ebiten.DrawImageOptions{}

	// Move the images's center to the upper left corner.
	op.GeoM.Translate(float64(-w)/2, float64(-h)/2)

	// Scale it down
	op.GeoM.Scale(0.5, 0.5)

	// Scale the image by the device ratio so that the rendering result can be same
	// on various (different-DPI) environments.
	scale := ebiten.DeviceScaleFactor()
	op.GeoM.Scale(scale, scale)

	// Move the image's center to the screen's center.
	op.GeoM.Translate(float64(sw)/2, float64(sh)/2)

	op.Filter = ebiten.FilterLinear
	screen.DrawImage(splashEE, op)
}

func initImages() {

	var err error
	splashEE, err = loadImage("ebitengine_splash_1920x1080_black.png")
	if err != nil {
		log.Fatalf("failed to read splash: %v", err)
		return
	}

}

var isCameraAvailable bool

func main() {
	initImages()

	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	mplusNormalFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    22,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}

	mplusBigFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    48,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Adjust the line height.
	mplusBigFont = text.FaceWithLineHeight(mplusBigFont, 54)

	ebiten.SetWindowSize(gameWidth, gameHeight)
	ebiten.SetWindowTitle("FaceCrime")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	setupCamera()
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

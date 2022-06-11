package main

import (
	"fmt"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"path/filepath"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	pigo "github.com/hdm/facecrime/pigo/core"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

var gameWidth = 1920
var gameHeight = 1080
var camWidth = 640
var camHeight = 480

var (
	mplusNormalFont font.Face
	mplusBigFont    font.Face
)

type Game struct{}

func (g *Game) Update() error {
	return nil
}

var ticks uint64

func (g *Game) Draw(screen *ebiten.Image) {
	ticks++
	if ticks < uint64(ebiten.MaxTPS()) {
		screen.DrawImage(splashEE, &ebiten.DrawImageOptions{})
		return
	}

	screen.Fill(color.RGBA{0, 0, 0, 0xff})

	if !isCameraAvailable {
		screen.Fill(color.RGBA{0xff, 0, 0, 0xff})
		ebitenutil.DebugPrint(screen, "No camera available")
		return
	}

	f := getFace()
	if f == nil {
		screen.Fill(color.RGBA{0x0, 0xff, 0, 0xff})
		ebitenutil.DebugPrint(screen, "No face detected. Get closer")
		return
	}

	faceClr := color.RGBA{255, 0, 0, 255}
	leftClr := color.RGBA{0, 0, 255, 255}
	rightClr := color.RGBA{255, 0, 255, 255}
	markClr := color.RGBA{255, 255, 255, 255}

	// drawface
	_ = faceClr

	row, col, scale := translateXFromCam(f.area[1]), translateYFromCam(f.area[0]), translateScaleFromCam(f.area[2])
	//g.drawCircle(screen, row+(scale/2), col, 256, faceClr)

	if f.left != nil {
		row, col, scale = translateXFromCam(f.left.Row), translateYFromCam(f.left.Col), translateScaleFromCam(int(f.left.Scale))
		// g.drawCircle(screen, col+(scale/2), row, 32, leftClr)
		text.Draw(screen, "L", mplusNormalFont, col+(scale/2), row, leftClr)
	}
	if f.right != nil {
		row, col, scale = translateXFromCam(f.right.Row), translateYFromCam(f.right.Col), translateScaleFromCam(int(f.right.Scale))
		// g.drawCircle(screen, col+(scale/2), row, 32, rightClr)
		_ = rightClr
		text.Draw(screen, "R", mplusNormalFont, col+(scale/2), row, rightClr)
	}

	for i, m := range f.marks {
		row, col, scale = translateXFromCam(m[1]), translateYFromCam(m[0]), translateScaleFromCam(int(m[2]))
		// g.drawCircle(screen, col+(scale/2), row, 4, markClr)
		msg := fmt.Sprintf("%.2d", i)
		text.Draw(screen, msg, mplusNormalFont, col+(scale/2), row, markClr)
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf(
		"FACES: %v\nAREA: %v [%v/%v/%v]\nLEFT: %v\nRIGHT: %v\nMARKS: %v\nLAST: %s\n",
		f.total, f.area, row, col, scale, f.left, f.right, f.marks, time.Since(time.Unix(0, f.ts))))
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
	left  *pigo.Puploc
	right *pigo.Puploc
	marks [][]int
	total int
	index int
	area  []int
	ts    int64
}

func processFaces(cnt int, idx int, area []int, left *pigo.Puploc, right *pigo.Puploc, landmarks [][]int) {
	if cnt > 0 {
		f := &Face{
			total: cnt,
			index: idx,
			left:  left,
			right: right,
			marks: landmarks,
			area:  area,
			ts:    time.Now().UnixNano(),
		}
		setFace(f)
	}
}

var splashEE *ebiten.Image
var splashUS *ebiten.Image

func loadImage(path string) (*ebiten.Image, error) {
	img, _, err := ebitenutil.NewImageFromFile(filepath.Join("images", path))
	return img, err
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
		Size:    10,
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

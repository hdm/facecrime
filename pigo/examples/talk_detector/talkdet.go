package main

import "C"

import (
	"io/ioutil"
	"log"
	"math"
	"runtime"
	"unsafe"

	pigo "github.com/hdm/facecrime/pigo/core"
)

type point struct {
	x, y int
}

var (
	cascade          []byte
	puplocCascade    []byte
	faceClassifier   *pigo.Pigo
	puplocClassifier *pigo.PuplocCascade
	flpcs            map[string][]*pigo.FlpCascade
	imgParams        *pigo.ImageParams
	err              error
)

var (
	eyeCascades  = []string{"lp46", "lp44", "lp42", "lp38", "lp312"}
	mouthCascade = []string{"lp93", "lp84", "lp82", "lp81"}
)

func main() {}

//export FindFaces
func FindFaces(pixels []uint8) uintptr {
	pointCh := make(chan uintptr)

	results := clusterDetection(pixels, 480, 640)
	dets := make([][]int, len(results))

	for i := 0; i < len(results); i++ {
		// Hack: the fifth value in the slice represents the detection type: face, pupils, landmark points.
		// The sixt value was included only to transfer the mouth aspect ratio.
		dets[i] = append(dets[i], results[i].Row, results[i].Col, results[i].Scale, int(results[i].Q), 0, 1)
		// left eye
		puploc := &pigo.Puploc{
			Row:      results[i].Row - int(0.085*float32(results[i].Scale)),
			Col:      results[i].Col - int(0.185*float32(results[i].Scale)),
			Scale:    float32(results[i].Scale) * 0.4,
			Perturbs: 63,
		}
		leftEye := puplocClassifier.RunDetector(*puploc, *imgParams, 0.0, false)
		if leftEye.Row > 0 && leftEye.Col > 0 {
			dets[i] = append(dets[i], leftEye.Row, leftEye.Col, int(leftEye.Scale), int(results[i].Q), 1, 1)
		}

		// right eye
		puploc = &pigo.Puploc{
			Row:      results[i].Row - int(0.085*float32(results[i].Scale)),
			Col:      results[i].Col + int(0.185*float32(results[i].Scale)),
			Scale:    float32(results[i].Scale) * 0.4,
			Perturbs: 63,
		}

		rightEye := puplocClassifier.RunDetector(*puploc, *imgParams, 0.0, false)
		if rightEye.Row > 0 && rightEye.Col > 0 {
			dets[i] = append(dets[i], rightEye.Row, rightEye.Col, int(rightEye.Scale), int(results[i].Q), 1, 1)
		}

		// Traverse all the eye cascades and run the detector on each of them.
		for _, eye := range eyeCascades {
			for _, flpc := range flpcs[eye] {
				flp := flpc.GetLandmarkPoint(leftEye, rightEye, *imgParams, puploc.Perturbs, false)
				if flp.Row > 0 && flp.Col > 0 {
					dets[i] = append(dets[i], flp.Row, flp.Col, int(flp.Scale), int(results[i].Q), 2, 1)
				}

				flp = flpc.GetLandmarkPoint(leftEye, rightEye, *imgParams, puploc.Perturbs, true)
				if flp.Row > 0 && flp.Col > 0 {
					dets[i] = append(dets[i], flp.Row, flp.Col, int(flp.Scale), int(results[i].Q), 2, 1)
				}
			}
		}

		mouthPoints := []int{}
		// Traverse all the mouth cascades and run the detector on each of them.
		for _, mouth := range mouthCascade {
			for _, flpc := range flpcs[mouth] {
				flp := flpc.GetLandmarkPoint(leftEye, rightEye, *imgParams, puploc.Perturbs, false)
				if flp.Row > 0 && flp.Col > 0 {
					mouthPoints = append(mouthPoints, flp.Row, flp.Col)
					dets[i] = append(dets[i], flp.Row, flp.Col, int(flp.Scale), int(results[i].Q), 2, 1)
				}
			}
		}
		flp := flpcs["lp84"][0].GetLandmarkPoint(leftEye, rightEye, *imgParams, puploc.Perturbs, true)
		if flp.Row > 0 && flp.Col > 0 {
			mouthPoints = append(mouthPoints, flp.Row, flp.Col)
			dets[i] = append(dets[i], flp.Row, flp.Col, int(flp.Scale), int(results[i].Q), 2, 1)
		}

		// Calculate the distance ratio between the two horizontal and
		// two vertical landmark points on the mouth section.
		// If the ratio is below 1, it means that the mouth is open, otherwise it means that it's closed.
		p1 := &point{x: mouthPoints[2], y: mouthPoints[3]}
		p2 := &point{x: mouthPoints[len(mouthPoints)-2], y: mouthPoints[len(mouthPoints)-1]}
		p3 := &point{x: mouthPoints[4], y: mouthPoints[5]}
		p4 := &point{x: mouthPoints[len(mouthPoints)-4], y: mouthPoints[len(mouthPoints)-3]}

		dist1 := math.Sqrt(math.Pow(float64(p2.y-p1.y), 2) + math.Pow(float64(p2.x-p1.x), 2))
		dist2 := math.Sqrt(math.Pow(float64(p4.y-p3.y), 2) + math.Pow(float64(p4.x-p3.x), 2))

		mar := int(round((dist1 / dist2) * 0.19))
		dets[i] = append(dets[i], flp.Row, flp.Col, int(flp.Scale), int(results[i].Q), 3, mar)
	}

	coords := make([]int, 0, len(dets))

	go func() {
		// Since in Go we cannot transfer a 2d array through an array pointer
		// we have to transform it into 1d array.
		for _, v := range dets {
			coords = append(coords, v...)
		}
		// Include as a first slice element the number of detected faces.
		// We need to transfer this value in order to define the Python array buffer length.
		coords = append([]int{len(dets), 0, 0, 0, 0, 0}, coords...)

		// Convert the slice into an array pointer.
		s := *(*[]uint8)(unsafe.Pointer(&coords))
		p := uintptr(unsafe.Pointer(&s[0]))

		// Ensure `det` is not freed up by GC prematurely.
		runtime.KeepAlive(coords)

		// return the pointer address
		pointCh <- p
	}()
	return <-pointCh
}

// clusterDetection runs Pigo face detector core methods
// and returns a cluster with the detected faces coordinates.
func clusterDetection(pixels []uint8, rows, cols int) []pigo.Detection {
	imgParams = &pigo.ImageParams{
		Pixels: pixels,
		Rows:   rows,
		Cols:   cols,
		Dim:    cols,
	}
	cParams := pigo.CascadeParams{
		MinSize:     100,
		MaxSize:     600,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
		ImageParams: *imgParams,
	}

	// Ensure that the face detection classifier is loaded only once.
	if len(cascade) == 0 {
		cascade, err = ioutil.ReadFile("../../cascade/facefinder")
		if err != nil {
			log.Fatalf("Error reading the cascade file: %v", err)
		}
		p := pigo.NewPigo()

		// Unpack the binary file. This will return the number of cascade trees,
		// the tree depth, the threshold and the prediction from tree's leaf nodes.
		faceClassifier, err = p.Unpack(cascade)
		if err != nil {
			log.Fatalf("Error unpacking the cascade file: %s", err)
		}
	}

	// Ensure that we load the pupil localization cascade only once
	if len(puplocCascade) == 0 {
		puplocCascade, err := ioutil.ReadFile("../../cascade/puploc")
		if err != nil {
			log.Fatalf("Error reading the puploc cascade file: %s", err)
		}
		puplocClassifier, err = puplocClassifier.UnpackCascade(puplocCascade)
		if err != nil {
			log.Fatalf("Error unpacking the puploc cascade file: %s", err)
		}

		flpcs, err = puplocClassifier.ReadCascadeDir("../../cascade/lps")
		if err != nil {
			log.Fatalf("Error unpacking the facial landmark detection cascades: %s", err)
		}
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := faceClassifier.RunCascade(cParams, 0.0)

	// Calculate the intersection over union (IoU) of two clusters.
	dets = faceClassifier.ClusterDetections(dets, 0.0)

	return dets
}

// round returns the nearest integer, rounding ties away from zero.
func round(x float64) float64 {
	t := math.Trunc(x)
	if math.Abs(x-t) >= 0.5 {
		return t + math.Copysign(1, x)
	}
	return t
}

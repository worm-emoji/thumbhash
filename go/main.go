package thumbhash

import (
	"fmt"
	"math"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// func abs(a float32) float32 {
// 	if a < 0 {
// 		return -a
// 	}
// 	return a
// }

func round(a float32) float32 {
	return float32(math.Round(float64(a)))
}

// func roundInt(a int) int {
// 	return int(math.Round(float64(a)))
// }

func RGBAToThumbhash(width int, height int, rgba []uint8) ([]byte, error) {
	if width > 100 || height > 100 {
		return nil, fmt.Errorf("width and height must be less than 100")
	}

	if len(rgba) != width*height*4 {
		return nil, fmt.Errorf("rgba must be width*height*4 bytes")
	}

	// Determine the average color
	var avgR, avgG, avgB, avgA float32

	for i := 0; i < len(rgba); i += 4 {
		alpha := float32(rgba[i+3]) / 255.0
		avgR += alpha / 255.0 * float32(rgba[i])
		avgG += alpha / 255.0 * float32(rgba[i+1])
		avgB += alpha / 255.0 * float32(rgba[i+2])
		avgA += alpha
	}

	if avgA > 0.0 {
		avgR /= avgA
		avgG /= avgA
		avgB /= avgA
	}

	hasAlpha := avgA < float32(width*height)
	lLimit := 7
	if hasAlpha {
		lLimit = 5
	}

	var (
		lx = max(int(round(float32(float32(lLimit*width)/maxF(float32(width), float32(height))))), 1)
		ly = max(int(round(float32(float32(lLimit*height)/maxF(float32(width), float32(height))))), 1)
		l  = make([]float32, 0, width*height)
		p  = make([]float32, 0, width*height)
		q  = make([]float32, 0, width*height)
		a  = make([]float32, 0, width*height)
	)

	// Convert the image from RGBA to LPQA (composite atop the average color)
	for i := 0; i < len(rgba); i += 4 {
		alpha := float32(rgba[i+3]) / 255.0
		r := avgR*(1.0-alpha) + alpha/255.0*float32(rgba[i])
		g := avgG*(1.0-alpha) + alpha/255.0*float32(rgba[i+1])
		b := avgB*(1.0-alpha) + alpha/255.0*float32(rgba[i+2])
		l = append(l, (r+g+b)/3.0)
		p = append(p, (r+g)/2.0-b)
		q = append(q, r-g)
		a = append(a, alpha)
	}

	encodeChannel := func(channel []float32, nx, ny int) (float32, []float32, float32) {
		dc := float32(0.0)
		ac := make([]float32, 0, nx*ny/2)
		scale := float32(0.0)
		fx := make([]float32, width)
		for cy := 0; cy < ny; cy++ {
			cx := 0
			for cx*ny < nx*(ny-cy) {
				f := float32(0.0)
				for x := 0; x < width; x++ {
					fx[x] = float32(math.Cos(math.Pi / float64(width) * float64(cx) * (float64(x) + 0.5)))
				}
				for y := 0; y < height; y++ {
					fy := float32(math.Cos(math.Pi / float64(height) * float64(cy) * (float64(y) + 0.5)))
					for x := 0; x < width; x++ {
						f += channel[x+y*width] * fx[x] * fy
					}
				}
				f /= float32(width * height)
				if cx > 0 || cy > 0 {
					ac = append(ac, f)
					scale = maxF(scale, float32(math.Abs(float64(f))))
				} else {
					dc = f
				}
				cx++
			}
		}
		if scale > 0.0 {
			for i := range ac {
				ac[i] = 0.5 + 0.5/scale*ac[i]
			}
		}
		return dc, ac, scale
	}

	lDC, lAC, lScale := encodeChannel(l, max(lx, 3), max(ly, 3))
	pDC, pAC, pScale := encodeChannel(p, 3, 3)
	qDC, qAC, qScale := encodeChannel(q, 3, 3)
	var aDC, aScale float32
	var aAC []float32
	if hasAlpha {
		aDC, aAC, aScale = encodeChannel(a, 5, 5)
	} else {
		aDC, aAC, aScale = 1.0, []float32{}, 1.0
	}

	// Write the constants
	isLandscape := width > height
	header24 := uint32(round(63.0*lDC)) |
		(uint32(round(31.5+31.5*pDC)) << 6) |
		(uint32(round(31.5+31.5*qDC)) << 12) |
		(uint32(round(31.0*lScale)) << 18)
	if hasAlpha {
		header24 |= 1 << 23
	}
	header16 := uint16(0)
	if isLandscape {
		header16 = uint16(ly)
	} else {
		header16 = uint16(lx)
	}
	header16 |= (uint16(round(63.0*pScale)) << 3) |
		(uint16(round(63.0*qScale)) << 9)
	if isLandscape {
		header16 |= 1 << 15
	}

	hash := make([]byte, 0, 25)
	hash = append(hash, byte(header24&255), byte((header24>>8)&255), byte(header24>>16), byte(header16&255), byte(header16>>8))
	isOdd := false
	if hasAlpha {
		hash = append(hash, byte(round(15.0*aDC))|byte(int(round(15.0*aScale))<<4))
	}

	// Write the varying factors
	acArrays := [][]float32{lAC, pAC, qAC}
	for _, ac := range acArrays {
		for _, f := range ac {
			u := byte(round(15.0 * f))
			if isOdd {
				hash[len(hash)-1] |= u << 4
			} else {
				hash = append(hash, u)
			}
			isOdd = !isOdd
		}
	}
	if hasAlpha {
		for _, f := range aAC {
			u := byte(round(15.0 * f))
			if isOdd {
				hash[len(hash)-1] |= u << 4
			} else {
				hash = append(hash, u)
			}
			isOdd = !isOdd
		}
	}

	return hash, nil
}

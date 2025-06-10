package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"os"
)

const (
	D = 4
	// Audio CD sector size
	SectorSize = 2352
	// Total size for CD (800MB)
	CDTotalSize = 800 * 1024 * 1024
	// Total size for DVD (4.7GB)
	DVDTotalSize = 4700 * 1024 * 1024
)

// delays array from original C++ code
var delays = [24]int{
	-24 * (3), -24*(1*D+2) + 1, 8 - 24*(2*D+3), 8 - 24*(3*D+2) + 1,
	16 - 24*(4*D+3), 16 - 24*(5*D+2) + 1, 2 - 24*(6*D+3), 2 - 24*(7*D+2) + 1,
	10 - 24*(8*D+3), 10 - 24*(9*D+2) + 1, 18 - 24*(10*D+3), 18 - 24*(11*D+2) + 1,

	4 - 24*(16*D+1), 4 - 24*(17*D) + 1, 12 - 24*(18*D+1), 12 - 24*(19*D) + 1,
	20 - 24*(20*D+1), 20 - 24*(21*D) + 1, 6 - 24*(22*D+1), 6 - 24*(23*D) + 1,
	14 - 24*(24*D+1), 14 - 24*(25*D) + 1, 22 - 24*(26*D+1), 22 - 24*(27*D) + 1,
}

// palette from original code
var palette = [4]byte{0x10, 0x21, 0x28, 0xAA}

// Converter handles the image to audio track conversion
type Converter struct {
	tr0       float64
	dtr       float64
	r0        float64
	mixColors bool
	discType  string
	
	// Internal state
	intseq  [24 * 28 * D]byte
	nh      int
	pinf    int
	buffer  [SectorSize]byte
	c       int
	
	// Progress tracking
	progressCallback func(int)
	cancelCallback   func() bool
}

// NewConverter creates a new converter with the given parameters
func NewConverter(tr0, dtr, r0 float64, mixColors bool, discType string) *Converter {
	return &Converter{
		tr0:       tr0,
		dtr:       dtr,
		r0:        r0,
		mixColors: mixColors,
		discType:  discType,
		nh:        28*D - 1,
		pinf:      0,
		c:         0,
	}
}

// SetProgressCallback sets a callback function for progress updates
func (conv *Converter) SetProgressCallback(callback func(int)) {
	conv.progressCallback = callback
}

// SetCancelCallback sets a callback function to check for cancellation
func (conv *Converter) SetCancelCallback(callback func() bool) {
	conv.cancelCallback = callback
}

// Convert converts an image to an audio track file
func (conv *Converter) Convert(ctx context.Context, img image.Image, filename string) error {
	// Determine total size based on disc type
	totalSize := CDTotalSize
	if conv.discType == "dvd" {
		totalSize = DVDTotalSize
	}
	
	// Create output file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	// Convert image bounds
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()
	
	// Initialize variables
	tr := conv.tr0
	r := conv.r0
	dr := conv.dtr * conv.r0 / conv.tr0
	c := 0.0
	
	// Disc geometry constants
	ir := 1500.0     // Image radius
	rcd := 57.5      // CD radius
	cx := float64(imgWidth) / 2
	cy := float64(imgHeight) / 2
	
	zs := 0
	zf := 0
	
	for c < float64(totalSize)-tr {
		// Check for cancellation
		if conv.cancelCallback != nil && conv.cancelCallback() {
			file.Close()
			os.Remove(filename)
			return fmt.Errorf("conversion cancelled")
		}
		
		// Check for context cancellation
		select {
		case <-ctx.Done():
			file.Close()
			os.Remove(filename)
			return ctx.Err()
		default:
		}
		
		// Update progress
		if conv.progressCallback != nil {
			progress := int(100 * c / float64(totalSize))
			conv.progressCallback(progress)
		}
		
		itr := int(tr)
		ri := ir * r / rcd
		
		// Process one track
		for i := 0; i < itr; i++ {
			alpha := 2 * math.Pi * float64(i) / float64(itr)
			xi := cx + ri*math.Cos(alpha)
			yi := cy + ri*math.Sin(alpha)
			
			// Sample the image
			pixelColor := conv.sampleImage(img, int(xi), int(yi), imgWidth, imgHeight)
			grayValue := conv.rgbaToGray(pixelColor)
			
			c1 := grayValue / 85
			c2 := c1 + 1
			if c2 > 3 {
				c2 = 3
			}
			
			var cl byte
			grayMod := int(grayValue % 85)
			if conv.mixColors {
				if rand.Intn(85) < grayMod || grayMod == 84 {
					cl = c2
				} else {
					cl = c1
				}
			} else {
				if grayMod > (zs*5+zf) || grayMod == 84 {
					cl = c2
				} else {
					cl = c1
				}
			}
			
			if err := conv.ad(palette[cl], file); err != nil {
				return fmt.Errorf("failed to write data: %w", err)
			}
			
			zf++
			if zf >= 5 {
				zf = 0
			}
		}
		
		c += tr
		ic := int(c)
		
		// Fill remaining samples if needed
		for int(c) > ic {
			if err := conv.ad(palette[0], file); err != nil {
				return fmt.Errorf("failed to write data: %w", err)
			}
			ic++
			zf++
			if zf >= 4 {
				zf = 0
			}
		}
		
		tr += conv.dtr
		r += dr
		
		zs++
		if zs >= 17 {
			zs = 0
		}
	}
	
	// Flush remaining buffer
	if conv.c > 0 {
		if _, err := file.Write(conv.buffer[:conv.c]); err != nil {
			return fmt.Errorf("failed to write final buffer: %w", err)
		}
	}
	
	return nil
}

// sampleImage safely samples a pixel from the image
func (conv *Converter) sampleImage(img image.Image, x, y, width, height int) color.RGBA {
	// Clamp coordinates to image bounds
	if x < 0 {
		x = 0
	}
	if x >= width {
		x = width - 1
	}
	if y < 0 {
		y = 0
	}
	if y >= height {
		y = height - 1
	}
	
	// Convert to RGBA
	r, g, b, a := img.At(x, y).RGBA()
	return color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}

// rgbaToGray converts RGBA to grayscale using standard luminance formula
func (conv *Converter) rgbaToGray(c color.RGBA) byte {
	// Standard luminance formula: 0.299*R + 0.587*G + 0.114*B
	gray := float64(c.R)*0.299 + float64(c.G)*0.587 + float64(c.B)*0.114
	return byte(gray)
}

// ad processes a byte through the delay sequence (from original algorithm)
func (conv *Converter) ad(b byte, file *os.File) error {
	conv.intseq[conv.n2m(delays[conv.pinf])] = b
	conv.pinf++
	
	if conv.pinf >= 24 {
		conv.pinf = 0
		conv.nh++
		if conv.nh >= 28*4 {
			conv.nh = 0
		}
		
		for i := 0; i < 24; i++ {
			if err := conv.bw(conv.intseq[conv.n2m(i)], file); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// n2m calculates the correct index in the circular buffer
func (conv *Converter) n2m(n int) int {
	index := conv.nh*24 + n
	if index >= 28*4*24 {
		return index - 28*4*24
	} else if index < 0 {
		return index + 28*4*24
	}
	return index
}

// bw buffers bytes and writes to file when buffer is full
func (conv *Converter) bw(b byte, file *os.File) error {
	conv.buffer[conv.c] = b
	conv.c++
	
	if conv.c >= SectorSize {
		if _, err := file.Write(conv.buffer[:]); err != nil {
			return err
		}
		conv.c = 0
	}
	
	return nil
}
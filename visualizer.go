package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// TrackVisualizer creates a visual representation of how the track will appear on disc
type TrackVisualizer struct {
	tr0      float64
	dtr      float64
	r0       float64
	discType string
}

// NewTrackVisualizer creates a new track visualizer
func NewTrackVisualizer(tr0, dtr, r0 float64, discType string) *TrackVisualizer {
	return &TrackVisualizer{
		tr0:      tr0,
		dtr:      dtr,
		r0:       r0,
		discType: discType,
	}
}

// VisualizeTrack reads a raw audio track and creates a disc visualization using multiple threads
func (v *TrackVisualizer) VisualizeTrack(trackFile, outputImage string) error {
	// Open the track file
	file, err := os.Open(trackFile)
	if err != nil {
		return fmt.Errorf("failed to open track file: %w", err)
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stats: %w", err)
	}
	
	fmt.Printf("Track file size: %.1f MB\n", float64(stat.Size())/(1024*1024))
	
	// Read entire file into memory for multi-threading
	fmt.Println("Loading track data into memory...")
	trackData, err := os.ReadFile(trackFile)
	if err != nil {
		return fmt.Errorf("failed to read track file: %w", err)
	}
	
	// Create disc image (smaller for faster processing)
	discSize := 1500
	img := image.NewRGBA(image.Rect(0, 0, discSize, discSize))
	
	// Fill with dark background
	for y := 0; y < discSize; y++ {
		for x := 0; x < discSize; x++ {
			img.Set(x, y, color.RGBA{20, 20, 20, 255})
		}
	}
	
	// Draw the disc pattern
	centerX, centerY := float64(discSize)/2, float64(discSize)/2
	maxRadius := centerX * 0.9 // Disc outer edge
	minRadius := centerX * 0.08 // Center hole
	
	// Simulate the conversion process to map samples to disc positions
	fmt.Println("Simulating conversion process to map samples to disc positions...")
	
	totalSamples := len(trackData) / 4
	fmt.Printf("Total samples to process: %d\n", totalSamples)
	
	// Constants from converter (disc geometry)
	ir := 1500.0     // Image radius
	rcd := 57.5      // CD radius (mm)
	
	// Simulate the converter's main loop (matching exact algorithm)
	tr := v.tr0
	r := v.r0
	dr := v.dtr * v.r0 / v.tr0  // Key: dr calculation from converter
	c := 0.0
	sampleIndex := 0
	
	type pixelData struct {
		x, y  int
		color color.RGBA
	}
	var pixels []pixelData
	
	// Debug: let's see how far we get
	maxR := 0.0
	iterationCount := 0
	
	// Continue until we reach the edge of the disc (r ≈ 58mm for CD)  
	for r < 58.0 {
		if r > maxR {
			maxR = r
		}
		iterationCount++
		itr := int(tr)
		ri := ir * r / rcd
		
		// Map ri to visualization coordinates
		rNormalized := ri / ir // Normalize to 0-1
		visR := minRadius + rNormalized*(maxRadius-minRadius)
		
		// Process one track
		for i := 0; i < itr && sampleIndex < totalSamples; i++ {
			// Skip some samples for faster processing
			if i%5 != 0 { // Process every 5th sample in each track
				sampleIndex++
				continue
			}
			
			// Get audio sample if available, otherwise use silence/pattern
			var sample int16
			if sampleIndex < totalSamples && sampleIndex*4+3 < len(trackData) {
				leftSample := int16(binary.LittleEndian.Uint16(trackData[sampleIndex*4:sampleIndex*4+2]))
				rightSample := int16(binary.LittleEndian.Uint16(trackData[sampleIndex*4+2:sampleIndex*4+4]))
				
				// Use the stronger of left/right channels
				sample = leftSample
				if abs(int(rightSample)) > abs(int(leftSample)) {
					sample = rightSample
				}
			} else {
				// Beyond available data - show the disc structure with low intensity
				sample = int16(1000) // Low intensity to show the spiral structure
			}
			
			// Calculate position on disc
			alpha := 2.0 * math.Pi * float64(i) / float64(itr)
			x := centerX + visR*math.Cos(alpha)
			y := centerY + visR*math.Sin(alpha)
			
			// Check bounds
			if x >= 0 && x < float64(discSize) && y >= 0 && y < float64(discSize) {
				// Map sample value to color intensity
				intensity := float64(abs(int(sample))) / 32768.0 // Normalize to 0-1
				
				// Create pixel color based on intensity
				var pixelColor color.RGBA
				if intensity > 0.01 {
					// Enhance contrast dramatically for visibility
					scaledIntensity := math.Min(intensity*4.0, 1.0)
					brightness := uint8(scaledIntensity * 255)
					
					// Use a high-contrast color
					pixelColor = color.RGBA{
						R: brightness,
						G: brightness,
						B: uint8(float64(brightness) * 1.2), // Slight blue tint
						A: 255,
					}
				} else {
					// Dark areas for contrast
					pixelColor = color.RGBA{15, 15, 20, 255}
				}
				
				pixels = append(pixels, pixelData{int(x), int(y), pixelColor})
			}
			
			// Always increment sample index (even beyond available data)  
			sampleIndex++
		}
		
		// Update track parameters for next iteration (exactly matching converter)
		c += tr
		tr += v.dtr  // tr increases by dtr each iteration
		r += dr      // r increases by dr each iteration
		
		// Progress indicator with radius info
		if int(c)%1000000 == 0 {
			fmt.Printf("\rPosition: %.1fM samples, r=%.2fmm, tr=%.0f", c/1000000, r, tr)
		}
	}
	
	fmt.Printf("\rMapped %d pixels total\n", len(pixels))
	fmt.Printf("Debug: iterations=%d, maxR=%.2fmm, finalTr=%.0f, finalC=%.0f\n", iterationCount, maxR, tr, c)
	
	// Apply pixels to image
	fmt.Println("Rendering pixels to disc image...")
	for i, pixel := range pixels {
		// Set the pixel and add neighboring pixels for better visibility
		img.Set(pixel.x, pixel.y, pixel.color)
		
		// Add neighboring pixels with blending for anti-aliasing
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				px, py := pixel.x+dx, pixel.y+dy
				if px >= 0 && px < discSize && py >= 0 && py < discSize {
					// Blend with existing pixel
					existing := img.RGBAAt(px, py)
					blended := blendColors(existing, pixel.color, 0.3)
					img.Set(px, py, blended)
				}
			}
		}
		
		if i%100000 == 0 && i > 0 {
			fmt.Printf("\rRendered %d pixels...", i)
		}
	}
	
	fmt.Printf("\rRendered %d pixels total\n", len(pixels))
	
	// Draw center hole
	for y := 0; y < discSize; y++ {
		for x := 0; x < discSize; x++ {
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			distance := math.Sqrt(dx*dx + dy*dy)
			
			if distance < minRadius {
				img.Set(x, y, color.RGBA{0, 0, 0, 255}) // Black center hole
			} else if distance > maxRadius {
				img.Set(x, y, color.RGBA{10, 10, 10, 255}) // Dark outside area
			}
		}
	}
	
	// Save the visualization
	fmt.Println("Saving visualization...")
	outFile, err := os.Create(outputImage)
	if err != nil {
		return fmt.Errorf("failed to create output image: %w", err)
	}
	defer outFile.Close()
	
	err = png.Encode(outFile, img)
	if err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}
	
	fmt.Printf("Disc visualization saved to: %s\n", outputImage)
	return nil
}

// Helper functions
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func blendColors(c1, c2 color.RGBA, alpha float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c1.R)*(1-alpha) + float64(c2.R)*alpha),
		G: uint8(float64(c1.G)*(1-alpha) + float64(c2.G)*alpha),
		B: uint8(float64(c1.B)*(1-alpha) + float64(c2.B)*alpha),
		A: 255,
	}
}
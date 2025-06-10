package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

// loadImage loads an image file and returns an image.Image
func loadImage(filename string) (image.Image, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	// Determine file type by extension
	ext := strings.ToLower(filepath.Ext(filename))
	
	var img image.Image
	switch ext {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
	case ".png":
		img, err = png.Decode(file)
	default:
		// Try to decode with the imaging library which supports more formats
		file.Seek(0, 0)
		img, err = imaging.Decode(file)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return img, nil
}

// processImageForDisc processes the image to fit disc dimensions and convert to appropriate format
func processImageForDisc(img image.Image, discType string) image.Image {
	// Define disc dimensions (pixels for a 3000x3000 virtual disc)
	discSize := 3000
	
	// DVD has a larger data area, so we can use more of the disc surface
	dataAreaRadius := 1250.0 // CD data area
	if discType == "dvd" {
		dataAreaRadius = 1350.0 // DVD has larger data area
	}
	
	// Calculate the usable area (avoiding center hole and outer edge)
	centerHoleRadius := 375.0 // Center hole is about 15mm, scaled to our 3000px disc
	outerRadius := dataAreaRadius
	
	// Create a new image with disc dimensions
	processedImg := imaging.New(discSize, discSize, color.RGBA{255, 255, 255, 255})
	
	// Get original image bounds
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()
	
	// Calculate scaling to fit in the data area
	maxSize := int(2 * (outerRadius - centerHoleRadius))
	scaleFactor := float64(maxSize) / math.Max(float64(originalWidth), float64(originalHeight))
	
	// Resize the image to fit
	newWidth := int(float64(originalWidth) * scaleFactor)
	newHeight := int(float64(originalHeight) * scaleFactor)
	resizedImg := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
	
	// Convert to grayscale
	grayImg := imaging.Grayscale(resizedImg)
	
	// Paste the resized image onto the disc, centered
	centerX := discSize / 2
	centerY := discSize / 2
	offsetX := centerX - newWidth/2
	offsetY := centerY - newHeight/2
	
	processedImg = imaging.Paste(processedImg, grayImg, image.Pt(offsetX, offsetY))
	
	return processedImg
}

// Enhanced image processing that mimics the original CD preview behavior
func createDiscImage(img image.Image, discType string) image.Image {
	// Create a 3000x3000 disc image (matching original code)
	discSize := 3000
	discImg := imaging.New(discSize, discSize, color.RGBA{255, 255, 255, 255})
	
	// Get image bounds
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()
	
	// Calculate scaling - fit image to roughly half the disc radius
	maxRadius := 1200.0 // Usable radius for image
	if discType == "dvd" {
		maxRadius = 1300.0 // DVD has slightly larger usable area
	}
	
	// Scale image to fit within the usable area
	maxDimension := math.Max(float64(imgWidth), float64(imgHeight))
	scale := (2 * maxRadius) / maxDimension
	
	newWidth := int(float64(imgWidth) * scale)
	newHeight := int(float64(imgHeight) * scale)
	
	// Resize and convert to grayscale
	resizedImg := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
	grayImg := imaging.Grayscale(resizedImg)
	
	// Center the image on the disc
	centerX := discSize / 2
	centerY := discSize / 2
	offsetX := centerX - newWidth/2
	offsetY := centerY - newHeight/2
	
	// Paste the image onto the white disc background
	discImg = imaging.Paste(discImg, grayImg, image.Pt(offsetX, offsetY))
	
	return discImg
}
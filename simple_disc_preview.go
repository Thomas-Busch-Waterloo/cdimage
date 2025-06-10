package main

import (
	"image"
	"image/color"
	"image/draw"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"github.com/disintegration/imaging"
)

// SimpleDiscPreview creates a simple disc preview using basic Fyne components
type SimpleDiscPreview struct {
	*fyne.Container
	
	// Image state
	originalImage  image.Image
	processedImage image.Image
	imageCanvas    *canvas.Image
	discCanvas     *canvas.Raster
	
	// Position and scale
	imageScale float32
	imageOffsetX, imageOffsetY float32
	
	// Disc properties
	discType string
	containerSize fyne.Size
	
	// Callback
	onImageChanged func(image.Image)
	
	// Mouse state
	lastMousePos fyne.Position
	isDragging   bool
}

// NewSimpleDiscPreview creates a new simple disc preview
func NewSimpleDiscPreview() *SimpleDiscPreview {
	preview := &SimpleDiscPreview{
		imageScale:   1.0,
		imageOffsetX: 0,
		imageOffsetY: 0,
		discType:     "cd",
		containerSize: fyne.NewSize(400, 400),
	}
	
	// Create a simple disc using basic shapes instead of raster
	preview.discCanvas = nil
	
	// Create image canvas (initially empty)
	preview.imageCanvas = canvas.NewImageFromImage(nil)
	preview.imageCanvas.FillMode = canvas.ImageFillOriginal
	preview.imageCanvas.Translucency = 0.3
	preview.imageCanvas.Hide()
	
	// Create a very obvious debug disc to make sure it's visible
	discBg := canvas.NewCircle(color.RGBA{255, 0, 0, 255}) // Bright red for debug
	discBg.StrokeColor = color.RGBA{0, 0, 255, 255} // Blue border
	discBg.StrokeWidth = 8
	discBg.Resize(fyne.NewSize(300, 300))
	discBg.Move(fyne.NewPos(50, 50))
	
	centerHole := canvas.NewCircle(color.RGBA{0, 255, 0, 255}) // Bright green for debug
	centerHole.Resize(fyne.NewSize(60, 60))
	centerHole.Move(fyne.NewPos(170, 170))
	
	// Create container with disc shapes and image
	preview.Container = container.NewWithoutLayout(discBg, centerHole, preview.imageCanvas)
	
	// Force initial refresh to ensure disc is visible
	preview.Container.Refresh()
	
	return preview
}

// SetImage sets the image to preview
func (s *SimpleDiscPreview) SetImage(img image.Image) {
	if img == nil {
		s.originalImage = nil
		s.processedImage = nil
		s.imageCanvas.Hide()
		s.Container.Refresh()
		return
	}
	
	s.originalImage = img
	
	// Process image (convert to grayscale and resize)
	bounds := img.Bounds()
	maxSize := 200
	if s.discType == "dvd" {
		maxSize = 220
	}
	
	// Resize if too large
	if bounds.Dx() > maxSize || bounds.Dy() > maxSize {
		img = imaging.Resize(img, maxSize, maxSize, imaging.Lanczos)
	}
	
	// Convert to grayscale
	s.processedImage = imaging.Grayscale(img)
	
	// Update canvas
	s.imageCanvas.Image = s.processedImage
	s.imageCanvas.Show()
	s.centerImage()
	s.Container.Refresh()
}

// SetDiscType sets the disc type
func (s *SimpleDiscPreview) SetDiscType(discType string) {
	s.discType = discType
	// Disc type change doesn't need to update the simple circle visualization
}

// centerImage centers the image on the disc
func (s *SimpleDiscPreview) centerImage() {
	if s.processedImage == nil {
		return
	}
	
	bounds := s.processedImage.Bounds()
	imgWidth := float32(bounds.Dx()) * s.imageScale
	imgHeight := float32(bounds.Dy()) * s.imageScale
	
	// Center in container
	centerX := s.containerSize.Width / 2
	centerY := s.containerSize.Height / 2
	
	s.imageOffsetX = centerX - imgWidth/2
	s.imageOffsetY = centerY - imgHeight/2
	
	s.updateImagePosition()
}

// updateImagePosition updates the image canvas position and size
func (s *SimpleDiscPreview) updateImagePosition() {
	if s.processedImage == nil {
		return
	}
	
	bounds := s.processedImage.Bounds()
	scaledWidth := float32(bounds.Dx()) * s.imageScale
	scaledHeight := float32(bounds.Dy()) * s.imageScale
	
	s.imageCanvas.Resize(fyne.NewSize(scaledWidth, scaledHeight))
	s.imageCanvas.Move(fyne.NewPos(s.imageOffsetX, s.imageOffsetY))
	s.imageCanvas.Refresh()
	
	if s.onImageChanged != nil {
		s.onImageChanged(s.getFinalImage())
	}
}

// SetOnImageChanged sets the callback for image changes
func (s *SimpleDiscPreview) SetOnImageChanged(callback func(image.Image)) {
	s.onImageChanged = callback
}

// Resize handles container resize
func (s *SimpleDiscPreview) Resize(size fyne.Size) {
	s.containerSize = size
	s.Container.Resize(size)
	
	// Update disc shapes to fit new size
	if len(s.Container.Objects) >= 2 {
		// Resize disc background
		discSize := fyne.NewSize(size.Width*0.9, size.Height*0.9)
		discPos := fyne.NewPos((size.Width-discSize.Width)/2, (size.Height-discSize.Height)/2)
		s.Container.Objects[0].Resize(discSize)
		s.Container.Objects[0].Move(discPos)
		
		// Resize and reposition center hole
		holeSize := fyne.NewSize(40, 40)
		holePos := fyne.NewPos(size.Width/2-20, size.Height/2-20)
		s.Container.Objects[1].Resize(holeSize)
		s.Container.Objects[1].Move(holePos)
	}
	
	// Reposition image after resize
	if s.processedImage != nil {
		s.updateImagePosition()
	}
}

// ZoomIn zooms the image in
func (s *SimpleDiscPreview) ZoomIn() {
	s.zoom(1.2)
}

// ZoomOut zooms the image out
func (s *SimpleDiscPreview) ZoomOut() {
	s.zoom(0.8)
}

// zoom applies a zoom factor
func (s *SimpleDiscPreview) zoom(factor float32) {
	if s.processedImage == nil {
		return
	}
	
	oldScale := s.imageScale
	s.imageScale *= factor
	
	// Limit scale
	if s.imageScale < 0.1 {
		s.imageScale = 0.1
	} else if s.imageScale > 5.0 {
		s.imageScale = 5.0
	}
	
	// Adjust position to zoom towards center
	if s.imageScale != oldScale {
		// Calculate current center
		bounds := s.processedImage.Bounds()
		oldWidth := float32(bounds.Dx()) * oldScale
		oldHeight := float32(bounds.Dy()) * oldScale
		currentCenterX := s.imageOffsetX + oldWidth/2
		currentCenterY := s.imageOffsetY + oldHeight/2
		
		// Calculate new position
		newWidth := float32(bounds.Dx()) * s.imageScale
		newHeight := float32(bounds.Dy()) * s.imageScale
		s.imageOffsetX = currentCenterX - newWidth/2
		s.imageOffsetY = currentCenterY - newHeight/2
	}
	
	s.updateImagePosition()
}

// StartDrag starts a drag operation
func (s *SimpleDiscPreview) StartDrag(pos fyne.Position) {
	s.lastMousePos = pos
	s.isDragging = true
}

// Drag moves the image during drag
func (s *SimpleDiscPreview) Drag(pos fyne.Position) {
	if !s.isDragging || s.processedImage == nil {
		return
	}
	
	deltaX := pos.X - s.lastMousePos.X
	deltaY := pos.Y - s.lastMousePos.Y
	
	s.imageOffsetX += deltaX
	s.imageOffsetY += deltaY
	
	s.lastMousePos = pos
	s.updateImagePosition()
}

// EndDrag ends the drag operation
func (s *SimpleDiscPreview) EndDrag() {
	s.isDragging = false
}


// getFinalImage returns the final composed image
func (s *SimpleDiscPreview) getFinalImage() image.Image {
	if s.processedImage == nil {
		return nil
	}
	
	// Create final image (3000x3000)
	finalImg := image.NewRGBA(image.Rect(0, 0, 3000, 3000))
	
	// Fill with white background
	draw.Draw(finalImg, finalImg.Bounds(), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
	
	// Calculate scaling from preview to final
	scale := 3000.0 / float64(s.containerSize.Width)
	
	// Scale and position the image
	bounds := s.processedImage.Bounds()
	finalWidth := int(float64(bounds.Dx()) * float64(s.imageScale) * scale)
	finalHeight := int(float64(bounds.Dy()) * float64(s.imageScale) * scale)
	
	if finalWidth > 0 && finalHeight > 0 {
		resizedImg := imaging.Resize(s.processedImage, finalWidth, finalHeight, imaging.Lanczos)
		
		finalX := int(float64(s.imageOffsetX) * scale)
		finalY := int(float64(s.imageOffsetY) * scale)
		
		drawRect := image.Rect(finalX, finalY, finalX+finalWidth, finalY+finalHeight)
		draw.Draw(finalImg, drawRect, resizedImg, image.Point{}, draw.Over)
	}
	
	return finalImg
}
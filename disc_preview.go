package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/disintegration/imaging"
)

// DiscPreview is an interactive disc visualization widget
type DiscPreview struct {
	widget.BaseWidget
	
	// Image state
	originalImage image.Image
	processedImage image.Image
	imageRect      fyne.Position
	imageSize      fyne.Size
	imageScale     float32
	
	// Disc parameters
	discType       string
	discSize       fyne.Size
	
	// Mouse interaction
	lastClickPos   fyne.Position
	isDragging     bool
	
	// Callback for image changes
	onImageChanged func(image.Image)
}

// NewDiscPreview creates a new interactive disc preview
func NewDiscPreview() *DiscPreview {
	preview := &DiscPreview{
		imageRect:  fyne.NewPos(0, 0),
		imageSize:  fyne.NewSize(200, 200),
		imageScale: 1.0,
		discType:   "cd",
		discSize:   fyne.NewSize(400, 400), // Default size, will be updated by Layout
	}
	preview.ExtendBaseWidget(preview)
	return preview
}

// SetImage sets the image to be previewed on the disc
func (d *DiscPreview) SetImage(img image.Image) {
	if img == nil {
		d.originalImage = nil
		d.processedImage = nil
		d.Refresh()
		return
	}
	
	d.originalImage = img
	d.processImageForDisc()
	
	// Give the layout a chance to set the proper disc size before centering
	// If disc size is not set, use a reasonable default
	if d.discSize.Width == 0 || d.discSize.Height == 0 {
		d.discSize = fyne.NewSize(400, 400)
	}
	
	d.centerImage()
	d.Refresh()
	
	if d.onImageChanged != nil {
		d.onImageChanged(d.getFinalImage())
	}
}

// SetDiscType sets the disc type (cd or dvd)
func (d *DiscPreview) SetDiscType(discType string) {
	d.discType = discType
	if d.originalImage != nil {
		d.processImageForDisc()
		d.Refresh()
	}
}

// SetOnImageChanged sets the callback for when the image positioning changes
func (d *DiscPreview) SetOnImageChanged(callback func(image.Image)) {
	d.onImageChanged = callback
}

// processImageForDisc converts the image to grayscale and scales it appropriately
func (d *DiscPreview) processImageForDisc() {
	if d.originalImage == nil {
		return
	}
	
	// Convert to grayscale
	grayImg := imaging.Grayscale(d.originalImage)
	
	// Scale to appropriate size for disc preview
	bounds := grayImg.Bounds()
	maxSize := 200
	if d.discType == "dvd" {
		maxSize = 220 // DVD can show slightly larger images
	}
	
	if bounds.Dx() > maxSize || bounds.Dy() > maxSize {
		grayImg = imaging.Resize(grayImg, maxSize, maxSize, imaging.Lanczos)
		bounds = grayImg.Bounds()
	}
	
	d.processedImage = grayImg
	d.imageSize = fyne.NewSize(float32(bounds.Dx()), float32(bounds.Dy()))
}

// centerImage centers the image on the disc
func (d *DiscPreview) centerImage() {
	if d.processedImage == nil {
		return
	}
	
	// Use the actual current disc size, ensuring it's valid
	discWidth := d.discSize.Width
	discHeight := d.discSize.Height
	
	if discWidth == 0 || discHeight == 0 {
		// Fallback to minimum size if not initialized
		discWidth = 300
		discHeight = 300
	}
	
	center := fyne.NewPos(discWidth/2, discHeight/2)
	d.imageRect = fyne.NewPos(
		center.X-d.imageSize.Width*d.imageScale/2,
		center.Y-d.imageSize.Height*d.imageScale/2,
	)
	
	d.Refresh()
	if d.onImageChanged != nil {
		d.onImageChanged(d.getFinalImage())
	}
}

// Focusable interface implementation
func (d *DiscPreview) FocusGained() {
	// Widget gained focus - can now receive keyboard events
}

func (d *DiscPreview) FocusLost() {
	// Widget lost focus
}

func (d *DiscPreview) TypedRune(r rune) {
	// Handle typed characters if needed
}

func (d *DiscPreview) TypedKey(key *fyne.KeyEvent) {
	d.KeyDown(key)
}

// Desktop mouse interface implementation  
func (d *DiscPreview) MouseIn(*desktop.MouseEvent) {
	// Mouse entered widget
}

func (d *DiscPreview) MouseOut() {
	// Mouse left widget
}

func (d *DiscPreview) MouseMoved(*desktop.MouseEvent) {
	// Mouse moved within widget
}

// zoomByFactor zooms the image by the given factor
func (d *DiscPreview) zoomByFactor(factor float32) {
	if d.processedImage == nil {
		return
	}
	
	oldScale := d.imageScale
	d.imageScale *= factor
	
	// Limit scale
	if d.imageScale < 0.1 {
		d.imageScale = 0.1
	} else if d.imageScale > 5.0 {
		d.imageScale = 5.0
	}
	
	// Adjust position to zoom towards center
	if d.imageScale != oldScale {
		currentCenterX := d.imageRect.X + d.imageSize.Width*oldScale/2
		currentCenterY := d.imageRect.Y + d.imageSize.Height*oldScale/2
		
		d.imageRect.X = currentCenterX - d.imageSize.Width*d.imageScale/2
		d.imageRect.Y = currentCenterY - d.imageSize.Height*d.imageScale/2
	}
	
	d.Refresh()
	if d.onImageChanged != nil {
		d.onImageChanged(d.getFinalImage())
	}
}

// CreateRenderer creates the visual representation
func (d *DiscPreview) CreateRenderer() fyne.WidgetRenderer {
	return &discPreviewRenderer{preview: d}
}

// getFinalImage returns the final composed image as it would appear on disc
func (d *DiscPreview) getFinalImage() image.Image {
	if d.processedImage == nil {
		return nil
	}
	
	// Create a disc-sized image (3000x3000 like original)
	finalImg := image.NewRGBA(image.Rect(0, 0, 3000, 3000))
	
	// Fill with white background
	draw.Draw(finalImg, finalImg.Bounds(), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
	
	// Calculate scaling from preview to final image
	previewToFinal := 3000.0 / float64(d.discSize.Width)
	
	// Scale and position the image
	scaledWidth := int(float64(d.imageSize.Width) * float64(d.imageScale) * previewToFinal)
	scaledHeight := int(float64(d.imageSize.Height) * float64(d.imageScale) * previewToFinal)
	
	if scaledWidth > 0 && scaledHeight > 0 {
		resizedImg := imaging.Resize(d.processedImage, scaledWidth, scaledHeight, imaging.Lanczos)
		
		// Calculate final position
		finalX := int((float64(d.imageRect.X) + float64(d.imageSize.Width*d.imageScale)/2) * previewToFinal - float64(scaledWidth)/2)
		finalY := int((float64(d.imageRect.Y) + float64(d.imageSize.Height*d.imageScale)/2) * previewToFinal - float64(scaledHeight)/2)
		
		// Draw the image onto the final image
		drawRect := image.Rect(finalX, finalY, finalX+scaledWidth, finalY+scaledHeight)
		draw.Draw(finalImg, drawRect, resizedImg, image.Point{}, draw.Over)
	}
	
	return finalImg
}

// Mouse event handlers
func (d *DiscPreview) Tapped(ev *fyne.PointEvent) {
	// Single tap doesn't do anything, double-tap centers
}

func (d *DiscPreview) DoubleTapped(ev *fyne.PointEvent) {
	// Double-tap to center
	d.centerImage()
}

func (d *DiscPreview) TappedSecondary(ev *fyne.PointEvent) {
	// Right-click menu could be added here
}

func (d *DiscPreview) Dragged(ev *fyne.DragEvent) {
	if d.processedImage == nil {
		return
	}
	
	// Move the image
	d.imageRect.X += ev.Dragged.DX
	d.imageRect.Y += ev.Dragged.DY
	
	d.Refresh()
	if d.onImageChanged != nil {
		d.onImageChanged(d.getFinalImage())
	}
}

func (d *DiscPreview) DragEnd() {
	// Drag ended
}

func (d *DiscPreview) KeyDown(key *fyne.KeyEvent) {
	if d.processedImage == nil {
		return
	}
	
	switch key.Name {
	case fyne.KeyPlus, fyne.KeyEqual: // + or = key (zoom in)
		d.zoomByFactor(1.2)
	case fyne.KeyMinus: // - key (zoom out)
		d.zoomByFactor(0.8)
	case fyne.KeySpace: // Space bar (center)
		d.centerImage()
	}
}

func (d *DiscPreview) Scrolled(ev *fyne.ScrollEvent) {
	if d.processedImage == nil {
		return
	}
	
	// Handle both vertical and horizontal scrolling (trackpads often use both)
	var scrollDelta float32
	if ev.Scrolled.DY != 0 {
		scrollDelta = ev.Scrolled.DY
	} else if ev.Scrolled.DX != 0 {
		scrollDelta = ev.Scrolled.DX
	} else {
		return // No scroll movement
	}
	
	// Zoom the image with improved sensitivity for trackpads
	oldScale := d.imageScale
	zoomFactor := float32(1.0)
	
	// Adjust zoom factor based on scroll magnitude
	scrollMagnitude := scrollDelta
	if scrollMagnitude < 0 {
		scrollMagnitude = -scrollMagnitude
	}
	
	// More sensitive scaling for small trackpad movements
	if scrollMagnitude < 10 {
		// Small movements (typical trackpad)
		if scrollDelta > 0 {
			zoomFactor = 1.05 // Zoom in
		} else {
			zoomFactor = 0.95 // Zoom out
		}
	} else {
		// Larger movements (mouse wheel)
		if scrollDelta > 0 {
			zoomFactor = 1.1 // Zoom in
		} else {
			zoomFactor = 0.9 // Zoom out
		}
	}
	
	d.imageScale *= zoomFactor
	
	// Limit scale
	if d.imageScale < 0.1 {
		d.imageScale = 0.1
	} else if d.imageScale > 5.0 {
		d.imageScale = 5.0
	}
	
	// Adjust position to zoom towards center
	if d.imageScale != oldScale {
		currentCenterX := d.imageRect.X + d.imageSize.Width*oldScale/2
		currentCenterY := d.imageRect.Y + d.imageSize.Height*oldScale/2
		
		d.imageRect.X = currentCenterX - d.imageSize.Width*d.imageScale/2
		d.imageRect.Y = currentCenterY - d.imageSize.Height*d.imageScale/2
	}
	
	d.Refresh()
	if d.onImageChanged != nil {
		d.onImageChanged(d.getFinalImage())
	}
}

// discPreviewRenderer handles the rendering
type discPreviewRenderer struct {
	preview *DiscPreview
	objects []fyne.CanvasObject
}

func (r *discPreviewRenderer) Layout(size fyne.Size) {
	// Update the preview's disc size
	r.preview.discSize = size
	
	// Update all canvas objects to fill the widget
	for _, obj := range r.objects {
		obj.Resize(size)
		obj.Move(fyne.NewPos(0, 0)) // Ensure objects start at origin
	}
}

func (r *discPreviewRenderer) MinSize() fyne.Size {
	return fyne.NewSize(300, 300)
}

func (r *discPreviewRenderer) Refresh() {
	r.objects = r.createDiscVisualization()
	canvas.Refresh(r.preview)
}

func (r *discPreviewRenderer) Objects() []fyne.CanvasObject {
	if len(r.objects) == 0 {
		r.objects = r.createDiscVisualization()
	}
	return r.objects
}

func (r *discPreviewRenderer) Destroy() {
	// Cleanup if needed
}

// createDiscVisualization creates the visual representation of the disc
func (r *discPreviewRenderer) createDiscVisualization() []fyne.CanvasObject {
	var objects []fyne.CanvasObject
	
	size := r.preview.discSize
	if size.Width == 0 || size.Height == 0 {
		size = fyne.NewSize(400, 400)
	}
	
	// Create disc background with gradient effect
	discBg := canvas.NewRasterWithPixels(func(x, y, w, h int) color.Color {
		centerX, centerY := float64(w)/2, float64(h)/2
		dx, dy := float64(x)-centerX, float64(y)-centerY
		distance := math.Sqrt(dx*dx + dy*dy)
		maxRadius := math.Min(centerX, centerY)
		
		// Outside disc area
		if distance > maxRadius {
			return color.RGBA{40, 40, 40, 255} // Dark background
		}
		
		// Create metallic disc effect
		normalizedDist := distance / maxRadius
		
		// Outer rim
		if normalizedDist > 0.95 {
			return color.RGBA{180, 180, 180, 255}
		}
		
		// Data area with slight blue tint
		if normalizedDist > 0.15 {
			intensity := uint8(200 + 30*math.Sin(normalizedDist*math.Pi*8)) // Spiral effect
			return color.RGBA{intensity, intensity, intensity + 20, 255}
		}
		
		// Center hole
		return color.RGBA{0, 0, 0, 255}
	})
	
	discBg.Resize(size)
	objects = append(objects, discBg)
	
	// Add image if present
	if r.preview.processedImage != nil {
		// Convert Go image to Fyne canvas image
		imgCanvas := canvas.NewImageFromImage(r.preview.processedImage)
		imgCanvas.FillMode = canvas.ImageFillOriginal
		
		// Apply scale
		scaledWidth := r.preview.imageSize.Width * r.preview.imageScale
		scaledHeight := r.preview.imageSize.Height * r.preview.imageScale
		imgCanvas.Resize(fyne.NewSize(scaledWidth, scaledHeight))
		imgCanvas.Move(r.preview.imageRect)
		
		// Make semi-transparent to show disc underneath
		imgCanvas.Translucency = 0.3
		
		objects = append(objects, imgCanvas)
	}
	
	// The center hole is already drawn in the raster background, no need for a separate object
	
	return objects
}
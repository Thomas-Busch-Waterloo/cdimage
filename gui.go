package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/disintegration/imaging"
)

// CDImageGUI represents the main GUI application
type CDImageGUI struct {
	app    fyne.App
	window fyne.Window
	
	// UI components
	imageLabel     *widget.Label
	previewCanvas  *canvas.Image
	discPreview    *SimpleDiscPreview
	progressBar    *widget.ProgressBar
	
	// Direct disc components for working visualization
	discContainer  *fyne.Container
	discCircle     *canvas.Circle
	centerHole     *canvas.Circle
	imageOverlay   *canvas.Image
	
	// Form inputs
	discTypeSelect  *widget.Select
	presetSelect    *widget.Select
	tr0Entry        *widget.Entry
	dtrEntry        *widget.Entry
	r0Entry         *widget.Entry
	mixColorsCheck  *widget.Check
	parallelCheck   *widget.Check
	outputEntry     *widget.Entry
	
	// Buttons
	loadImageBtn    *widget.Button
	convertBtn      *widget.Button
	burnBtn         *widget.Button
	centerImageBtn  *widget.Button
	zoomInBtn       *widget.Button
	zoomOutBtn      *widget.Button
	
	// Burning components
	driveSelect     *widget.Select
	availableDrives []OpticalDrive
	
	// State
	currentImage    image.Image
	currentImagePath string
	isConverting    bool
	cancelFunc      context.CancelFunc
}

// NewCDImageGUI creates a new GUI application
func NewCDImageGUI() *CDImageGUI {
	myApp := app.New()
	myApp.SetIcon(theme.DocumentIcon())
	
	window := myApp.NewWindow("CDImage - Burn Pictures to CD/DVD")
	window.Resize(fyne.NewSize(900, 1000)) // Much taller to accommodate all content
	window.SetMaster()
	
	gui := &CDImageGUI{
		app:    myApp,
		window: window,
	}
	
	gui.setupUI()
	return gui
}

// setupUI initializes all UI components
func (gui *CDImageGUI) setupUI() {
	// Create components
	gui.createComponents()
	
	// Layout the UI
	content := gui.createLayout()
	
	gui.window.SetContent(content)
	
	// Set initial values
	gui.resetForm()
}

// createComponents creates all UI widgets
func (gui *CDImageGUI) createComponents() {
	// Image section
	gui.imageLabel = widget.NewLabel("No image loaded")
	gui.previewCanvas = canvas.NewImageFromImage(nil)
	gui.previewCanvas.FillMode = canvas.ImageFillContain
	gui.previewCanvas.SetMinSize(fyne.NewSize(300, 300))
	
	// Create enhanced disc preview
	gui.discPreview = NewSimpleDiscPreview()
	gui.discPreview.SetOnImageChanged(func(img image.Image) {
		// Update the current processed image when user adjusts positioning
		gui.currentImage = img
	})
	
	// Progress bar
	gui.progressBar = widget.NewProgressBar()
	gui.progressBar.Hide()
	
	// Form inputs
	gui.discTypeSelect = widget.NewSelect([]string{"CD", "DVD"}, func(value string) {
		gui.updatePresetOptions()
	})
	
	gui.presetSelect = widget.NewSelect([]string{}, func(value string) {
		gui.loadPresetValues(value)
	})
	
	// Set initial selection after both widgets are created
	gui.discTypeSelect.SetSelected("CD")
	
	gui.tr0Entry = widget.NewEntry()
	gui.tr0Entry.SetPlaceHolder("22951.52")
	
	gui.dtrEntry = widget.NewEntry()
	gui.dtrEntry.SetPlaceHolder("1.386596")
	
	gui.r0Entry = widget.NewEntry()
	gui.r0Entry.SetPlaceHolder("24.5")
	
	gui.mixColorsCheck = widget.NewCheck("Use random color mixing", nil)
	gui.parallelCheck = widget.NewCheck("Use multi-threaded conversion", nil)
	gui.parallelCheck.SetChecked(true)
	
	gui.outputEntry = widget.NewEntry()
	gui.outputEntry.SetText("track.raw")
	
	// Buttons
	gui.loadImageBtn = widget.NewButtonWithIcon("Load Image", theme.FolderOpenIcon(), gui.loadImage)
	gui.convertBtn = widget.NewButtonWithIcon("Convert to Audio Track", theme.MediaPlayIcon(), gui.startConversion)
	gui.convertBtn.Disable()
	
	gui.centerImageBtn = widget.NewButtonWithIcon("Center", theme.MediaRecordIcon(), func() {
		gui.centerImageOnDisc()
	})
	gui.centerImageBtn.Disable()
	
	gui.zoomInBtn = widget.NewButtonWithIcon("Zoom In", theme.ZoomInIcon(), func() {
		gui.zoomImageOnDisc(1.2)
	})
	gui.zoomInBtn.Disable()
	
	gui.zoomOutBtn = widget.NewButtonWithIcon("Zoom Out", theme.ZoomOutIcon(), func() {
		gui.zoomImageOnDisc(0.8)
	})
	gui.zoomOutBtn.Disable()
	
	gui.burnBtn = widget.NewButtonWithIcon("Burn to Disc", theme.MediaRecordIcon(), gui.startBurning)
	gui.burnBtn.Disable()
	
	// Drive selection
	gui.detectOpticalDrives()
	driveOptions := []string{}
	for _, drive := range gui.availableDrives {
		driveOptions = append(driveOptions, fmt.Sprintf("%s (%s %s)", drive.Device, drive.Vendor, drive.Model))
	}
	gui.driveSelect = widget.NewSelect(driveOptions, nil)
	if len(driveOptions) > 0 {
		gui.driveSelect.SetSelected(driveOptions[0])
	}
}

// createLayout arranges all components in the window
func (gui *CDImageGUI) createLayout() fyne.CanvasObject {
	// Left panel - Image and disc preview
	imageControls := container.NewHBox(
		gui.centerImageBtn,
		gui.zoomInBtn,
		gui.zoomOutBtn,
		widget.NewLabel("Drag: move • Scroll/±buttons: zoom • Dbl-click/Space: center"),
	)
	
	imageSection := container.NewVBox(
		widget.NewCard("Image Preview", "", 
			container.NewVBox(
				gui.imageLabel,
				gui.previewCanvas,
			),
		),
		widget.NewCard("Interactive Disc Preview", "", 
			container.NewVBox(
				imageControls,
				gui.createInteractiveDiscContainer(),
			),
		),
	)
	
	// Right panel - Controls
	parametersForm := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Disc Type", gui.discTypeSelect),
			widget.NewFormItem("Preset", gui.presetSelect),
			widget.NewFormItem("TR0", gui.tr0Entry),
			widget.NewFormItem("DTR", gui.dtrEntry),
			widget.NewFormItem("R0", gui.r0Entry),
			widget.NewFormItem("Output File", gui.outputEntry),
		),
		gui.mixColorsCheck,
		gui.parallelCheck,
	)
	
	// Burning section
	burningForm := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Optical Drive", gui.driveSelect),
		),
		container.NewHBox(
			gui.burnBtn,
			widget.NewButton("Refresh Drives", func() {
				gui.detectOpticalDrives()
				gui.updateDriveOptions()
			}),
		),
	)
	
	controlsSection := container.NewVBox(
		widget.NewCard("Parameters", "", parametersForm),
		widget.NewCard("Actions", "", 
			container.NewVBox(
				container.NewHBox(
					gui.loadImageBtn,
					gui.convertBtn,
				),
				gui.progressBar,
			),
		),
		widget.NewCard("Burning", "", burningForm),
	)
	
	// Main layout
	content := container.NewHSplit(
		imageSection,
		controlsSection,
	)
	content.SetOffset(0.6) // Give more space to image preview
	
	return content
}

// createInteractiveDiscContainer creates a container with mouse and keyboard handling
func (gui *CDImageGUI) createInteractiveDiscContainer() fyne.CanvasObject {
	// Create the working disc visualization
	discBg := canvas.NewRectangle(color.RGBA{50, 50, 50, 255}) // Dark background
	discBg.Resize(fyne.NewSize(450, 450))
	
	discCircle := canvas.NewCircle(color.RGBA{200, 200, 255, 255}) // Light blue disc
	discCircle.StrokeColor = color.RGBA{100, 100, 200, 255}
	discCircle.StrokeWidth = 3
	discCircle.Resize(fyne.NewSize(350, 350))
	discCircle.Move(fyne.NewPos(50, 50))
	
	centerHole := canvas.NewCircle(color.RGBA{0, 0, 0, 255}) // Black center
	centerHole.Resize(fyne.NewSize(50, 50))
	centerHole.Move(fyne.NewPos(200, 200)) // Center: (450-50)/2 = 200
	
	// Create container with disc and prepare to add image overlay
	container := container.NewWithoutLayout(discBg, discCircle, centerHole)
	container.Resize(fyne.NewSize(450, 450)) // Slightly larger for better visibility
	
	// Store references for image overlay
	gui.discCircle = discCircle
	gui.discContainer = container
	gui.centerHole = centerHole
	
	// Handle drag events
	var dragging bool
	
	// Create a transparent event handler for interaction
	eventHandler := NewTransparentEventHandler()
	eventHandler.Resize(fyne.NewSize(450, 450))
	eventHandler.Move(fyne.NewPos(0, 0))
	// Store drag state
	var lastDragPos fyne.Position
	
	eventHandler.onDragStart = func(pos fyne.Position) {
		dragging = true
		lastDragPos = pos
	}
	eventHandler.onDrag = func(pos fyne.Position) {
		if dragging && gui.imageOverlay != nil {
			// Calculate drag delta
			deltaX := pos.X - lastDragPos.X
			deltaY := pos.Y - lastDragPos.Y
			
			// Move the image overlay
			currentPos := gui.imageOverlay.Position()
			gui.imageOverlay.Move(fyne.NewPos(currentPos.X+deltaX, currentPos.Y+deltaY))
			gui.discContainer.Refresh()
			
			lastDragPos = pos
		}
	}
	eventHandler.onDragEnd = func() {
		dragging = false
	}
	eventHandler.onDoubleClick = func() {
		gui.centerImageOnDisc()
	}
	eventHandler.onScroll = func(delta float32) {
		if gui.imageOverlay != nil {
			// Zoom the image
			currentSize := gui.imageOverlay.Size()
			scaleFactor := float32(1.0)
			
			if delta > 0 {
				scaleFactor = 1.1 // Zoom in
			} else {
				scaleFactor = 0.9 // Zoom out
			}
			
			newWidth := currentSize.Width * scaleFactor
			newHeight := currentSize.Height * scaleFactor
			
			// Limit size
			if newWidth < 20 || newHeight < 20 {
				return // Too small
			}
			if newWidth > 400 || newHeight > 400 {
				return // Too large
			}
			
			// Keep image centered during zoom
			currentPos := gui.imageOverlay.Position()
			centerX := currentPos.X + currentSize.Width/2
			centerY := currentPos.Y + currentSize.Height/2
			
			gui.imageOverlay.Resize(fyne.NewSize(newWidth, newHeight))
			gui.imageOverlay.Move(fyne.NewPos(centerX-newWidth/2, centerY-newHeight/2))
			gui.discContainer.Refresh()
		}
	}
	
	// Add the transparent event handler on top
	container.Add(eventHandler)
	return container
}

// TransparentEventHandler is a completely transparent widget for handling events
type TransparentEventHandler struct {
	widget.BaseWidget
	onDragStart   func(fyne.Position)
	onDrag        func(fyne.Position)
	onDragEnd     func()
	onDoubleClick func()
	onScroll      func(float32)
	dragging      bool
	lastPos       fyne.Position
}

// NewTransparentEventHandler creates a new transparent event handler
func NewTransparentEventHandler() *TransparentEventHandler {
	handler := &TransparentEventHandler{}
	handler.ExtendBaseWidget(handler)
	return handler
}

// CreateRenderer creates a completely transparent renderer
func (t *TransparentEventHandler) CreateRenderer() fyne.WidgetRenderer {
	return &transparentRenderer{}
}

// transparentRenderer renders nothing (completely transparent)
type transparentRenderer struct{}

func (r *transparentRenderer) Layout(size fyne.Size)         {}
func (r *transparentRenderer) MinSize() fyne.Size           { return fyne.NewSize(0, 0) }
func (r *transparentRenderer) Refresh()                     {}
func (r *transparentRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{} }
func (r *transparentRenderer) Destroy()                     {}

// Event handlers
func (t *TransparentEventHandler) Dragged(e *fyne.DragEvent) {
	if !t.dragging && t.onDragStart != nil {
		t.onDragStart(e.Position)
		t.dragging = true
		t.lastPos = e.Position
	}
	
	if t.dragging && t.onDrag != nil {
		t.onDrag(e.Position)
		t.lastPos = e.Position
	}
}

func (t *TransparentEventHandler) DragEnd() {
	if t.dragging && t.onDragEnd != nil {
		t.onDragEnd()
	}
	t.dragging = false
}

func (t *TransparentEventHandler) DoubleTapped(e *fyne.PointEvent) {
	if t.onDoubleClick != nil {
		t.onDoubleClick()
	}
}

func (t *TransparentEventHandler) Scrolled(e *fyne.ScrollEvent) {
	if t.onScroll != nil {
		delta := e.Scrolled.DY
		if delta == 0 {
			delta = e.Scrolled.DX // Handle horizontal scroll
		}
		t.onScroll(delta)
	}
}

func (t *TransparentEventHandler) Tapped(e *fyne.PointEvent) {
	// Single tap handling if needed
}

// addImageToDisc adds an image overlay to the disc visualization
func (gui *CDImageGUI) addImageToDisc(img image.Image) {
	if img == nil || gui.discContainer == nil {
		return
	}
	
	// Remove existing image overlay if present
	if gui.imageOverlay != nil {
		gui.discContainer.Remove(gui.imageOverlay)
	}
	
	// Process image (convert to grayscale and resize for preview)
	bounds := img.Bounds()
	maxSize := 150 // Reasonable size for disc overlay
	if bounds.Dx() > maxSize || bounds.Dy() > maxSize {
		img = imaging.Resize(img, maxSize, maxSize, imaging.Lanczos)
		bounds = img.Bounds()
	}
	
	grayImg := imaging.Grayscale(img)
	
	// Create image overlay
	gui.imageOverlay = canvas.NewImageFromImage(grayImg)
	gui.imageOverlay.FillMode = canvas.ImageFillOriginal
	gui.imageOverlay.Translucency = 0.4 // Semi-transparent
	
	// Position and size the image (centered on disc)
	imgWidth := float32(bounds.Dx())
	imgHeight := float32(bounds.Dy())
	gui.imageOverlay.Resize(fyne.NewSize(imgWidth, imgHeight))
	
	// Center on the disc (disc center is at 225,225, so center image there)
	gui.imageOverlay.Move(fyne.NewPos(225-imgWidth/2, 225-imgHeight/2))
	
	// Add to container (before the event handler so it's below it)
	objects := gui.discContainer.Objects
	if len(objects) > 0 && objects[len(objects)-1] != nil {
		// Insert before the last object (event handler)
		gui.discContainer.Objects = append(objects[:len(objects)-1], gui.imageOverlay, objects[len(objects)-1])
	} else {
		gui.discContainer.Add(gui.imageOverlay)
	}
	
	gui.discContainer.Refresh()
}

// centerImageOnDisc centers the image overlay on the disc
func (gui *CDImageGUI) centerImageOnDisc() {
	if gui.imageOverlay == nil {
		return
	}
	
	// Get current image size
	size := gui.imageOverlay.Size()
	
	// Center on the disc (disc center is at 225,225)
	gui.imageOverlay.Move(fyne.NewPos(225-size.Width/2, 225-size.Height/2))
	gui.discContainer.Refresh()
}

// zoomImageOnDisc zooms the image overlay by the given factor
func (gui *CDImageGUI) zoomImageOnDisc(factor float32) {
	if gui.imageOverlay == nil {
		return
	}
	
	// Get current size and position
	currentSize := gui.imageOverlay.Size()
	currentPos := gui.imageOverlay.Position()
	
	// Calculate new size
	newWidth := currentSize.Width * factor
	newHeight := currentSize.Height * factor
	
	// Limit size
	if newWidth < 20 || newHeight < 20 || newWidth > 400 || newHeight > 400 {
		return
	}
	
	// Keep image centered during zoom
	centerX := currentPos.X + currentSize.Width/2
	centerY := currentPos.Y + currentSize.Height/2
	
	gui.imageOverlay.Resize(fyne.NewSize(newWidth, newHeight))
	gui.imageOverlay.Move(fyne.NewPos(centerX-newWidth/2, centerY-newHeight/2))
	gui.discContainer.Refresh()
}

// detectOpticalDrives scans for available optical drives
func (gui *CDImageGUI) detectOpticalDrives() {
	drives, err := DetectOpticalDrives()
	if err != nil {
		drives = []OpticalDrive{} // Empty list on error
	}
	gui.availableDrives = drives
}

// updateDriveOptions updates the drive selection dropdown
func (gui *CDImageGUI) updateDriveOptions() {
	driveOptions := []string{}
	for _, drive := range gui.availableDrives {
		status := "Ready"
		if !drive.IsReady {
			status = "Not Ready"
		}
		
		capabilities := ""
		if drive.CanBurnCD && drive.CanBurnDVD {
			capabilities = "CD/DVD"
		} else if drive.CanBurnCD {
			capabilities = "CD"
		} else if drive.CanBurnDVD {
			capabilities = "DVD"
		} else {
			capabilities = "Read-only"
		}
		
		driveOptions = append(driveOptions, 
			fmt.Sprintf("%s - %s %s (%s, %s)", 
				drive.Device, drive.Vendor, drive.Model, capabilities, status))
	}
	
	gui.driveSelect.Options = driveOptions
	if len(driveOptions) > 0 {
		gui.driveSelect.SetSelected(driveOptions[0])
	}
	gui.driveSelect.Refresh()
}

// createDiscPreview creates a simple disc visualization (legacy method, keeping for compatibility)
func (gui *CDImageGUI) createDiscPreview() fyne.CanvasObject {
	// Create a simple disc representation
	disc := canvas.NewCircle(theme.PrimaryColor())
	disc.StrokeColor = theme.ForegroundColor()
	disc.StrokeWidth = 2
	disc.Resize(fyne.NewSize(200, 200))
	
	center := canvas.NewCircle(theme.BackgroundColor())
	center.StrokeColor = theme.ForegroundColor()
	center.StrokeWidth = 1
	center.Resize(fyne.NewSize(40, 40))
	
	container := container.NewWithoutLayout(disc, center)
	
	// Position center hole
	center.Move(fyne.NewPos(80, 80))
	
	return container
}

// loadImage opens file dialog and loads an image
func (gui *CDImageGUI) loadImage() {
	// Create file dialog with filters
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()
		
		// Load the image
		img, err := loadImage(reader.URI().Path())
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to load image: %w", err), gui.window)
			return
		}
		
		gui.currentImage = img
		gui.currentImagePath = reader.URI().Path()
		
		// Update UI
		filename := filepath.Base(gui.currentImagePath)
		gui.imageLabel.SetText(filename)
		
		// Update traditional preview
		processedImg := createDiscImage(img, strings.ToLower(gui.discTypeSelect.Selected))
		gui.previewCanvas.Image = processedImg
		gui.previewCanvas.Refresh()
		
		// Update interactive disc preview with direct image overlay
		gui.addImageToDisc(img)
		
		// Enable buttons
		gui.convertBtn.Enable()
		gui.centerImageBtn.Enable()
		gui.zoomInBtn.Enable()
		gui.zoomOutBtn.Enable()
		
	}, gui.window)
	
	// Set file filters
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".jpg", ".jpeg", ".png", ".gif", ".bmp"}))
	fileDialog.Show()
}

// updatePresetOptions updates preset dropdown based on selected disc type
func (gui *CDImageGUI) updatePresetOptions() {
	// Safety check to ensure widgets are initialized
	if gui.discTypeSelect == nil || gui.presetSelect == nil {
		return
	}
	
	discType := strings.ToLower(gui.discTypeSelect.Selected)
	presets := GetPresets()
	
	var options []string
	for key, preset := range presets {
		if preset.DiscType == discType {
			options = append(options, key)
		}
	}
	
	gui.presetSelect.Options = options
	if len(options) > 0 {
		gui.presetSelect.SetSelected(options[0])
		gui.loadPresetValues(options[0])
	}
	gui.presetSelect.Refresh()
	
	// Update disc preview with new disc type
	if gui.discPreview != nil {
		gui.discPreview.SetDiscType(discType)
	}
}

// loadPresetValues loads preset values into form fields
func (gui *CDImageGUI) loadPresetValues(presetKey string) {
	if presetKey == "" {
		return
	}
	
	// Safety check to ensure entry widgets are initialized
	if gui.tr0Entry == nil || gui.dtrEntry == nil || gui.r0Entry == nil {
		return
	}
	
	preset, exists := GetPresetByName(presetKey)
	if !exists {
		return
	}
	
	gui.tr0Entry.SetText(fmt.Sprintf("%.2f", preset.Tr0))
	gui.dtrEntry.SetText(fmt.Sprintf("%.6f", preset.Dtr))
	gui.r0Entry.SetText(fmt.Sprintf("%.1f", preset.R0))
}

// resetForm resets all form fields to defaults
func (gui *CDImageGUI) resetForm() {
	gui.discTypeSelect.SetSelected("CD")
	gui.updatePresetOptions()
	gui.mixColorsCheck.SetChecked(false)
	gui.parallelCheck.SetChecked(true)
	gui.outputEntry.SetText("track.raw")
}

// startConversion begins the image conversion process
func (gui *CDImageGUI) startConversion() {
	if gui.currentImage == nil {
		dialog.ShowError(fmt.Errorf("No image loaded"), gui.window)
		return
	}
	
	// Validate inputs
	tr0, err := strconv.ParseFloat(gui.tr0Entry.Text, 64)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Invalid TR0 value: %w", err), gui.window)
		return
	}
	
	dtr, err := strconv.ParseFloat(gui.dtrEntry.Text, 64)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Invalid DTR value: %w", err), gui.window)
		return
	}
	
	r0, err := strconv.ParseFloat(gui.r0Entry.Text, 64)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Invalid R0 value: %w", err), gui.window)
		return
	}
	
	if gui.outputEntry.Text == "" {
		dialog.ShowError(fmt.Errorf("Output file cannot be empty"), gui.window)
		return
	}
	
	// Disable UI during conversion
	gui.setConvertingState(true)
	
	// Start conversion in goroutine
	go gui.runConversion(tr0, dtr, r0)
}

// runConversion runs the actual conversion process
func (gui *CDImageGUI) runConversion(tr0, dtr, r0 float64) {
	defer gui.setConvertingState(false)
	
	discType := strings.ToLower(gui.discTypeSelect.Selected)
	mixColors := gui.mixColorsCheck.Checked
	useParallel := gui.parallelCheck.Checked
	outputFile := gui.outputEntry.Text
	
	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	gui.cancelFunc = cancel
	defer cancel()
	
	// Process image
	processedImg := createDiscImage(gui.currentImage, discType)
	
	// Create converter
	var converter interface {
		Convert(context.Context, image.Image, string) error
		SetProgressCallback(func(int))
		SetCancelCallback(func() bool)
	}
	
	if useParallel {
		converter = NewMultiThreadedConverter(tr0, dtr, r0, mixColors, discType)
	} else {
		converter = NewConverter(tr0, dtr, r0, mixColors, discType)
	}
	
	// Set up progress callback
	converter.SetProgressCallback(func(progress int) {
		// Update progress bar on main thread
		gui.progressBar.SetValue(float64(progress) / 100.0)
	})
	
	// Set up cancellation callback
	cancelled := false
	converter.SetCancelCallback(func() bool {
		return cancelled
	})
	
	// Start conversion
	var err error
	if useParallel {
		if mtconv, ok := converter.(*MultiThreadedConverter); ok {
			err = mtconv.ConvertParallel(ctx, processedImg, outputFile)
		} else {
			err = converter.Convert(ctx, processedImg, outputFile)
		}
	} else {
		err = converter.Convert(ctx, processedImg, outputFile)
	}
	
	// Show result
	if err != nil {
		if ctx.Err() != nil {
			dialog.ShowInformation("Cancelled", "Conversion was cancelled.", gui.window)
		} else {
			dialog.ShowError(fmt.Errorf("Conversion failed: %w", err), gui.window)
		}
	} else {
		// Enable burn button after successful conversion
		gui.burnBtn.Enable()
		dialog.ShowInformation("Success", 
			fmt.Sprintf("Conversion completed successfully!\nOutput: %s\n\nYou can now burn the track to a disc.", outputFile), 
			gui.window)
	}
}

// setConvertingState updates UI state during conversion
func (gui *CDImageGUI) setConvertingState(converting bool) {
	gui.isConverting = converting
	
	if converting {
		gui.convertBtn.SetText("Cancel")
		gui.convertBtn.SetIcon(theme.CancelIcon())
		gui.convertBtn.OnTapped = func() {
			if gui.cancelFunc != nil {
				gui.cancelFunc()
			}
		}
		gui.loadImageBtn.Disable()
		gui.progressBar.Show()
		gui.progressBar.SetValue(0)
	} else {
		gui.convertBtn.SetText("Convert to Audio Track")
		gui.convertBtn.SetIcon(theme.MediaPlayIcon())
		gui.convertBtn.OnTapped = gui.startConversion
		gui.loadImageBtn.Enable()
		gui.progressBar.Hide()
		gui.cancelFunc = nil
	}
}

// startBurning begins the disc burning process
func (gui *CDImageGUI) startBurning() {
	// Check if we have a track file to burn
	outputFile := gui.outputEntry.Text
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		dialog.ShowError(fmt.Errorf("No track file found. Please convert an image first."), gui.window)
		return
	}
	
	// Check if a drive is selected
	if gui.driveSelect.Selected == "" || len(gui.availableDrives) == 0 {
		dialog.ShowError(fmt.Errorf("No optical drive selected"), gui.window)
		return
	}
	
	// Find the selected drive
	selectedIndex := -1
	for i, option := range gui.driveSelect.Options {
		if option == gui.driveSelect.Selected {
			selectedIndex = i
			break
		}
	}
	
	if selectedIndex < 0 || selectedIndex >= len(gui.availableDrives) {
		dialog.ShowError(fmt.Errorf("Invalid drive selection"), gui.window)
		return
	}
	
	selectedDrive := gui.availableDrives[selectedIndex]
	discType := strings.ToLower(gui.discTypeSelect.Selected)
	
	// Check drive capabilities
	if discType == "cd" && !selectedDrive.CanBurnCD {
		dialog.ShowError(fmt.Errorf("Selected drive cannot burn CDs"), gui.window)
		return
	}
	if discType == "dvd" && !selectedDrive.CanBurnDVD {
		dialog.ShowError(fmt.Errorf("Selected drive cannot burn DVDs"), gui.window)
		return
	}
	
	// Show confirmation dialog with burning command
	command := GetBurningCommand(selectedDrive, outputFile, discType)
	
	confirmMsg := fmt.Sprintf(
		"Ready to burn to %s\n\n"+
		"Drive: %s (%s %s)\n"+
		"Disc Type: %s\n"+
		"Track File: %s\n\n"+
		"Command: %s\n\n"+
		"Make sure you have a blank %s disc in the drive.\n"+
		"This operation cannot be undone!",
		selectedDrive.Device, selectedDrive.Device, selectedDrive.Vendor, selectedDrive.Model,
		strings.ToUpper(discType), outputFile, command, strings.ToUpper(discType))
	
	dialog.ShowConfirm("Confirm Burning", confirmMsg, func(confirmed bool) {
		if confirmed {
			gui.performBurn(selectedDrive, outputFile, discType)
		}
	}, gui.window)
}

// performBurn executes the actual burning process
func (gui *CDImageGUI) performBurn(drive OpticalDrive, trackFile string, discType string) {
	// Disable UI during burning
	gui.burnBtn.SetText("Burning...")
	gui.burnBtn.Disable()
	gui.convertBtn.Disable()
	gui.loadImageBtn.Disable()
	
	go func() {
		defer func() {
			// Re-enable UI
			gui.burnBtn.SetText("Burn to Disc")
			gui.burnBtn.Enable()
			gui.convertBtn.Enable()
			gui.loadImageBtn.Enable()
		}()
		
		// Check for disc
		hasDisc, discStatus, err := CheckDiscInDrive(drive)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Error checking disc: %w", err), gui.window)
			return
		}
		
		if !hasDisc {
			dialog.ShowError(fmt.Errorf("Disc status: %s", discStatus), gui.window)
			return
		}
		
		// Perform the burn
		err = BurnAudioTrack(drive, trackFile, discType)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Burning failed: %w", err), gui.window)
		} else {
			dialog.ShowInformation("Success", 
				fmt.Sprintf("Successfully burned track to %s!\n\nYour image should now be visible on the disc surface.", 
					drive.Device), gui.window)
		}
	}()
}


// Run starts the GUI application
func (gui *CDImageGUI) Run() {
	gui.window.ShowAndRun()
}

// runGUI starts the GUI version of the application
func runGUI() {
	gui := NewCDImageGUI()
	gui.Run()
}
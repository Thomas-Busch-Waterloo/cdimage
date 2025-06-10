package main

import (
	"context"
	"fmt"
	"image"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/schollz/progressbar/v3"
)

// burnImage handles the main burning logic
func burnImage(inputFile, outputFile, discType string, tr0, dtr, r0 float64, mixColors bool, preset string, useMultithread bool) error {
	// Validate disc type
	discType = strings.ToLower(discType)
	if discType != "cd" && discType != "dvd" {
		return fmt.Errorf("invalid disc type: %s (must be 'cd' or 'dvd')", discType)
	}

	// Load image
	fmt.Printf("Loading image: %s\n", inputFile)
	img, err := loadImage(inputFile)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}

	// Process image for disc
	processedImg := createDiscImage(img, discType)

	// Determine parameters
	var discPreset DiscPreset
	var usePreset bool

	if preset != "" {
		var exists bool
		discPreset, exists = GetPresetByName(preset)
		if !exists {
			return fmt.Errorf("preset '%s' not found (use 'cdimage list-presets' to see available presets)", preset)
		}
		usePreset = true
		
		// Ensure preset matches disc type
		if discPreset.DiscType != discType {
			return fmt.Errorf("preset '%s' is for %s, but disc type is %s", preset, discPreset.DiscType, discType)
		}
	} else if tr0 == 0 || dtr == 0 {
		// Use default preset for disc type
		discPreset = GetDefaultPreset(discType)
		usePreset = true
		fmt.Printf("Using default preset for %s: %s\n", strings.ToUpper(discType), discPreset.Name)
	}

	// Set final parameters
	finalTr0 := tr0
	finalDtr := dtr
	finalR0 := r0

	if usePreset {
		finalTr0 = discPreset.Tr0
		finalDtr = discPreset.Dtr
		finalR0 = discPreset.R0
		fmt.Printf("Using preset: %s\n", discPreset.Name)
	}

	fmt.Printf("Parameters - tr0: %.2f, dtr: %.6f, r0: %.1f\n", finalTr0, finalDtr, finalR0)
	fmt.Printf("Mix colors: %t\n", mixColors)
	fmt.Printf("Multi-threading: %t\n", useMultithread)

	// Create progress bar
	bar := progressbar.NewOptions(100,
		progressbar.OptionSetDescription("Converting"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(100*time.Millisecond), // Update at most every 100ms
		progressbar.OptionShowCount(),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionClearOnFinish(),
	)

	// Create converter (choose between single and multi-threaded)
	var converter interface {
		Convert(context.Context, image.Image, string) error
		SetProgressCallback(func(int))
		SetCancelCallback(func() bool)
	}

	if useMultithread {
		converter = NewMultiThreadedConverter(finalTr0, finalDtr, finalR0, mixColors, discType)
	} else {
		converter = NewConverter(finalTr0, finalDtr, finalR0, mixColors, discType)
	}

	// Set up progress tracking with throttling
	lastUpdate := time.Now()
	lastProgress := -1
	converter.SetProgressCallback(func(progress int) {
		now := time.Now()
		// Only update if progress changed and enough time has passed (100ms)
		if progress != lastProgress && now.Sub(lastUpdate) > 100*time.Millisecond {
			bar.Set(progress)
			lastUpdate = now
			lastProgress = progress
		}
	})

	// Set up cancellation handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	cancelled := false
	converter.SetCancelCallback(func() bool {
		return cancelled
	})

	go func() {
		<-sigChan
		fmt.Printf("\nReceived interrupt signal, cancelling...\n")
		cancelled = true
		cancel()
	}()

	// Start conversion
	fmt.Printf("Converting image to %s audio track...\n", strings.ToUpper(discType))
	fmt.Printf("Output file: %s\n", outputFile)
	
	startTime := time.Now()
	
	// Choose conversion method based on threading option
	var convErr error
	if useMultithread {
		if mtconv, ok := converter.(*MultiThreadedConverter); ok {
			convErr = mtconv.ConvertParallel(ctx, processedImg, outputFile)
		} else {
			convErr = converter.Convert(ctx, processedImg, outputFile)
		}
	} else {
		convErr = converter.Convert(ctx, processedImg, outputFile)
	}
	
	duration := time.Since(startTime)
	
	// Ensure progress bar reaches 100% and finishes cleanly
	bar.Set(100)
	bar.Finish()
	fmt.Printf("\n") // Add single newline after progress bar

	if convErr != nil {
		if cancelled || ctx.Err() != nil {
			fmt.Printf("Conversion cancelled.\n")
			return nil
		}
		return fmt.Errorf("conversion failed: %w", convErr)
	}

	// Check if file was created successfully
	if info, err := os.Stat(outputFile); err != nil {
		return fmt.Errorf("output file was not created: %w", err)
	} else {
		fileSize := float64(info.Size()) / (1024 * 1024) // Size in MB
		fmt.Printf("\nConversion completed successfully!\n")
		fmt.Printf("Duration: %v\n", duration.Truncate(time.Second))
		fmt.Printf("Output file size: %.1f MB\n", fileSize)
		fmt.Printf("\nTo burn the track to a %s:\n", strings.ToUpper(discType))
		
		if discType == "cd" {
			fmt.Printf("  cdrecord -audio dev=/dev/sr0 %s\n", outputFile)
			fmt.Printf("  OR\n")
			fmt.Printf("  wodim -audio dev=/dev/sr0 %s\n", outputFile)
		} else {
			fmt.Printf("  growisofs -audio -Z /dev/sr0=%s\n", outputFile)
			fmt.Printf("  OR\n")
			fmt.Printf("  cdrecord -audio dev=/dev/sr0 %s\n", outputFile)
		}
		
		fmt.Printf("\nNote: Replace /dev/sr0 with your actual optical drive device.\n")
		fmt.Printf("Use 'cdrecord -scanbus' or 'wodim -scanbus' to find your drive.\n")
	}

	return nil
}
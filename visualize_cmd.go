package main

import (
	"fmt"
	"strconv"
	"strings"
)

// visualizeTrack creates a visual representation of a raw track file
func visualizeTrack(trackFile, outputImage, discType string, tr0, dtr, r0 float64, preset string) error {
	// Validate input file
	if trackFile == "" {
		return fmt.Errorf("track file is required")
	}

	// Validate disc type
	discType = strings.ToLower(discType)
	if discType != "cd" && discType != "dvd" {
		return fmt.Errorf("disc type must be 'cd' or 'dvd'")
	}

	// Use preset if specified
	if preset != "" {
		presetData, exists := GetPresetByName(preset)
		if !exists {
			return fmt.Errorf("preset '%s' not found. Use 'list-presets' to see available presets", preset)
		}
		
		// Only use preset values if not explicitly overridden
		if tr0 == 0 {
			tr0 = presetData.Tr0
		}
		if dtr == 0 {
			dtr = presetData.Dtr
		}
		
		fmt.Printf("Using preset: %s (%s)\n", preset, presetData.Name)
	} else {
		// Use default values for disc type if no preset specified
		if tr0 == 0 || dtr == 0 {
			defaultPresets := GetPresets()
			var defaultKey string
			
			if discType == "cd" {
				defaultKey = "verbatim-cd-rw-1"
			} else {
				defaultKey = "generic-dvd-r"
			}
			
			if defaultPreset, exists := defaultPresets[defaultKey]; exists {
				if tr0 == 0 {
					tr0 = defaultPreset.Tr0
				}
				if dtr == 0 {
					dtr = defaultPreset.Dtr
				}
				fmt.Printf("Using default %s preset: %s\n", strings.ToUpper(discType), defaultKey)
			}
		}
	}

	// Validate parameters
	if tr0 <= 0 || dtr <= 0 || r0 <= 0 {
		return fmt.Errorf("invalid parameters: tr0=%.2f, dtr=%.6f, r0=%.1f (all must be > 0)", tr0, dtr, r0)
	}

	fmt.Printf("Visualization parameters:\n")
	fmt.Printf("  Track file: %s\n", trackFile)
	fmt.Printf("  Output image: %s\n", outputImage)
	fmt.Printf("  Disc type: %s\n", strings.ToUpper(discType))
	fmt.Printf("  TR0: %s\n", formatFloat(tr0))
	fmt.Printf("  DTR: %s\n", formatFloat(dtr))
	fmt.Printf("  R0: %s\n", formatFloat(r0))
	fmt.Printf("\n")

	// Create visualizer and generate the image
	visualizer := NewTrackVisualizer(tr0, dtr, r0, discType)
	
	fmt.Println("Reading track data and creating visualization...")
	fmt.Println("This may take a few minutes for large tracks...")
	
	err := visualizer.VisualizeTrack(trackFile, outputImage)
	if err != nil {
		return fmt.Errorf("visualization failed: %w", err)
	}

	fmt.Printf("\n✓ Visualization completed successfully!\n")
	fmt.Printf("✓ Open %s to see how your track will look on the disc\n", outputImage)
	
	return nil
}

// formatFloat formats a float64 with appropriate precision
func formatFloat(f float64) string {
	if f > 1000 {
		return fmt.Sprintf("%.0f", f)
	} else if f > 10 {
		return fmt.Sprintf("%.2f", f)
	} else {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
}
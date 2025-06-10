package main

import (
	"fmt"
	"strings"
)

// DiscPreset represents the parameters for a specific disc type
type DiscPreset struct {
	Name     string
	DiscType string // "cd" or "dvd"
	Tr0      float64
	Dtr      float64
	R0       float64
}

// GetPresets returns all available disc presets
func GetPresets() map[string]DiscPreset {
	return map[string]DiscPreset{
		// CD presets from original application
		"verbatim-cd-rw-1": {
			Name:     "Verbatim CD-RW Hi-Speed 8x-10x 700 MB SERL 1",
			DiscType: "cd",
			Tr0:      22951.52,
			Dtr:      1.3865961,
			R0:       24.5,
		},
		"verbatim-cd-rw-2": {
			Name:     "Verbatim CD-RW Hi-Speed 8x-10x 700 MB SERL 2",
			DiscType: "cd",
			Tr0:      22951.07,
			Dtr:      1.3865958,
			R0:       24.5,
		},
		"eperformance-cd-rw": {
			Name:     "eProformance CD-RW 4x-10x 700 MB Prodisk Technology Inc",
			DiscType: "cd",
			Tr0:      22936.085,
			Dtr:      1.38314,
			R0:       24.5,
		},
		"tdk-cd-rw": {
			Name:     "TDK CD-RW 4x-12x HIGH SPEED 700MB 80MIN",
			DiscType: "cd",
			Tr0:      23000.145,
			Dtr:      1.38659775,
			R0:       24.5,
		},
		// DVD presets - these are estimated based on DVD specifications
		// DVD has different geometry - larger data area and different track spacing
		"generic-dvd-r": {
			Name:     "Generic DVD-R 4.7GB",
			DiscType: "dvd",
			Tr0:      48000.0,   // Higher initial track count for DVD
			Dtr:      0.74,      // Tighter track spacing for DVD
			R0:       24.0,      // Inner radius similar to CD
		},
		"generic-dvd-rw": {
			Name:     "Generic DVD-RW 4.7GB",
			DiscType: "dvd",
			Tr0:      48050.0,
			Dtr:      0.741,
			R0:       24.0,
		},
		"verbatim-dvd-r": {
			Name:     "Verbatim DVD-R 16x 4.7GB",
			DiscType: "dvd",
			Tr0:      47980.0,
			Dtr:      0.739,
			R0:       24.0,
		},
		"sony-dvd-rw": {
			Name:     "Sony DVD-RW 4x 4.7GB",
			DiscType: "dvd",
			Tr0:      48100.0,
			Dtr:      0.742,
			R0:       24.0,
		},
	}
}

// GetPresetByName returns a preset by its key name
func GetPresetByName(name string) (DiscPreset, bool) {
	presets := GetPresets()
	preset, exists := presets[name]
	return preset, exists
}

// GetDefaultPreset returns the default preset for a disc type
func GetDefaultPreset(discType string) DiscPreset {
	switch strings.ToLower(discType) {
	case "dvd":
		return GetPresets()["generic-dvd-r"]
	case "cd":
		fallthrough
	default:
		return GetPresets()["verbatim-cd-rw-1"]
	}
}

// listPresets prints all available presets
func listPresets() {
	presets := GetPresets()
	
	fmt.Println("Available disc presets:")
	fmt.Println()
	
	// Group by disc type
	cdPresets := make([]string, 0)
	dvdPresets := make([]string, 0)
	
	for key, preset := range presets {
		if preset.DiscType == "cd" {
			cdPresets = append(cdPresets, key)
		} else if preset.DiscType == "dvd" {
			dvdPresets = append(dvdPresets, key)
		}
	}
	
	if len(cdPresets) > 0 {
		fmt.Println("CD Presets:")
		for _, key := range cdPresets {
			preset := presets[key]
			fmt.Printf("  %-20s - %s (tr0=%.2f, dtr=%.6f, r0=%.1f)\n",
				key, preset.Name, preset.Tr0, preset.Dtr, preset.R0)
		}
		fmt.Println()
	}
	
	if len(dvdPresets) > 0 {
		fmt.Println("DVD Presets:")
		for _, key := range dvdPresets {
			preset := presets[key]
			fmt.Printf("  %-20s - %s (tr0=%.2f, dtr=%.6f, r0=%.1f)\n",
				key, preset.Name, preset.Tr0, preset.Dtr, preset.R0)
		}
		fmt.Println()
	}
	
	fmt.Println("Usage: cdimage burn -i image.jpg -p preset-name")
}
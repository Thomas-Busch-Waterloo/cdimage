// CDImage - A tool for burning visible pictures on a compact disc/DVD surface
// Copyright (C) 2008-2022 arduinocelentano, 2025 Go port
//
// This program is free software: you can redistribute it and/or modify it under the terms
// of the GNU General Public License as published by the Free Software Foundation,
// either version 3 of the License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
// without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
// See the GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along with this program.
// If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "1.0.0"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "cdimage",
		Short: "A tool for burning visible pictures on a CD/DVD surface",
		Long: `CDImage burns visible pictures onto CD and DVD surfaces by converting images
to audio tracks that create patterns when burned. Supports both CD and DVD formats.`,
		Version: version,
	}

	// Add subcommands
	rootCmd.AddCommand(createBurnCmd())
	rootCmd.AddCommand(createListPresetsCmd())
	rootCmd.AddCommand(createGUICmd())
	rootCmd.AddCommand(createVisualizeCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func createBurnCmd() *cobra.Command {
	var (
		inputFile      string
		outputFile     string
		discType       string
		tr0            float64
		dtr            float64
		r0             float64
		mixColors      bool
		preset         string
		useMultithread bool
	)

	cmd := &cobra.Command{
		Use:   "burn",
		Short: "Convert image to burnable audio track",
		Long: `Convert an image file to an audio track that can be burned onto a CD or DVD
to create a visible pattern on the disc surface.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return burnImage(inputFile, outputFile, discType, tr0, dtr, r0, mixColors, preset, useMultithread)
		},
	}

	cmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input image file (required)")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "track.raw", "Output audio track file")
	cmd.Flags().StringVarP(&discType, "type", "t", "cd", "Disc type: cd or dvd")
	cmd.Flags().Float64Var(&tr0, "tr0", 0, "Initial track parameter (use preset if 0)")
	cmd.Flags().Float64Var(&dtr, "dtr", 0, "Track delta parameter (use preset if 0)")
	cmd.Flags().Float64Var(&r0, "r0", 24.5, "Initial radius parameter")
	cmd.Flags().BoolVar(&mixColors, "mix-colors", false, "Use random color mixing")
	cmd.Flags().StringVarP(&preset, "preset", "p", "", "Use disc preset (see list-presets)")
	cmd.Flags().BoolVarP(&useMultithread, "parallel", "j", true, "Use multi-threaded conversion (default: true)")

	cmd.MarkFlagRequired("input")

	return cmd
}

func createListPresetsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-presets",
		Short: "List available disc presets",
		Long:  "List all available disc presets with their parameters",
		Run: func(cmd *cobra.Command, args []string) {
			listPresets()
		},
	}
}

func createGUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gui",
		Short: "Launch the graphical user interface",
		Long:  "Launch the CDImage GUI application for interactive image conversion",
		Run: func(cmd *cobra.Command, args []string) {
			runGUI()
		},
	}
}

func createVisualizeCmd() *cobra.Command {
	var (
		trackFile   string
		outputImage string
		discType    string
		tr0         float64
		dtr         float64
		r0          float64
		preset      string
	)

	cmd := &cobra.Command{
		Use:   "visualize",
		Short: "Visualize how a raw track will look on disc",
		Long: `Create a PNG image showing how the raw audio track will appear when
burned onto a CD or DVD surface. This lets you preview the result without
wasting blank discs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return visualizeTrack(trackFile, outputImage, discType, tr0, dtr, r0, preset)
		},
	}

	cmd.Flags().StringVarP(&trackFile, "track", "t", "", "Raw track file to visualize (required)")
	cmd.Flags().StringVarP(&outputImage, "output", "o", "disc_preview.png", "Output PNG image file")
	cmd.Flags().StringVarP(&discType, "type", "d", "cd", "Disc type: cd or dvd")
	cmd.Flags().Float64Var(&tr0, "tr0", 0, "Initial track parameter (use preset if 0)")
	cmd.Flags().Float64Var(&dtr, "dtr", 0, "Track delta parameter (use preset if 0)")
	cmd.Flags().Float64Var(&r0, "r0", 24.5, "Initial radius parameter")
	cmd.Flags().StringVarP(&preset, "preset", "p", "", "Use disc preset (see list-presets)")

	cmd.MarkFlagRequired("track")

	return cmd
}
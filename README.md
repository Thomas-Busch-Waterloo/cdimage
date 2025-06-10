# CDImage - Go Edition

A Go rewrite of the CDImage tool for burning visible pictures on CD and DVD surfaces.

## Overview

CDImage converts images into audio tracks that create visible patterns when burned onto optical discs. This Go version extends the original Qt application with DVD support and a command-line interface optimized for Linux.

## Features

- **CD Support**: All original CD functionality with predefined disc presets
- **DVD Support**: NEW - Extended geometry parameters for DVD-R and DVD-RW discs
- **Multi-threaded Processing**: NEW - Parallel conversion using multiple CPU cores for faster processing
- **Modern Progress Bar**: NEW - Clean progress visualization without terminal clutter
- **Interactive GUI**: NEW - Full graphical interface with real-time disc visualization and image positioning
- **Optical Drive Detection**: NEW - Automatic detection of CD/DVD burners with burning capabilities  
- **Direct Burning**: NEW - Burn directly from the GUI to your optical drive
- **CLI Interface**: Command-line tool optimized for Linux workflows
- **Progress Tracking**: Real-time conversion progress with cancellation support
- **Multiple Image Formats**: Support for JPEG, PNG, and other common formats
- **Preset Management**: Built-in presets for various disc brands and types

## Installation

### Prerequisites

```bash
# Install Go 1.21 or later
# On Ubuntu/Debian:
sudo apt install golang-go

# On Arch/Manjaro:
sudo pacman -S go

# On CentOS/RHEL/Fedora:
sudo dnf install golang

# For GUI support, install additional dependencies:
# Ubuntu/Debian:
sudo apt-get install libgl1-mesa-dev xorg-dev

# Fedora:
sudo dnf install mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel

# Arch:
sudo pacman -S libgl libxcursor libxrandr libxinerama libxi
```

### Build from Source

```bash
# Clone or download the source
cd cdimage
go mod download
go build -o cdimage .
```

## Usage

### GUI Mode (Recommended)

```bash
# Launch the graphical interface
./cdimage gui
```

The GUI provides:
- **Interactive Disc Preview**: Real-time visualization of how your image will appear on the disc
- **Image Positioning**: Drag, zoom (scroll wheel), and double-click to center
- **Visual Parameter Controls**: Easy preset selection and parameter adjustment
- **Optical Drive Detection**: Automatic detection of available CD/DVD burners
- **Direct Burning**: Convert and burn in one workflow with drive capability checking
- **Real-time Progress**: Visual progress bars for both conversion and burning
- **File Browser**: Easy image loading with format filtering

### CLI Mode

```bash
# Convert image for CD using default preset
./cdimage burn -i image.jpg -o track.raw

# Convert for DVD with specific preset
./cdimage burn -i image.jpg -o dvd_track.raw -t dvd -p generic-dvd-r

# Use multi-threading (default) for faster conversion
./cdimage burn -i image.jpg -o track.raw -j

# Disable multi-threading for single-core processing
./cdimage burn -i image.jpg -o track.raw -j=false

# Use custom parameters
./cdimage burn -i image.jpg -o track.raw --tr0 23000 --dtr 1.386 --r0 24.5
```

### Available Commands

```bash
# Launch GUI
./cdimage gui

# List all available presets
./cdimage list-presets

# Convert image with progress bar
./cdimage burn -i photo.jpg -o output.raw -p verbatim-cd-rw-1

# DVD conversion with color mixing and parallel processing
./cdimage burn -i artwork.png -o dvd.raw -t dvd --mix-colors -j
```

### Command Options

- `-i, --input`: Input image file (required)
- `-o, --output`: Output audio track file (default: track.raw)
- `-t, --type`: Disc type - "cd" or "dvd" (default: cd)
- `-p, --preset`: Use predefined disc preset
- `-j, --parallel`: Enable multi-threaded conversion (default: true)
- `--tr0`: Initial track parameter (overrides preset)
- `--dtr`: Track delta parameter (overrides preset)
- `--r0`: Initial radius parameter (default: 24.5)
- `--mix-colors`: Enable random color mixing

## Burning the Track

After conversion, burn the audio track to your disc:

### For CDs:
```bash
# Using cdrecord
cdrecord -audio dev=/dev/sr0 track.raw

# Using wodim
wodim -audio dev=/dev/sr0 track.raw
```

### For DVDs:
```bash
# Using growisofs
growisofs -audio -Z /dev/sr0=dvd_track.raw

# Or with cdrecord (some drives)
cdrecord -audio dev=/dev/sr0 dvd_track.raw
```

### Find Your Drive:
```bash
cdrecord -scanbus
# or
wodim -scanbus
```

## Disc Presets

### CD Presets:
- `verbatim-cd-rw-1`: Verbatim CD-RW Hi-Speed 8x-10x 700 MB SERL 1
- `verbatim-cd-rw-2`: Verbatim CD-RW Hi-Speed 8x-10x 700 MB SERL 2
- `eperformance-cd-rw`: eProformance CD-RW 4x-10x 700 MB
- `tdk-cd-rw`: TDK CD-RW 4x-12x HIGH SPEED 700MB

### DVD Presets:
- `generic-dvd-r`: Generic DVD-R 4.7GB (recommended starting point)
- `generic-dvd-rw`: Generic DVD-RW 4.7GB
- `verbatim-dvd-r`: Verbatim DVD-R 16x 4.7GB
- `sony-dvd-rw`: Sony DVD-RW 4x 4.7GB

## DVD Support Details

This Go version adds comprehensive DVD support with:

- **Larger Data Capacity**: DVDs support ~4.7GB vs ~800MB for CDs
- **Different Geometry**: Optimized track spacing and parameters for DVD format
- **Extended Burn Area**: Larger usable surface area for images
- **DVD-Specific Presets**: Calibrated parameters for common DVD brands

## Technical Notes

### Image Processing
- Images are automatically converted to grayscale
- Automatically scaled to fit disc geometry
- Supports high-contrast images for best results

### Performance
- **Multi-threaded Processing**: Utilizes all CPU cores for faster conversion
- **Smart Progress Tracking**: Clean progress bars without terminal clutter
- **Graceful Cancellation**: Ctrl+C support in CLI, Cancel button in GUI
- **Memory-efficient Streaming**: Handles large files without excessive RAM usage
- **Optimized for DVD**: Up to 6x larger capacity than CD, optimized processing

### File Formats
- Output: Raw audio track (.raw)
- Input: JPEG, PNG, and other formats supported by Go imaging library

## Calibration for Unknown Discs

If your disc isn't in the presets:

1. **Start with a similar preset** (same type: CD/DVD, same brand if available)
2. **Make small parameter adjustments**:
   - `tr0`: Initial track count (higher = more tracks)
   - `dtr`: Track spacing increment (lower = tighter spacing)
   - `r0`: Inner radius (usually 24.0-24.5)
3. **Test burn on a rewritable disc** first
4. **Adjust based on results** and re-burn

## Troubleshooting

### Common Issues:

**"Permission denied" when burning:**
```bash
sudo chmod 666 /dev/sr0
# or run burning command with sudo
```

**No image visible after burning:**
- Try a different disc preset
- Ensure high contrast in source image
- Check if your drive supports the burning mode
- Use a CD-RW/DVD-RW for testing

**Conversion takes too long:**
- DVD conversion is significantly longer than CD (~6x more data)  
- Use Ctrl+C to cancel if needed (CLI) or Cancel button (GUI)
- Consider using a smaller/simpler image
- Enable multi-threading with `-j` flag (CLI) or checkbox (GUI)

**GUI won't start:**
- Install GUI dependencies: `make gui-deps`
- Check that X11/Wayland display is available
- Try running from terminal to see error messages

**No optical drives detected:**
- Check that drives are connected and powered on
- Ensure user has permission to access optical devices
- Try refreshing drives in GUI or restart application
- Verify drives work with other burning software

**Burning fails:**
- Ensure disc is blank and compatible with drive
- Check that burning tools are installed (`cdrecord`, `wodim`, or `growisofs`)
- Verify user has permission to access optical devices
- Try different disc brand or speed

## License

GNU General Public License v3.0 - see LICENSE file.

Original Qt implementation copyright (C) 2008-2022 arduinocelentano
Go port additions copyright (C) 2025

## Credits

- Original algorithm and implementation by arduinocelentano
- Based on work by [unDEFER] and [argon] (Instructables)
- Go port with DVD support and CLI interface

## Contributing

Issues and improvements welcome. When adding new disc presets, please include:
- Exact disc model and manufacturer
- Verified working parameters (tr0, dtr, r0)
- Sample burn results if possible
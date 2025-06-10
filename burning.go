package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// OpticalDrive represents an optical drive that can burn discs
type OpticalDrive struct {
	Device      string // Device path like /dev/sr0
	Name        string // Human readable name
	Vendor      string
	Model       string
	CanBurnCD   bool
	CanBurnDVD  bool
	IsReady     bool
}

// DetectOpticalDrives finds all available optical drives on the system
func DetectOpticalDrives() ([]OpticalDrive, error) {
	var drives []OpticalDrive
	
	// Method 1: Check /proc/sys/dev/cdrom/info
	procDrives, err := detectFromProc()
	if err == nil && len(procDrives) > 0 {
		drives = append(drives, procDrives...)
	}
	
	// Method 2: Use lsblk to find optical devices
	lsblkDrives, err := detectFromLsblk()
	if err == nil {
		// Merge with proc results or use if proc failed
		if len(drives) == 0 {
			drives = lsblkDrives
		} else {
			// Enhance existing drives with lsblk info
			for i := range drives {
				for _, lsblkDrive := range lsblkDrives {
					if drives[i].Device == lsblkDrive.Device {
						if drives[i].Name == "" {
							drives[i].Name = lsblkDrive.Name
						}
						break
					}
				}
			}
		}
	}
	
	// Method 3: Fallback - check common device paths
	if len(drives) == 0 {
		drives = detectFromDevices()
	}
	
	// Enhance drive info with cdrecord/wodim scan
	enhanceDrivesWithCdrecord(&drives)
	
	return drives, nil
}

// detectFromProc reads optical drive info from /proc/sys/dev/cdrom/info
func detectFromProc() ([]OpticalDrive, error) {
	file, err := os.Open("/proc/sys/dev/cdrom/info")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var drives []OpticalDrive
	var driveNames []string
	var canWriteCD []bool
	var canWriteDVD []bool
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if strings.HasPrefix(line, "drive name:") {
			names := strings.Fields(line)[2:] // Skip "drive name:"
			for _, name := range names {
				driveNames = append(driveNames, "/dev/"+name)
			}
		} else if strings.HasPrefix(line, "Can write CD-R:") {
			values := strings.Fields(line)[3:] // Skip "Can write CD-R:"
			for _, val := range values {
				canWriteCD = append(canWriteCD, val == "1")
			}
		} else if strings.HasPrefix(line, "Can write DVD-R:") {
			values := strings.Fields(line)[3:] // Skip "Can write DVD-R:"
			for _, val := range values {
				canWriteDVD = append(canWriteDVD, val == "1")
			}
		}
	}
	
	// Combine the information
	for i, name := range driveNames {
		drive := OpticalDrive{
			Device:  name,
			Name:    filepath.Base(name),
			IsReady: true,
		}
		
		if i < len(canWriteCD) {
			drive.CanBurnCD = canWriteCD[i]
		}
		if i < len(canWriteDVD) {
			drive.CanBurnDVD = canWriteDVD[i]
		}
		
		drives = append(drives, drive)
	}
	
	return drives, scanner.Err()
}

// detectFromLsblk uses lsblk to find optical devices
func detectFromLsblk() ([]OpticalDrive, error) {
	cmd := exec.Command("lsblk", "-d", "-n", "-o", "NAME,TYPE,VENDOR,MODEL", "/dev/sr*")
	output, err := cmd.Output()
	if err != nil {
		// Try alternative approach
		cmd = exec.Command("lsblk", "-d", "-n", "-o", "NAME,TYPE,VENDOR,MODEL")
		output, err = cmd.Output()
		if err != nil {
			return nil, err
		}
	}
	
	var drives []OpticalDrive
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		
		if len(fields) >= 2 && fields[1] == "rom" {
			drive := OpticalDrive{
				Device:  "/dev/" + fields[0],
				Name:    fields[0],
				IsReady: true,
			}
			
			if len(fields) > 2 {
				drive.Vendor = fields[2]
			}
			if len(fields) > 3 {
				drive.Model = strings.Join(fields[3:], " ")
			}
			
			drives = append(drives, drive)
		}
	}
	
	return drives, scanner.Err()
}

// detectFromDevices checks common device paths
func detectFromDevices() []OpticalDrive {
	var drives []OpticalDrive
	
	// Common optical drive device paths
	devicePaths := []string{
		"/dev/sr0", "/dev/sr1", "/dev/sr2", "/dev/sr3",
		"/dev/cdrom", "/dev/dvd", "/dev/cdrw", "/dev/dvdrw",
	}
	
	for _, device := range devicePaths {
		if _, err := os.Stat(device); err == nil {
			drives = append(drives, OpticalDrive{
				Device:  device,
				Name:    filepath.Base(device),
				IsReady: true,
			})
		}
	}
	
	return drives
}

// enhanceDrivesWithCdrecord uses cdrecord/wodim to get additional drive info
func enhanceDrivesWithCdrecord(drives *[]OpticalDrive) {
	// Try cdrecord first, then wodim
	tools := []string{"cdrecord", "wodim"}
	
	for _, tool := range tools {
		if enhanceWithTool(drives, tool) {
			break
		}
	}
}

// enhanceWithTool enhances drive info using a specific burning tool
func enhanceWithTool(drives *[]OpticalDrive, tool string) bool {
	cmd := exec.Command(tool, "-scanbus")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	// Parse cdrecord/wodim output
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	re := regexp.MustCompile(`(\d+,\d+,\d+)\s+\d+\)\s+'([^']+)'\s+'([^']+)'`)
	
	deviceIndex := 0
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		
		if len(matches) >= 4 && deviceIndex < len(*drives) {
			(*drives)[deviceIndex].Vendor = strings.TrimSpace(matches[2])
			(*drives)[deviceIndex].Model = strings.TrimSpace(matches[3])
			
			// Assume modern drives can burn both CD and DVD
			(*drives)[deviceIndex].CanBurnCD = true
			(*drives)[deviceIndex].CanBurnDVD = true
			
			deviceIndex++
		}
	}
	
	return true
}

// BurnAudioTrack burns an audio track to the specified drive
func BurnAudioTrack(drive OpticalDrive, trackFile string, discType string) error {
	// Determine the appropriate burning tool and command
	var cmd *exec.Cmd
	
	// Try different burning tools in order of preference
	if _, err := exec.LookPath("cdrecord"); err == nil {
		cmd = exec.Command("cdrecord", "-audio", fmt.Sprintf("dev=%s", drive.Device), trackFile)
	} else if _, err := exec.LookPath("wodim"); err == nil {
		cmd = exec.Command("wodim", "-audio", fmt.Sprintf("dev=%s", drive.Device), trackFile)
	} else if _, err := exec.LookPath("growisofs"); err == nil && discType == "dvd" {
		cmd = exec.Command("growisofs", "-audio", fmt.Sprintf("-Z %s=%s", drive.Device, trackFile))
	} else {
		return fmt.Errorf("no suitable burning tool found (cdrecord, wodim, or growisofs)")
	}
	
	// Set up command to show output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

// CheckDiscInDrive checks if there's a writable disc in the drive
func CheckDiscInDrive(drive OpticalDrive) (bool, string, error) {
	// Use blkid to get disc info
	cmd := exec.Command("blkid", "-p", drive.Device)
	output, err := cmd.Output()
	
	if err != nil {
		// No disc or unreadable disc
		return false, "No disc detected", nil
	}
	
	outputStr := string(output)
	
	// Check for different disc types
	if strings.Contains(outputStr, "iso9660") {
		return false, "Data disc detected (not blank)", nil
	}
	
	// For blank discs, blkid might fail, so we try other methods
	cmd = exec.Command("dd", "if="+drive.Device, "bs=1", "count=1")
	err = cmd.Run()
	
	if err != nil {
		return true, "Blank disc ready", nil
	}
	
	return false, "Disc present but not blank", nil
}

// GetBurningCommand returns the command line that would be used for burning
func GetBurningCommand(drive OpticalDrive, trackFile string, discType string) string {
	if _, err := exec.LookPath("cdrecord"); err == nil {
		return fmt.Sprintf("cdrecord -audio dev=%s %s", drive.Device, trackFile)
	} else if _, err := exec.LookPath("wodim"); err == nil {
		return fmt.Sprintf("wodim -audio dev=%s %s", drive.Device, trackFile)
	} else if _, err := exec.LookPath("growisofs"); err == nil && discType == "dvd" {
		return fmt.Sprintf("growisofs -audio -Z %s=%s", drive.Device, trackFile)
	}
	
	return "No burning tool available"
}
package main

import (
	"context"
	"fmt"
	"image"
	"math"
	"math/rand"
	"os"
	"runtime"
	"sync"
)

// TrackJob represents a single track processing job
type TrackJob struct {
	trackIndex int
	tr         float64
	r          float64
	rcd        float64
	cx, cy     float64
	ir         float64
	itr        int
	zs, zf     int
}

// TrackResult holds the result of processing a track
type TrackResult struct {
	trackIndex int
	data       []byte
	err        error
}

// MultiThreadedConverter extends the basic converter with parallel processing
type MultiThreadedConverter struct {
	*Converter
	numWorkers int
	jobs       chan TrackJob
	results    chan TrackResult
	wg         sync.WaitGroup
}

// NewMultiThreadedConverter creates a new multi-threaded converter
func NewMultiThreadedConverter(tr0, dtr, r0 float64, mixColors bool, discType string) *MultiThreadedConverter {
	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8 // Cap at 8 to avoid memory issues
	}
	
	return &MultiThreadedConverter{
		Converter:  NewConverter(tr0, dtr, r0, mixColors, discType),
		numWorkers: numWorkers,
		jobs:       make(chan TrackJob, numWorkers*2),
		results:    make(chan TrackResult, numWorkers*2),
	}
}

// ConvertParallel converts an image using multiple goroutines for track processing
func (mtconv *MultiThreadedConverter) ConvertParallel(ctx context.Context, img image.Image, filename string) error {
	// Determine total size based on disc type
	totalSize := CDTotalSize
	if mtconv.discType == "dvd" {
		totalSize = DVDTotalSize
	}
	
	// Create output file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	// Convert image bounds
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()
	
	// Initialize variables
	tr := mtconv.tr0
	r := mtconv.r0
	dr := mtconv.dtr * mtconv.r0 / mtconv.tr0
	c := 0.0
	
	// Disc geometry constants
	ir := 1500.0     // Image radius
	rcd := 57.5      // CD radius
	cx := float64(imgWidth) / 2
	cy := float64(imgHeight) / 2
	
	zs := 0
	zf := 0
	trackIndex := 0
	
	// Start worker goroutines
	for i := 0; i < mtconv.numWorkers; i++ {
		mtconv.wg.Add(1)
		go mtconv.trackWorker(ctx, img, imgWidth, imgHeight)
	}
	
	// Track buffer to maintain order
	trackBuffer := make(map[int][]byte)
	nextTrackToWrite := 0
	
	// Start result collector
	resultsDone := make(chan bool)
	go func() {
		defer close(resultsDone)
		
		for result := range mtconv.results {
			if result.err != nil {
				// Handle error (could store first error and continue or abort)
				continue
			}
			
			// Buffer the track data
			trackBuffer[result.trackIndex] = result.data
			
			// Write sequential tracks to file
			for {
				if data, exists := trackBuffer[nextTrackToWrite]; exists {
					if _, err := file.Write(data); err != nil {
						// Handle write error
						break
					}
					delete(trackBuffer, nextTrackToWrite)
					nextTrackToWrite++
				} else {
					break
				}
			}
		}
	}()
	
	// Generate jobs for tracks
	jobsDone := make(chan bool)
	go func() {
		defer close(jobsDone)
		defer close(mtconv.jobs)
		
		for c < float64(totalSize)-tr {
			// Check for cancellation
			select {
			case <-ctx.Done():
				return
			default:
			}
			
			if mtconv.cancelCallback != nil && mtconv.cancelCallback() {
				return
			}
			
			// Update progress
			if mtconv.progressCallback != nil {
				progress := int(100 * c / float64(totalSize))
				mtconv.progressCallback(progress)
			}
			
			itr := int(tr)
			
			job := TrackJob{
				trackIndex: trackIndex,
				tr:         tr,
				r:          r,
				rcd:        rcd,
				cx:         cx,
				cy:         cy,
				ir:         ir,
				itr:        itr,
				zs:         zs,
				zf:         zf,
			}
			
			select {
			case mtconv.jobs <- job:
			case <-ctx.Done():
				return
			}
			
			c += tr
			tr += mtconv.dtr
			r += dr
			trackIndex++
			
			zs++
			if zs >= 17 {
				zs = 0
			}
		}
	}()
	
	// Wait for job generation to complete
	<-jobsDone
	
	// Wait for all workers to finish
	mtconv.wg.Wait()
	close(mtconv.results)
	
	// Wait for result collection to complete
	<-resultsDone
	
	return nil
}

// trackWorker processes individual tracks in parallel
func (mtconv *MultiThreadedConverter) trackWorker(ctx context.Context, img image.Image, imgWidth, imgHeight int) {
	defer mtconv.wg.Done()
	
	for {
		select {
		case job, ok := <-mtconv.jobs:
			if !ok {
				return // Channel closed, worker done
			}
			
			data, err := mtconv.processTrack(ctx, img, imgWidth, imgHeight, job)
			
			select {
			case mtconv.results <- TrackResult{
				trackIndex: job.trackIndex,
				data:       data,
				err:        err,
			}:
			case <-ctx.Done():
				return
			}
			
		case <-ctx.Done():
			return
		}
	}
}

// processTrack processes a single track and returns the audio data
func (mtconv *MultiThreadedConverter) processTrack(ctx context.Context, img image.Image, imgWidth, imgHeight int, job TrackJob) ([]byte, error) {
	ri := job.ir * job.r / job.rcd
	trackData := make([]byte, 0, job.itr*4) // Estimate capacity
	
	localZf := job.zf
	
	// Process one track
	for i := 0; i < job.itr; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		alpha := 2 * math.Pi * float64(i) / float64(job.itr)
		xi := job.cx + ri*math.Cos(alpha)
		yi := job.cy + ri*math.Sin(alpha)
		
		// Sample the image
		pixelColor := mtconv.sampleImage(img, int(xi), int(yi), imgWidth, imgHeight)
		grayValue := mtconv.rgbaToGray(pixelColor)
		
		c1 := grayValue / 85
		c2 := c1 + 1
		if c2 > 3 {
			c2 = 3
		}
		
		var cl byte
		grayMod := int(grayValue % 85)
		if mtconv.mixColors {
			if rand.Intn(85) < grayMod || grayMod == 84 {
				cl = c2
			} else {
				cl = c1
			}
		} else {
			if grayMod > (job.zs*5+localZf) || grayMod == 84 {
				cl = c2
			} else {
				cl = c1
			}
		}
		
		// For now, just append the palette byte directly
		// In a real implementation, we'd need to handle the delay sequence
		trackData = append(trackData, palette[cl])
		
		localZf++
		if localZf >= 5 {
			localZf = 0
		}
	}
	
	return trackData, nil
}

// SetNumWorkers allows customizing the number of worker goroutines
func (mtconv *MultiThreadedConverter) SetNumWorkers(numWorkers int) {
	if numWorkers > 0 && numWorkers <= 16 {
		mtconv.numWorkers = numWorkers
	}
}
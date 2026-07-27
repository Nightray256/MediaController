package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.NewWithID("iRay")
	w := a.NewWindow("Video Compressor")
	w.Resize(fyne.NewSize(680, 480))

	var selectedFilePath string

	fileLabel := widget.NewLabel("No file selected")
	sizeEntry := widget.NewEntry()
	sizeEntry.SetPlaceHolder("Enter target size in MB")
	statusLabel := widget.NewLabel("Status: Ready")

	selectBtn := widget.NewButton("Select Video File", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			selectedFilePath = reader.URI().Path()
			fileLabel.SetText(fmt.Sprintf("Selected File: %s", filepath.Base(selectedFilePath)))
		}, w)
	})

	compressBtn := widget.NewButton("Compress Video", func() {
		if selectedFilePath == "" {
			dialog.ShowInformation("No file selected", "Please select a video file first.", w)
			return
		}
		targetSize := sizeEntry.Text
		if targetSize == "" {
			dialog.ShowInformation("Missing target size", "Please enter a target size in MB.", w)
			return
		}

		statusLabel.SetText("Status: Compressing... , It may take a few minutes.")

		ext := filepath.Ext(selectedFilePath)
		base := selectedFilePath[0 : len(selectedFilePath)-len(ext)]
		outputPath := fmt.Sprintf("%s_compressed%s", base, ext)

		go func() {
			err := compressVideo(selectedFilePath, outputPath, targetSize)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Status: Compression failed: (%v)", err))
				dialog.ShowError(err, w)
			} else {
				statusLabel.SetText("Status: Compression completed.")
				dialog.ShowInformation("Compression completed", fmt.Sprintf("Compressed video \nsaved to: %s", outputPath), w)
			}
		}()
	})

	content := container.NewVBox(
		widget.NewLabel("Choose original video"),
		selectBtn,
		fileLabel,
		widget.NewLabel(""),

		widget.NewLabel("Enter target size in MB"),
		sizeEntry,
		widget.NewLabel(""),

		compressBtn,
		statusLabel,
	)

	w.SetContent(content)
	w.ShowAndRun()
}

func compressVideo(inputPath, outputPath, targetSizeStr string) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	exeDir := filepath.Dir(exePath)
	ffmpegPath := filepath.Join(exeDir, "ffmpeg.exe")
	ffprobePath := filepath.Join(exeDir, "ffprobe.exe")

	if _, err := os.Stat(ffmpegPath); os.IsNotExist(err) {
		return fmt.Errorf("ffmpeg executable not found in the current directory")
	}
	if _, err := os.Stat(ffprobePath); os.IsNotExist(err) {
		return fmt.Errorf("ffprobe executable not found in the current directory")
	}

	targetSizeMB, err := strconv.ParseFloat(targetSizeStr, 64)
	if err != nil || targetSizeMB <= 0 {
		return fmt.Errorf("invalid target size: %v", err)
	}

	probeCmd := exec.Command(ffprobePath, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", inputPath)
	probeOutput, err := probeCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get video duration: %v", err)
	}

	durationStr := strings.TrimSpace(string(probeOutput))
	durationSec, err := strconv.ParseFloat(durationStr, 64)
	if err != nil || durationSec <= 0 {
		return fmt.Errorf("failed to parse video duration: %v", err)
	}

	audioBitrate := 128.0
	totalBitrate := (targetSizeMB * 8192) / durationSec
	videoBitrate := totalBitrate - audioBitrate

	if videoBitrate < 100 {
		videoBitrate = 100
	}

	videoBitrateArg := fmt.Sprintf("%.0fk", videoBitrate)
	audioBitrateArg := fmt.Sprintf("%.0fk", audioBitrate)

	cmd := exec.Command(ffmpegPath, "-i", inputPath, "-b:v", videoBitrateArg, "-b:a", audioBitrateArg, "-y", outputPath)

	err = cmd.Run()
	return err
}

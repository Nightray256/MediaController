package main

import (
	"fmt"
	"path/filepath"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

func buildVideoUI(w fyne.Window) fyne.CanvasObject {
	var selectedVideoPath string

	videoFileLabel := widget.NewLabel("No file selected")
	sizeEntry := widget.NewEntry()
	sizeEntry.SetPlaceHolder("Enter target size in MB")
	videoStatusLabel := widget.NewLabel("Status: Ready")

	videoProgressBar := widget.NewProgressBar()
	videoProgressBar.SetValue(0.0)

	selectVideoBtn := widget.NewButton("Select Video File", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			selectedVideoPath = reader.URI().Path()
			videoFileLabel.SetText(fmt.Sprintf("Selected File: %s", filepath.Base(selectedVideoPath)))
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".mp4", ".mov", ".avi", ".mkv"}))
		fd.Show()
	})

	compressVideoBtn := widget.NewButton("Compress Video", func() {
		if selectedVideoPath == "" {
			dialog.ShowInformation("No file selected", "Please select a video file first.", w)
			return
		}

		targetSizeStr := sizeEntry.Text
		if targetSizeStr == "" {
			dialog.ShowInformation("Missing target size", "Please enter a target size in MB.", w)
			return
		}
		if _, err := strconv.ParseFloat(targetSizeStr, 64); err != nil {
			dialog.ShowInformation("Invalid input", "Target size must be a valid number.", w)
			return
		}

		videoStatusLabel.SetText("Status: Compressing... It may take a few minutes.")
		videoProgressBar.SetValue(0.0)

		ext := filepath.Ext(selectedVideoPath)
		base := selectedVideoPath[0 : len(selectedVideoPath)-len(ext)]
		outputPath := fmt.Sprintf("%s_compressed%s", base, ext)

		go func() {
			err := compressVideo(selectedVideoPath, outputPath, targetSizeStr, func(progress float64) {
				videoProgressBar.SetValue(progress)
			})

			if err != nil {
				videoStatusLabel.SetText(fmt.Sprintf("Status: Compression failed: (%v)", err))
				dialog.ShowError(err, w)
				videoProgressBar.SetValue(0.0)
			} else {
				videoStatusLabel.SetText("Status: Compression completed.")
				successDialog := dialog.NewInformation("Compression completed", fmt.Sprintf("Compressed video \nsaved to: %s", outputPath), w)
				successDialog.SetOnClosed(func() {
					videoProgressBar.SetValue(0.0)
					sizeEntry.SetText("")
				})
				successDialog.Show()
			}
		}()
	})

	return container.NewVBox(
		widget.NewLabel("Choose original video"),
		selectVideoBtn,
		videoFileLabel,
		widget.NewLabel(""),
		widget.NewLabel("Enter target size in MB"),
		sizeEntry,
		widget.NewLabel(""),
		compressVideoBtn,
		videoProgressBar,
		videoStatusLabel,
	)
}

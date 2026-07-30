package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

func buildPhotoUI(w fyne.Window) fyne.CanvasObject {
	var selectedPhotoPath string
	var origW, origH float64
	var isUpdating bool

	photoFileLabel := widget.NewLabel("No file selected")
	photoStatusLabel := widget.NewLabel("Status: Ready")

	photoProgressBar := widget.NewProgressBar()
	photoProgressBar.SetValue(0.0)

	widthEntry := widget.NewEntry()
	widthEntry.SetPlaceHolder("Width (px)")
	heightEntry := widget.NewEntry()
	heightEntry.SetPlaceHolder("Height (px)")

	keepRatioCheck := widget.NewCheck("Keep Aspect Ratio", nil)
	keepRatioCheck.SetChecked(true)

	widthEntry.OnChanged = func(text string) {
		if !keepRatioCheck.Checked || isUpdating || origW == 0 {
			return
		}
		newW, err := strconv.ParseFloat(text, 64)
		if err == nil {
			isUpdating = true
			newH := newW / (origW / origH)
			heightEntry.SetText(fmt.Sprintf("%.0f", newH))
			isUpdating = false
		}
	}

	heightEntry.OnChanged = func(text string) {
		if !keepRatioCheck.Checked || isUpdating || origH == 0 {
			return
		}
		newH, err := strconv.ParseFloat(text, 64)
		if err == nil {
			isUpdating = true
			newW := newH * (origW / origH)
			widthEntry.SetText(fmt.Sprintf("%.0f", newW))
			isUpdating = false
		}
	}

	selectPhotoBtn := widget.NewButton("Select Photo File", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()
			selectedPhotoPath = reader.URI().Path()
			photoFileLabel.SetText(fmt.Sprintf("Selected File: %s", filepath.Base(selectedPhotoPath)))

			file, err := os.Open(selectedPhotoPath)
			if err == nil {
				defer file.Close()
				config, _, err := image.DecodeConfig(file)
				if err == nil {
					origW = float64(config.Width)
					origH = float64(config.Height)
					isUpdating = true
					widthEntry.SetText(fmt.Sprintf("%d", config.Width))
					heightEntry.SetText(fmt.Sprintf("%d", config.Height))
					isUpdating = false
				}
			}
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".jpg", ".jpeg", ".png"}))
		fd.Show()
	})

	processPhotoBtn := widget.NewButton("Process Photo", func() {
		if selectedPhotoPath == "" {
			dialog.ShowInformation("No file selected", "Please select a photo file first.", w)
			return
		}

		targetW := widthEntry.Text
		targetH := heightEntry.Text
		if targetW == "" || targetH == "" {
			dialog.ShowInformation("Missing dimensions", "Please enter both width and height.", w)
			return
		}

		photoStatusLabel.SetText("Status: Processing...")
		photoProgressBar.SetValue(0.0)

		ext := filepath.Ext(selectedPhotoPath)
		base := selectedPhotoPath[0 : len(selectedPhotoPath)-len(ext)]
		outputPath := fmt.Sprintf("%s_resized%s", base, ext)

		go func() {
			err := processPhotoFile(selectedPhotoPath, outputPath, targetW, targetH, func(progress float64) {
				photoProgressBar.SetValue(progress)
			})

			if err != nil {
				photoStatusLabel.SetText(fmt.Sprintf("Status: Processing failed: (%v)", err))
				photoProgressBar.SetValue(0.0)
				dialog.ShowError(err, w)
			} else {
				photoStatusLabel.SetText("Status: Processing completed.")
				successDialog := dialog.NewInformation("Processing completed", fmt.Sprintf("Processed photo \nsaved to: %s", outputPath), w)
				successDialog.SetOnClosed(func() {
					photoProgressBar.SetValue(0.0)
				})
				successDialog.Show()
			}
		}()
	})

	return container.NewVBox(
		widget.NewLabel("Choose original photo"),
		selectPhotoBtn,
		photoFileLabel,
		widget.NewLabel(""),
		widget.NewLabel("Enter new resolution (Width x Height):"),
		container.NewGridWithColumns(2, widthEntry, heightEntry),
		keepRatioCheck,
		processPhotoBtn,
		photoProgressBar,
		photoStatusLabel,
	)
}

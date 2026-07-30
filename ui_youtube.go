package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func buildYoutubeUI(w fyne.Window) fyne.CanvasObject {
	var downloadFolder string
	ytUrlEntry := widget.NewEntry()
	ytUrlEntry.SetPlaceHolder("Enter YouTube URL")
	ytFolderLabel := widget.NewLabel("No folder selected")
	ytStatusLabel := widget.NewLabel("Status: Ready")

	ytProgressBar := widget.NewProgressBar()
	ytProgressBar.SetValue(0.0)

	formatSelect := widget.NewSelect([]string{
		"Best Video (Auto)",
		"1080p Video",
		"720p Video",
		"480p Video",
		"Audio Only (MP3)",
	}, nil)
	formatSelect.SetSelected("Best Video (Auto)")

	selectFolderBtn := widget.NewButton("Select Download Folder", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			downloadFolder = uri.Path()
			ytFolderLabel.SetText(fmt.Sprintf("Selected Folder: %s", downloadFolder))
		}, w)
	})

	downloadBtn := widget.NewButton("Download YouTube Video", func() {
		url := ytUrlEntry.Text
		if url == "" {
			dialog.ShowInformation("Missing URL", "Please enter a YouTube URL.", w)
			return
		}
		if downloadFolder == "" {
			dialog.ShowInformation("Missing folder", "Please select a download folder.", w)
			return
		}

		ytStatusLabel.SetText("Status: Downloading...")
		ytProgressBar.SetValue(0.0)

		selectedFormat := formatSelect.Selected

		go func() {
			err := downloadYTVideo(url, downloadFolder, selectedFormat, func(progress float64) {
				ytProgressBar.SetValue(progress)
			})

			if err != nil {
				ytStatusLabel.SetText(fmt.Sprintf("Status: Download failed: (%v)", err))
				dialog.ShowError(err, w)
				ytProgressBar.SetValue(0.0)
			} else {
				ytStatusLabel.SetText("Status: Download completed.")
				successDialog := dialog.NewInformation("Download completed", "Video has been saved successfully.", w)
				successDialog.SetOnClosed(func() {
					ytProgressBar.SetValue(0.0)
					ytUrlEntry.SetText("")
				})
				successDialog.Show()
			}
		}()
	})

	return container.NewVBox(
		widget.NewLabel("Enter YouTube URL"),
		ytUrlEntry,
		widget.NewLabel(""),
		widget.NewLabel("Select Destination Folder"),
		formatSelect,
		selectFolderBtn,
		ytFolderLabel,
		widget.NewLabel(""),
		downloadBtn,
		ytProgressBar,
		ytStatusLabel,
	)
}

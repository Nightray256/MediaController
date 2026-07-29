package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

var hideWindowSysProcAttr = &syscall.SysProcAttr{HideWindow: true}

//go:embed icon.png
var iconBytes []byte

func main() {
	a := app.NewWithID("iRay")

	icon := fyne.NewStaticResource("icon", iconBytes)
	a.SetIcon(icon)

	w := a.NewWindow("Media Cotroller")
	w.Resize(fyne.NewSize(680, 480))

	var selectedVideoPath string

	videoFileLabel := widget.NewLabel("No file selected")
	sizeEntry := widget.NewEntry()
	sizeEntry.SetPlaceHolder("Enter target size in MB")
	videoStatusLabel := widget.NewLabel("Status: Ready")

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
		targetSize := sizeEntry.Text
		if targetSize == "" {
			dialog.ShowInformation("Missing target size", "Please enter a target size in MB.", w)
			return
		}

		videoStatusLabel.SetText("Status: Compressing... , It may take a few minutes.")

		ext := filepath.Ext(selectedVideoPath)
		base := selectedVideoPath[0 : len(selectedVideoPath)-len(ext)]
		outputPath := fmt.Sprintf("%s_compressed%s", base, ext)

		go func() {
			err := compressVideo(selectedVideoPath, outputPath, targetSize)
			if err != nil {
				videoStatusLabel.SetText(fmt.Sprintf("Status: Compression failed: (%v)", err))
				dialog.ShowError(err, w)
			} else {
				videoStatusLabel.SetText("Status: Compression completed.")
				dialog.ShowInformation("Compression completed", fmt.Sprintf("Compressed video \nsaved to: %s", outputPath), w)
			}
		}()
	})

	videoUI := container.NewVBox(
		widget.NewLabel("Choose original video"),
		selectVideoBtn,
		videoFileLabel,
		widget.NewLabel(""),

		widget.NewLabel("Enter target size in MB"),
		sizeEntry,
		widget.NewLabel(""),

		compressVideoBtn,
		videoStatusLabel,
	)

	var selectedPhotoPath string
	var origW, origH float64
	var isUpdating bool

	photoFileLabel := widget.NewLabel("No file selected")
	photoStatusLabel := widget.NewLabel("Status: Ready")

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

		ext := filepath.Ext(selectedPhotoPath)
		base := selectedPhotoPath[0 : len(selectedPhotoPath)-len(ext)]
		outputPath := fmt.Sprintf("%s_resized%s", base, ext)

		go func() {
			err := processPhotoFile(selectedPhotoPath, outputPath, targetW, targetH)
			if err != nil {
				photoStatusLabel.SetText(fmt.Sprintf("Status: Processing failed: (%v)", err))
				dialog.ShowError(err, w)
			} else {
				photoStatusLabel.SetText("Status: Processing completed.")
				dialog.ShowInformation("Processing completed", fmt.Sprintf("Processed photo \nsaved to: %s", outputPath), w)
			}
		}()
	})

	photoUI := container.NewVBox(
		widget.NewLabel("Choose original photo"),
		selectPhotoBtn,
		photoFileLabel,
		widget.NewLabel(""),

		widget.NewLabel("Enter new resolution (Width x Height):"),
		container.NewGridWithColumns(2, widthEntry, heightEntry),
		keepRatioCheck,
		processPhotoBtn,
		photoStatusLabel,
	)

	var downloadFolder string
	ytUrlEntry := widget.NewEntry()
	ytUrlEntry.SetPlaceHolder("Enter YouTube URL")
	ytFolderLabel := widget.NewLabel("No folder selected")
	ytStatusLabel := widget.NewLabel("Status: Ready")

	ytProgressBar := widget.NewProgressBar()
	ytProgressBar.SetValue(0.0)

	formatSelect := widget.NewSelect([]string{
		"1080p Video",
		"720p Video",
		"480p Video",
		"Audio Only (MP3)",
	}, nil)
	formatSelect.SetSelected("1080p Video")

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
			} else {
				ytStatusLabel.SetText("Status: Download completed.")
				dialog.ShowInformation("Download completed", "Video has been saved successfully.", w)
			}
		}()
	})

	ytUI := container.NewVBox(
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

	tabs := container.NewAppTabs(
		container.NewTabItem("Video", videoUI),
		container.NewTabItem("Photo", photoUI),
		container.NewTabItem("YouTube", ytUI),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	w.SetContent(tabs)
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

func processPhotoFile(inputPath, outputPath, widthStr, heightStr string) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	exeDir := filepath.Dir(exePath)
	ffmpegPath := filepath.Join(exeDir, "ffmpeg.exe")

	if _, err := os.Stat(ffmpegPath); os.IsNotExist(err) {
		return fmt.Errorf("ffmpeg executable not found in the current directory")
	}

	scaleArg := fmt.Sprintf("scale=%s:%s", widthStr, heightStr)

	cmd := exec.Command(ffmpegPath, "-i", inputPath, "-vf", scaleArg, "-y", outputPath)

	err = cmd.Run()
	return err
}

func downloadYTVideo(url, outputFolder, formatChoice string, updateProgress func(progress float64)) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exeDir := filepath.Dir(exePath)
	ytdlpPath := filepath.Join(exeDir, "yt-dlp.exe")

	if _, err := os.Stat(ytdlpPath); os.IsNotExist(err) {
		return fmt.Errorf("yt-dlp executable not found in the current directory")
	}

	outputTemplate := filepath.Join(outputFolder, "%(title)s.%(ext)s")

	var args []string

	switch formatChoice {
	case "1080p Video":
		args = []string{"-f", "bestvideo[height<=1080]+bestaudio/best", "--merge-output-format", "mp4", "-o", outputTemplate, url}
	case "720p Video":
		args = []string{"-f", "bestvideo[height<=720]+bestaudio/best", "--merge-output-format", "mp4", "-o", outputTemplate, url}
	case "480p Video":
		args = []string{"-f", "bestvideo[height<=480]+bestaudio/best", "--merge-output-format", "mp4", "-o", outputTemplate, url}
	case "Audio Only (MP3)":
		args = []string{"-f", "bestaudio/best", "-x", "--audio-format", "mp3", "-o", outputTemplate, url}
	default:
		args = []string{"-f", "bestvideo+bestaudio/best", "--merge-output-format", "mp4", "-o", outputTemplate, url}
	}

	args = append(args, "--no-playlist", "--newline", "-o", outputTemplate, url)

	cmd := exec.Command(ytdlpPath, args...)
	cmd.SysProcAttr = hideWindowSysProcAttr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	progressRegex := regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%`)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		matches := progressRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			if percent, err := strconv.ParseFloat(matches[1], 64); err == nil {
				updateProgress(percent / 100.0)
			}
		}
	}

	return cmd.Wait()
}

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

var hideWindowSysProcAttr = &syscall.SysProcAttr{HideWindow: true}

func getToolPath(toolName string) string {
	exePath, err := os.Executable()
	if err != nil {
		return toolName
	}
	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, toolName)
}

func compressVideo(inputPath, outputPath, targetSizeStr string, updateProgress func(progress float64)) error {
	ffmpegPath := getToolPath("ffmpeg.exe")
	ffprobePath := getToolPath("ffprobe.exe")

	if _, err := os.Stat(ffmpegPath); os.IsNotExist(err) {
		return fmt.Errorf("ffmpeg.exe not found in the same folder")
	}

	targetSizeMB, err := strconv.ParseFloat(targetSizeStr, 64)
	if err != nil || targetSizeMB <= 0 {
		return fmt.Errorf("invalid target size: %v", err)
	}

	probeCmd := exec.Command(ffprobePath, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", inputPath)
	probeCmd.SysProcAttr = hideWindowSysProcAttr
	probeOutput, err := probeCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get video duration: %v", err)
	}

	durationSec, err := strconv.ParseFloat(strings.TrimSpace(string(probeOutput)), 64)
	if err != nil || durationSec <= 0 {
		return fmt.Errorf("failed to parse video duration")
	}

	audioBitrate := 128.0
	videoBitrate := ((targetSizeMB * 8192) / durationSec) - audioBitrate
	if videoBitrate < 100 {
		videoBitrate = 100
	}

	cmd := exec.Command(ffmpegPath, "-i", inputPath, "-b:v", fmt.Sprintf("%.0fk", videoBitrate), "-b:a", fmt.Sprintf("%.0fk", audioBitrate), "-progress", "pipe:1", "-nostats", "-y", outputPath)
	cmd.SysProcAttr = hideWindowSysProcAttr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_us=") {
			usStr := strings.TrimPrefix(line, "out_time_us=")
			if us, err := strconv.ParseFloat(usStr, 64); err == nil {
				progress := (us / 1000000.0) / durationSec
				if progress > 1.0 {
					progress = 1.0
				} else if progress < 0 {
					progress = 0
				}
				updateProgress(progress)
			}
		}
	}

	return cmd.Wait()
}

func processPhotoFile(inputPath, outputPath string, targetW, targetH int, updateProgress func(progress float64)) error {
	ffmpegPath := getToolPath("ffmpeg.exe")

	if _, err := os.Stat(ffmpegPath); os.IsNotExist(err) {
		return fmt.Errorf("ffmpeg.exe not found in the same folder")
	}

	updateProgress(0.1)

	scaleArg := fmt.Sprintf("scale=%d:%d", targetW, targetH)
	cmd := exec.Command(ffmpegPath, "-i", inputPath, "-vf", scaleArg, "-y", outputPath)
	cmd.SysProcAttr = hideWindowSysProcAttr

	updateProgress(0.5)

	err := cmd.Run()
	if err != nil {
		return err
	}

	updateProgress(1.0)
	return nil
}

func downloadYTVideo(url, outputFolder, formatChoice string, updateProgress func(progress float64)) error {
	ytdlpPath := getToolPath("yt-dlp.exe")

	if _, err := os.Stat(ytdlpPath); os.IsNotExist(err) {
		return fmt.Errorf("yt-dlp.exe not found in the same folder")
	}

	outputTemplate := filepath.Join(outputFolder, "%(title)s.%(ext)s")
	var args []string

	switch formatChoice {
	case "1080p Video":
		args = []string{"-f", "bestvideo[height<=1080]+bestaudio/best", "--merge-output-format", "mp4"}
	case "720p Video":
		args = []string{"-f", "bestvideo[height<=720]+bestaudio/best", "--merge-output-format", "mp4"}
	case "480p Video":
		args = []string{"-f", "bestvideo[height<=480]+bestaudio/best", "--merge-output-format", "mp4"}
	case "Audio Only (MP3)":
		args = []string{"-f", "bestaudio/best", "-x", "--audio-format", "mp3"}
	default:
		args = []string{"-f", "bestvideo+bestaudio/best", "--merge-output-format", "mp4"}
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

package main

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

//go:embed tools/*
var toolsFS embed.FS

var hideWindowSysProcAttr = &syscall.SysProcAttr{HideWindow: true}

var tempToolsDir string

func initTools() error {
	dir, err := os.MkdirTemp("", "MediaControllerTools-*")
	if err != nil {
		return err
	}
	tempToolsDir = dir

	files := []string{"ffmpeg.exe", "ffprobe.exe", "yt-dlp.exe"}
	for _, file := range files {
		data, err := toolsFS.ReadFile("tools/" + file)
		if err != nil {
			return fmt.Errorf("failed to read embedded tool %s: %v", file, err)
		}

		outPath := filepath.Join(tempToolsDir, file)
		err = os.WriteFile(outPath, data, 0777)
		if err != nil {
			return fmt.Errorf("failed to write tool to temp: %v", err)
		}
	}
	return nil
}

func cleanupTools() {
	if tempToolsDir != "" {
		os.RemoveAll(tempToolsDir)
	}
}

func getToolPath(toolName string) string {
	return filepath.Join(tempToolsDir, toolName)
}

func compressVideo(inputPath, outputPath, targetSizeStr string) error {
	ffmpegPath := getToolPath("ffmpeg.exe")
	ffprobePath := getToolPath("ffprobe.exe")

	if _, err := os.Stat(ffmpegPath); os.IsNotExist(err) {
		return fmt.Errorf("ffmpeg.exe not found in tools folder")
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

	cmd := exec.Command(ffmpegPath, "-i", inputPath, "-b:v", fmt.Sprintf("%.0fk", videoBitrate), "-b:a", fmt.Sprintf("%.0fk", audioBitrate), "-y", outputPath)
	cmd.SysProcAttr = hideWindowSysProcAttr
	return cmd.Run()
}

func processPhotoFile(inputPath, outputPath, widthStr, heightStr string) error {
	ffmpegPath := getToolPath("ffmpeg.exe")

	if _, err := os.Stat(ffmpegPath); os.IsNotExist(err) {
		return fmt.Errorf("ffmpeg.exe not found in tools folder")
	}

	scaleArg := fmt.Sprintf("scale=%s:%s", widthStr, heightStr)
	cmd := exec.Command(ffmpegPath, "-i", inputPath, "-vf", scaleArg, "-y", outputPath)
	cmd.SysProcAttr = hideWindowSysProcAttr
	return cmd.Run()
}

func downloadYTVideo(url, outputFolder, formatChoice string, updateProgress func(progress float64)) error {
	ytdlpPath := getToolPath("yt-dlp.exe")

	if _, err := os.Stat(ytdlpPath); os.IsNotExist(err) {
		return fmt.Errorf("yt-dlp.exe not found in tools folder")
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

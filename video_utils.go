package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"os/exec"
)

func getVideoAspectRatio(videoPath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", videoPath)
	var b bytes.Buffer
	cmd.Stdout = &b

	if err := cmd.Run(); err != nil {
		return "", err
	}

	type ffprobeOutput struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	var f ffprobeOutput
	if err := json.Unmarshal(b.Bytes(), &f); err != nil {
		return "", err
	}

	if f.Streams[0].Width == 0 || f.Streams[0].Height == 0 {
		return "", errors.New("Video Height/Width == 0")
	}
	w := f.Streams[0].Width
	h := f.Streams[0].Height
	r := float64(w) / float64(h)

	const (
		landscape = 16.0 / 9.0
		portrait  = 9.0 / 16.0
		eps       = 0.02 // error tolerance
	)

	if math.Abs(r-landscape) < eps {
		return "16:9", nil
	}
	if math.Abs(r-portrait) < eps {
		return "9:16", nil
	}
	return "other", nil
}

func processVideoForFastStart(filePath string) (string, error) {

	outputFileString := filePath + ".processed"
	cmd := exec.Command(
		"ffmpeg",
		"-i", filePath,
		"-c", "copy",
		"-movflags", "faststart",
		"-f", "mp4",
		outputFileString,
	)
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return outputFileString, nil
}

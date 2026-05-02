package acrrual_utils

import (
	"bufio"
	"fmt"
	"os"
)

type Mechanic struct {
	Match      string `json:"match"`
	Reward     int    `json:"reward"`
	RewardType string `json:"reward_type"`
}

type FileScanner struct {
	file    *os.File
	scanner *bufio.Scanner
}

func NewFileScanner(filename string) (*FileScanner, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	return &FileScanner{file: file, scanner: bufio.NewScanner(file)}, nil
}

func (s *FileScanner) Close() error {
	return s.file.Close()
}

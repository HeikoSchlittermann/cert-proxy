package main

import (
	"bufio"
	"io"
	"os"
	"strings"
)

type CNList map[string]interface{}

func (cnlist *CNList) AppendFromFile(filename string) (err error) {

	var file io.ReadCloser

	switch filename {
	case "":
		return
	case "-":
		file = os.Stdin
	default:
		file, err = os.Open(filename)
		if err != nil {
			return
		}
		defer file.Close()
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if l := strings.Trim(strings.SplitN(scanner.Text(), "#", 2)[0], " \t\r"); l != "" {
			(*cnlist)[l] = nil
		}
	}
	return scanner.Err()
}

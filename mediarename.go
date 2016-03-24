// Copyright ©2016 The mediarename Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

func main() {
	var (
		dry = flag.Bool("n", false, "Dry run")
	)
	flag.Parse()

	_, err := exec.LookPath("exiftool")
	if err != nil {
		log.Fatal("exiftool not found")
	}

	files, err := ioutil.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		src := file.Name()
		ext := strings.ToLower(filepath.Ext(src))
		if ext != ".jpg" && ext != ".mov" && ext != ".dng" {
			// Unsupported extension.
			continue
		}

		tags, err := ReadTags(src)
		if err != nil {
			continue
		}

		if err = tags.Valid(); err != nil {
			fmt.Printf("%s: %v, skipping\n", src, err)
			continue
		}

		dst := tags.ToFileName()
		if !*dry {
			os.Chmod(dst, 0644)
			os.Chtimes(dst, tags.DateTimeOriginal, tags.DateTimeOriginal)
		}
		if _, err = os.Stat(dst); err == nil {
			fmt.Println("Destination file", dst, "exists, skipping.")
			continue
		}
		if !*dry {
			os.Rename(src, dst)
		}
		fmt.Println("Renamed", src, "to", dst)
	}
}

func ReadTags(filename string) (*ExifTags, error) {
	// Print dates in RFC3339 format.
	out, err := exec.Command("exiftool", "-j", "-d", "%Y-%m-%dT%H:%M:%SZ", filename).Output()
	if err != nil {
		return nil, err
	}

	var tags []ExifTags
	err = json.Unmarshal(out, &tags)
	if err != nil {
		return nil, err
	}

	return &tags[0], nil
}

type ExifTags struct {
	DateTimeOriginal time.Time
	FileName         string
	FileNumber       string
	Model            string
}

func (tags *ExifTags) Valid() error {
	switch {
	case tags.FileName == "":
		return errors.New("no FileName tag")
	case tags.DateTimeOriginal.IsZero():
		return errors.New("no DateTimeOriginal tag")
	default:
		return nil
	}
}

func (tags *ExifTags) ToFileName() string {
	name := "VCH" // Given prefix. TODO: Configurable?
	ext := strings.ToLower(filepath.Ext(tags.FileName))

	t := tags.DateTimeOriginal.Format(time.RFC3339)
	name += "_" + strings.Replace(t, ":", ".", -1)

	if tags.Model != "" {
		name += "_" + strings.Replace(tags.Model, " ", "", -1)
	}

	n := tags.FileNumber
	if n == "" {
		base := strings.TrimSuffix(tags.FileName, ext)
		n = strings.TrimPrefix(base, strings.TrimRightFunc(base, func(r rune) bool {
			return unicode.IsDigit(r)
		}))
	}
	if n != "" {
		name += "_" + n
	}

	return name + ext
}

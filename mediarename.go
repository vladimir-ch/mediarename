// Copyright ©2016 The mediarename Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
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

func usage() {
	fmt.Fprintf(os.Stderr, "usage: mediarename [options]\n")
	fmt.Fprintf(os.Stderr, "options:\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	var (
		dry    = flag.Bool("n", false, "Dry run")
		prefix = flag.String("p", "VCH", "File name prefix")
		tz     = flag.String("tz", "UTC", "Time zone")
	)
	flag.Usage = usage
	flag.Parse()

	_, err := exec.LookPath("exiftool")
	if err != nil {
		log.Fatal("exiftool not found")
	}

	loc, err := time.LoadLocation(*tz)
	if err != nil {
		log.Fatal(err)
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

		var supported bool
		switch ext {
		case ".jpg", ".dng", ".cr2", ".mov", ".mp4":
			supported = true
		}
		if !supported {
			continue
		}

		tags, err := ReadTags(src)
		if err != nil {
			fmt.Printf("error reading tags from %v: %v\n", src, err)
			continue
		}
		if tags.FileName == "" {
			tags.FileName = src
		}

		dst, err := tags.ToFileName(loc)
		if err != nil {
			fmt.Printf("error creating destination file name for %s (%v)\n", src, err)
			continue
		}

		t, err := tags.TimeIn(loc)
		if err != nil {
			fmt.Printf("error formatting time for %s (%v)\n", src, err)
			continue
		}

		dst = *prefix + "_" + dst
		if _, err = os.Stat(dst); err == nil {
			fmt.Println("Destination file", dst, "exists, skipping.")
			if !*dry {
				os.Chmod(dst, 0644)
				os.Chtimes(dst, t, t)
			}
			continue
		}
		if !*dry {
			os.Rename(src, dst)
			os.Chmod(dst, 0644)
			os.Chtimes(dst, t, t)
		}
		fmt.Println("Renamed", src, "to", dst)
	}
}

func ReadTags(filename string) (*ExifTags, error) {
	out, err := exec.Command("exiftool", "-j", filename).Output()
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
	DateTimeOriginal string
	FileName         string
	FileNumber       string
	Model            string
}

func (tags *ExifTags) TimeIn(loc *time.Location) (time.Time, error) {
	t, err := time.ParseInLocation("2006:01:02 15:04:05", tags.DateTimeOriginal, loc)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func (tags *ExifTags) ToFileName(loc *time.Location) (string, error) {
	t, err := tags.TimeIn(loc)
	if err != nil {
		return "", err
	}

	ft := t.Format(time.RFC3339)
	name := strings.Replace(ft, ":", ".", -1) // Remove : from the file name because of Windows.

	if tags.Model != "" {
		name += "_" + strings.Replace(tags.Model, " ", "", -1)
	}

	ext := strings.ToLower(filepath.Ext(tags.FileName))
	n := tags.FileNumber
	if n == "" {
		base := strings.TrimSuffix(tags.FileName, ext)
		// base without the longest sequence of digits on the right.
		basePrefix := strings.TrimRightFunc(base, func(r rune) bool {
			return unicode.IsDigit(r)
		})
		n = strings.TrimPrefix(base, basePrefix)
	}
	if n != "" {
		name += "_" + n
	}

	return name + ext, nil
}

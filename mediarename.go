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

		if err = tags.Valid(); err != nil {
			fmt.Printf("%s: %v, skipping\n", src, err)
			continue
		}

		// Given prefix.
		// TODO: Make it configurable.
		dst := "VCH_" + tags.ToFileName()
		if !*dry {
			os.Chmod(dst, 0644)
			os.Chtimes(dst, tags.DateTimeOriginal.Time, tags.DateTimeOriginal.Time)
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

type CustomTime struct {
	time.Time
}

func (ct *CustomTime) UnmarshalJSON(b []byte) (err error) {
	t := strings.Replace(string(b), "\"", "", -1)
	ct.Time, err = time.Parse("2006:01:02 15:04:05", t)
	return err
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
	DateTimeOriginal CustomTime
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
	t := tags.DateTimeOriginal.Format(time.RFC3339)
	// Remove : from the file name because of Windows.
	name := strings.Replace(t, ":", ".", -1)

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

	return name + ext
}

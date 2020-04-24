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

func usage() {
	fmt.Fprintf(os.Stderr, "usage: mediarename [options]\n")
	fmt.Fprintf(os.Stderr, "options:\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	log.SetPrefix("mediarename: ")
	log.SetFlags(0)
	var (
		dry    = flag.Bool("n", false, "Dry run")
		prefix = flag.String("p", "", "File name prefix")
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
		tags, err := ReadTags(src)
		if err != nil {
			log.Printf("error reading tags from %v (%v)\n", src, err)
			continue
		}
		if tags.FileName == "" {
			tags.FileName = src
		}

		dst, err := tags.ToFileName(loc)
		if err != nil {
			log.Printf("error creating destination file name for %s (%v)\n", src, err)
			continue
		}
		if *prefix != "" {
			dst = *prefix + "_" + dst
		}

		t, err := tags.TimeIn(loc)
		if err != nil {
			log.Printf("error formatting time for %s (%v)\n", src, err)
			continue
		}

		if _, err = os.Stat(dst); err == nil {
			log.Printf("destination file %s exists, skipping.", dst)
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
		log.Printf("renamed %v to %v", src, dst)
	}
}

func ReadTags(filename string) (*ExifTags, error) {
	out, err := exec.Command("exiftool", "-j", filename).Output()
	if err != nil {
		return nil, errors.New("exiftool error")
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
	CreateDate       string
	MediaCreateDate  string
	ModifyDate       string

	FileName   string
	FileNumber string

	Model       string
	Information string
}

func (tags *ExifTags) TimeIn(loc *time.Location) (time.Time, error) {
	date := tags.DateTimeOriginal
	if date == "" {
		date = tags.CreateDate
	}
	if date == "" {
		date = tags.MediaCreateDate
	}
	if date == "" {
		date = tags.ModifyDate
	}
	if date == "" {
		return time.Time{}, errors.New("no date tag found")
	}

	t, err := time.ParseInLocation("2006:01:02 15:04:05", date, loc)
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

	fields := []string{t.Format(time.RFC3339)}

	model := tags.Model
	if model == "" {
		// For example MOV files from Ricoh WG-M1 store the model in
		// Information.
		model = tags.Information
	}
	if model != "" {
		fields = append(fields, model)
	}

	filenum := tags.FileNumber
	if filenum == "" {
		// FileNumber tag doesn't exist, try to extract a number from
		// the file name.
		filenum = fileNumberFromPath(tags.FileName)
	}
	if filenum != "" {
		fields = append(fields, filenum)
	}

	filename := strings.Join(fields, "_")
	filename = strings.Replace(filename, ":", ".", -1)      // Remove : from the file name because of Windows.
	filename = strings.Replace(filename, " ", "", -1)       // Remove spaces from the filename.
	filename = strings.Replace(filename, string(0), "", -1) // Remove zero bytes.
	ext := strings.ToLower(filepath.Ext(tags.FileName))
	return filename + ext, nil
}

func fileNumberFromPath(path string) string {
	base := strings.TrimSuffix(path, filepath.Ext(path))
	prefix := strings.TrimRightFunc(base, func(r rune) bool {
		return unicode.IsDigit(r)
	})
	return strings.TrimPrefix(base, prefix)
}

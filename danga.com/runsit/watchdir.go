/*
Copyright 2011 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"io"
	"os"
	"strings"
	"time"
)

const buffered = 8

type DirWatcher interface {
	Updates() <-chan TaskFile
}

type TaskFile interface {
	// Name returns the task's base name, without any directory
	// prefix or .json suffix.
	Name() string

	// Open opens the JSON configuration file for reading.
	Open() (io.ReadCloser, error)
}

var osDirWatcher DirWatcher // if nil, default polling impl is used

func dirWatcher() DirWatcher {
	if dw := osDirWatcher; dw != nil {
		return dw
	}
	return &pollingDirWatcher{dir: *configDir}
}

// pollingDirWatcher is the portable implementation of DirWatcher that
// simply polls the directory every few seconds.
type pollingDirWatcher struct {
	dir string
	c   chan TaskFile
}

func (w *pollingDirWatcher) Updates() <-chan TaskFile {
	if w.c == nil {
		w.c = make(chan TaskFile, buffered)
		go w.poll()
	}
	return w.c
}

func (w *pollingDirWatcher) poll() {
	last := map[string]time.Time{} // last modtime
	for {
		d, err := os.Open(w.dir)
		if err != nil {
			logger.Printf("Error opening directory %q: %v", w.dir, err)
			time.Sleep(15 * time.Second)
			continue
		}
		fis, err := d.Readdir(-1)
		if err != nil {
			logger.Printf("Error reading directory %q: %v", w.dir, err)
			time.Sleep(15 * time.Second)
			continue
		}
		for _, fi := range fis {
			name := fi.Name()
			if !strings.HasSuffix(name, ".json") {
				continue
			}
			m := fi.ModTime()
			if em, ok := last[name]; ok && em.Equal(m) {
				continue
			}
			logger.Printf("name = %q, modtime = %v", name, m)
			last[name] = m
		}
		time.Sleep(5 * time.Second)
	}
}
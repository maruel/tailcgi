// Copyright 2025 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Tool tailcgi tails a file.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cgi"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
)

const hdr = `<!DOCTYPE html>
<html>
<head><title>Log :: %s</title>
<style type="text/css">
pre { font-family: monospace; }
.preformattedContent {
	overflow: auto;
	width: 100%%;
	height: 100%%;
}
</style>
</head>
<body>
<p/>
<div id="Content">
<div class="preformatted"><div class="preformattedContent">
<pre>
`

func serverCGI() (int, error) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if len(os.Args) != 1 {
		return 500, errors.New("unexpected arguments")
	}

	r, err := cgi.Request()
	if err != nil {
		return 400, fmt.Errorf("invalid request: %w", err)
	}
	if !strings.HasPrefix(r.URL.Path, "/") {
		return 400, fmt.Errorf("invalid request: %q", r.URL.Path)
	}
	filename := r.URL.Path[1:]
	if filename != filepath.Base(filename) {
		return 400, fmt.Errorf("invalid request: %q", r.URL.Path)
	}
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return 404, errors.New("file not found")
		}
		return 400, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return 500, fmt.Errorf("error creating watcher: %w", err)
	}
	defer watcher.Close()
	if err = watcher.Add(filename); err != nil {
		return 500, fmt.Errorf("error creating watcher: %w", err)
	}
	// At that point, we cannot return an error anymore.
	fmt.Printf("Content-Type: text/html; charset=utf-8\n")
	fmt.Printf("Status: 200 OK\n\n")
	fmt.Printf(hdr, filename)
	// TODO:
	// - grep
	// - parse JSON
	// - only display last N lines
	if _, err = io.Copy(os.Stdout, f); err != nil {
		return 0, nil
	}
	os.Stdout.Sync()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return 0, nil
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				if _, err := io.Copy(os.Stdout, f); err != nil {
					return 0, nil
				}
				os.Stdout.Sync()
			}
		case <-watcher.Errors:
			return 0, nil
		case <-ctx.Done():
			return 0, nil
		}
	}
}

func main() {
	fmt.Printf("Cache-Control: no-cache\n")
	fmt.Printf("X-Accel-Buffering: no\n")
	if code, err := serverCGI(); err != nil {
		fmt.Printf("Content-Type: text/plain\n")
		fmt.Printf("Status: %d %s\n\n", code, http.StatusText(code))
		fmt.Printf("Error: %v\n", err)
	}
}

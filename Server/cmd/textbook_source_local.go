//go:build !textbook_embed

package main

import (
	"io/fs"
	"os"
)

func textbookPages() fs.FS {
	dir := os.Getenv("TEXTBOOK_DIR")
	if dir == "" {
		dir = "../Textbooks/pages"
	}
	return os.DirFS(dir)
}

//go:build textbook_embed

package main

import (
	"embed"
	"io/fs"
)

//go:embed textbooks
var embeddedTextbooks embed.FS

func textbookPages() fs.FS {
	pages, err := fs.Sub(embeddedTextbooks, "textbooks")
	if err != nil {
		panic(err)
	}
	return pages
}

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: validate_textbooks DIRECTORY")
	}
	textbooks, err := repo.NewTextbooks(context.Background(), os.DirFS(os.Args[1]))
	if err != nil {
		log.Fatal(err)
	}
	list, err := textbooks.List(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	if len(list) == 0 {
		log.Fatal("no textbook pages found")
	}
	_, _ = fmt.Fprintln(os.Stdout, "validated textbook pages:", len(list))
}

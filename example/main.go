package main

import (
	"log"
	"os"

	"github/cross-cpm/go-shutil"
)

func main() {
	src := os.Args[1]
	dst := os.Args[2]
	log.Printf("test copy %s to %s\n", src, dst)
	_, err := shutil.CopyTree(src, dst, nil)
	log.Println("error", err)
}

/*
Diff takes two paths as input and checks them for differences. If both paths are files then diff reports whether the
files are different. If they are directories, it reports differences in contents of the directories.

Usage:

	diff [flags] path1 path2

The flags are:

	-h, --help        Print this help.
	-r, --recursive   Recursively compare directories.

Diff's reporting is not provided in any specific order and may vary across runs as it parallelizes comparisons.
*/
package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/pflag"
)

// Number of bytes to read at once from a file.
const CHUNK_SIZE = 4 * 1024

var wg sync.WaitGroup
var red = color.New(color.FgHiRed).SprintFunc()
var yellow = color.New(color.FgHiYellow).SprintFunc()
var magenta = color.New(color.FgHiMagenta).SprintFunc()

// checkErr checks for a non nil error and exits the program after logging it.
func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// cmpFiles compares two files byte for byte and returns whether they are equal or not.
func cmpFiles(file1 string, file2 string) bool {
	// Open both files and get their stats.
	f1, err := os.Open(file1)
	checkErr(err)
	defer f1.Close()

	stat1, err := f1.Stat()
	checkErr(err)

	f2, err := os.Open(file2)
	checkErr(err)
	defer f2.Close()

	stat2, err := f2.Stat()
	checkErr(err)

	// If files have different sizes they cannot be same.
	if stat1.Size() != stat2.Size() {
		return false
	}

	// Read bytes in chunks and compare them.
	b1 := make([]byte, CHUNK_SIZE)
	b2 := make([]byte, CHUNK_SIZE)
	for {
		n1, err1 := f1.Read(b1)
		n2, err2 := f2.Read(b2)

		// If both files end at the same time they are the same, otherwise they are different.
		if err1 == io.EOF && err2 == io.EOF {
			return true
		} else if err1 == io.EOF && err2 == nil {
			return false
		} else if err1 == nil && err2 == io.EOF {
			return false
		} else if err1 != nil || err2 != nil {
			log.Fatal(err1, err2)
		}

		// If number of bytes read are not same files are different.
		if n1 != n2 {
			return false
		}

		if n1 < CHUNK_SIZE {
			b1 = b1[:CHUNK_SIZE]
			b2 = b2[:CHUNK_SIZE]
		}

		// If all bytes are not same files are different.
		if !bytes.Equal(b1, b2) {
			return false
		}
	}
}

// diffFiles compares two files and outputs whether they are different. Should be called via a goroutine.
func diffFiles(file1 string, file2 string) {
	if !cmpFiles(file1, file2) {
		fmt.Printf("Files %v and %v %s\n", file1, file2, red("differ"))
	}

	wg.Done()
}

// diffDirs compares two directories (recursively if specified) and outputs which items are different. Should be called
// via a goroutine.
func diffDirs(dir1 string, dir2 string, recursive bool) {
	// Read directories.
	files1, err := os.ReadDir(dir1)
	checkErr(err)
	files2, err := os.ReadDir(dir2)
	checkErr(err)

	// Creates maps for tracking which files have been checked.
	type d struct {
		e fs.DirEntry
		c bool
	}
	fileSet1 := make(map[string]d)
	for _, f := range files1 {
		fileSet1[f.Name()] = d{f, false}
	}
	fileSet2 := make(map[string]d)
	for _, f := range files2 {
		fileSet2[f.Name()] = d{f, false}
	}

	// Iterate through contents first directory.
	for _, f := range files1 {
		name := f.Name()
		f2, ok := fileSet2[name]

		// If item is present in second directory, compare them if possible.
		if ok {
			path1 := path.Join(dir1, name)
			path2 := path.Join(dir2, name)
			if !f.IsDir() && !f2.e.IsDir() {
				wg.Add(1)
				go diffFiles(path1, path2)
			} else if f.IsDir() && f2.e.IsDir() {
				if recursive {
					wg.Add(1)
					go diffDirs(path1, path2, true)
				} else {
					fmt.Printf("Common subdirectories: %v and %v\n", path1, path2)
				}
			} else if f.IsDir() && !f2.e.IsDir() {
				fmt.Printf("%v is a %s while %v is a %s\n", path1, magenta("directory"), path2, magenta("file"))
			} else {
				fmt.Printf("%v is a %s while %v is a %s\n", path1, magenta("file"), path2, magenta("directory"))
			}

			f2.c = true
			fileSet2[name] = f2
		} else {
			fmt.Printf("%s %v: %v\n", yellow("Only in"), dir1, name)
		}
	}

	// All non-checked items in second directory are only present in that directory.
	for _, f := range files2 {
		name := f.Name()

		if !fileSet2[name].c {
			fmt.Printf("%s %v: %v\n", yellow("Only in"), dir2, name)
		}
	}

	wg.Done()
}

func main() {
	log.SetFlags(0)

	// Define and parse arguments.
	help := pflag.BoolP("help", "h", false, "Print this help.")
	recursive := pflag.BoolP("recursive", "r", false, "Recursively compare directories.")
	pflag.Parse()

	// Print help if requested or if wrong number of arguments are provided.
	if *help || len(pflag.Args()) != 2 {
		fmt.Println("Usage: diff [flags] path1 path2")
		pflag.PrintDefaults()
		os.Exit(0)
	}

	// Ensure path1 and path2 are either both files or both directories and act accordingly.
	path1 := pflag.Args()[0]
	path2 := pflag.Args()[1]
	stat1, err := os.Stat(path1)
	checkErr(err)
	stat2, err := os.Stat(path2)
	checkErr(err)
	if !stat1.IsDir() && !stat2.IsDir() {
		wg.Add(1)
		go diffFiles(path1, path2)
		wg.Wait()
	} else if stat1.IsDir() && stat2.IsDir() {
		wg.Add(1)
		go diffDirs(path1, path2, *recursive)
		wg.Wait()
	} else {
		fmt.Println("Cannot compare between a file and a directory.")
		os.Exit(1)
	}
}

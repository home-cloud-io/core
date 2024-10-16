package host

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type (
	// Replacer take in a line in a file and outputs the replacement line (which could be the same if no change is needed)
	Replacer func(line string) string
)

var (
	// fileMutex is a safety check to make sure we don't accidentally write to the same file from multiple threads
	// in the future this could be put into a map keyed off of filenames to allow parallel writes to different files
	fileMutex = sync.Mutex{}
)

// LineByLineReplace will process all lines in the given file running all Replacers against each line.
// NOTE: the Replacers will be run in the order they appear in the slice
func LineByLineReplace(filename string, replacers []Replacer) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// read original file
	reader, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer reader.Close()

	// create temp file
	writer, err := os.CreateTemp("", fmt.Sprintf("%s-*.tmp", filepath.Base(filename)))
	if err != nil {
		return err
	}
	defer writer.Close()

	// execute replacers (writing into the temp file)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		for _, r := range replacers {
			line = r(line)
		}
		_, err := io.WriteString(writer, line+"\n")
		if err != nil {
			return err
		}
	}
	err = scanner.Err()
	if err != nil {
		return err
	}

	// make sure the temp file was successfully written to
	err = writer.Close()
	if err != nil {
		return err
	}

	// close original file
	err = reader.Close()
	if err != nil {
		return err
	}

	// overwrite the original file with the temp file
	err = os.Rename(writer.Name(), filename)
	if err != nil {
		return err
	}

	return nil
}

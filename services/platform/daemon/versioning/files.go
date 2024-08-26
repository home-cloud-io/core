package versioning

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type (
	// replacer take in a line in a file and outputs the replacement line (which could be the same if no change is needed)
	replacer func(line string) string
)

// lineByLineReplace will process all lines in the given file running all replacers against each line.
// NOTE: the replacers will be run in the order they appear in the slice
func lineByLineReplace(filename string, replacers []replacer) error {
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

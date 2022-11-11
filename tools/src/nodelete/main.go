// Command nodelete takes a patch file on stdin, and behaves like `sed /^-/d`, but doesn't produce a
// malformed patch.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
)

func main() {
	if err := Filter(os.Stdout, os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
		os.Exit(1)
	}
}

var (
	reA   = regexp.MustCompile(`^--- \S+` + "\t")
	reB   = regexp.MustCompile(`^\+\+\+ \S+` + "\t")
	reSec = regexp.MustCompile(`^@@ -([0-9]+),[0-9]+ \+([0-9]+),[0-9]+ @@`)
)

func Filter(dst io.Writer, src io.Reader) error {
	scanner := bufio.NewScanner(src)

	var file []string

	var section struct {
		empty bool
		lines []string

		aBeg int
		aLen int
		bBeg int
		bLen int
	}
	section.empty = true

	flushSection := func() {
		if !section.empty {
			file = append(file, fmt.Sprintf("@@ -%d,%d +%d,%d @@",
				section.aBeg, section.aLen,
				section.bBeg, section.bLen))
			file = append(file, section.lines...)
		}
		section.empty = true
		section.lines = nil
		section.aLen = 0
		section.bLen = 0
	}

	flushFile := func() {
		flushSection()
		if len(file) > 2 {
			for _, line := range file {
				fmt.Fprintln(dst, line)
			}
		}
		file = nil
	}

	i := 0
	for scanner.Scan() {
		i++
		line := scanner.Text()
		if line == "" {
			flushFile()
			fmt.Fprintln(dst)
			continue
		}
		switch line[0] {
		case '-':
			if reA.MatchString(line) {
				flushFile()
				file = append(file, line)
			}
		case '+':
			if reB.MatchString(line) {
				flushFile()
				file = append(file, line)
			} else {
				section.lines = append(section.lines, line)
				section.empty = false
				section.bLen++
			}
		case ' ':
			section.lines = append(section.lines, line)
			section.aLen++
			section.bLen++
		case '@':
			flushSection()
			parts := reSec.FindStringSubmatch(line)
			if parts == nil {
				return fmt.Errorf("line %d doesn't look like a patch: %q", i, line)
			}
			section.aBeg, _ = strconv.Atoi(parts[1])
			section.bBeg, _ = strconv.Atoi(parts[2])
		default:
			return fmt.Errorf("line %d doesn't look like a patch: %q", i, line)
		}
	}
	flushFile()
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

package main

import (
	"errors"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/fatih/color"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mfroeh/gogrep/regex"
)

var submatchColors = []*color.Color{
	color.New(color.FgRed),
	color.New(color.FgGreen),
	color.New(color.FgYellow),
	color.New(color.FgBlue),
	color.New(color.FgMagenta),
	color.New(color.FgCyan),
}

var cli struct {
	Pattern string   `arg:"" name:"pattern" help:"Regex pattern to use in search" type:"string"`
	Paths   []string `arg:"" optional:"" name:"path" help:"Paths to search" type:"path"`
}

func main() {
	kong.Parse(&cli,
		kong.Name("gogrep"),
		kong.Description("Recursively searches the current directory for lines matching a regex pattern."),
		kong.UsageOnError(),
	)

	re, err := regex.Compile(cli.Pattern)
	if err != nil {
		log.Fatalf("failed to build regex: %v", err)
	}

	if len(cli.Paths) == 0 {
		cli.Paths = []string{"."}
	}

	for _, path := range cli.Paths {
		info, err := os.Lstat(path)
		if err != nil {
			log.Fatalf("%s: %v", path, err)
		}

		if info.IsDir() {
			err = recursivelySearchDir(path, &re)
		} else {
			err = searchFile(path, &re)
		}

		if err != nil {
			log.Fatalf("%v", err)
		}
	}

}

func recursivelySearchDir(path string, re *regex.Regex) error {
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		// resolve symlinks
		var info os.FileInfo
		for {
			info, err = os.Stat(path)
			// symlinks may be broken, in that case, just ignore them
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			if info.Mode()&fs.ModeSymlink != fs.ModeSymlink {
				break
			}

			path, err = os.Readlink(path)
			if err != nil {
				return err
			}
		}

		// symlink may resolve to a directory, in which case we just ignore it
		if info.IsDir() {
			return nil
		}

		return searchFile(path, re)
	})

	return err
}

func searchFile(path string, re *regex.Regex) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	printFileHeader := false
	for i, line := range strings.Split(string(content), "\n") {
		matches := re.FindAllSubmatches(line, -1)
		if len(matches) == 0 {
			continue
		}

		if !printFileHeader {
			printFileHeader = true
			fmt.Println(path, ":")
		}

		out := strings.Builder{}
		lastMatchEnd := 0
		for _, match := range matches {
			out.WriteString(line[lastMatchEnd:match[0].Offset])
			out.WriteString(formatMatch(match))
			lastMatchEnd = match[0].Offset + len(match[0].Str)
		}
		out.WriteString(line[lastMatchEnd:])
		fmt.Printf("%d:%s\n", i+1, out.String())
	}

	if printFileHeader {
		fmt.Println()
	}

	return nil
}

func formatMatch(match []regex.Submatch) string {
	fullMatch := match[0].Str
	if len(match) == 1 || len(match) > len(submatchColors) {
		return submatchColors[0].Sprint(fullMatch)
	}

	out := strings.Builder{}
	matchOff := 0
	for i, sm := range match[1:] {
		offRelativeToMatch := sm.Offset - match[0].Offset
		submatchColors[0].Fprint(&out, fullMatch[matchOff:offRelativeToMatch])
		submatchColors[i+1].Fprint(&out, sm.Str)
		matchOff = offRelativeToMatch + len(sm.Str)
	}
	submatchColors[0].Fprint(&out, fullMatch[matchOff:])
	return out.String()
}

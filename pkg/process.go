// Copyright (c) 2017 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package pkg

import (
	"bytes"
	"errors"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/vcs"
)

// Implementation taken from "isStandardImportPath" in go's source.
func isStdLibPath(path string) bool {
	i := strings.Index(path, "/")
	if i < 0 {
		i = len(path)
	}
	elem := path[:i]
	return !strings.Contains(elem, ".")
}

// sortImports takes in an "import" body and returns it sorted
func sortImports(in []byte, sections []string) []byte {
	type importLine struct {
		index int    // index into inLines
		path  string // import path used for sorting
	}
	// imports holds all the import lines, separated by section. The
	// first section is for stdlib imports, the following sections
	// hold the user specified sections, the final section is for
	// everything else.
	imports := make([][]importLine, len(sections)+2)
	addImport := func(section, index int, importPath string) {
		imports[section] = append(imports[section], importLine{index, importPath})
	}
	stdlib := 0
	offset := 1
	other := len(imports) - 1

	inLines := bytes.Split(in, []byte{'\n'})
	for i, line := range inLines {
		if len(line) == 0 {
			continue
		}
		start := bytes.IndexByte(line, '"')
		if start == -1 {
			continue
		}
		if comment := bytes.Index(line, []byte("//")); comment > -1 && comment < start {
			continue
		}

		start++ // skip '"'
		end := bytes.IndexByte(line[start:], '"') + start
		s := string(line[start:end])

		found := false
		for j, sect := range sections {
			if strings.HasPrefix(s, sect) && (len(sect) == len(s) || s[len(sect)] == '/') {
				addImport(j+offset, i, s)
				found = true
				break
			}
		}
		if found {
			continue
		}

		if isStdLibPath(s) {
			addImport(stdlib, i, s)
		} else {
			addImport(other, i, s)
		}
	}

	out := make([]byte, 0, len(in)+2)
	needSeperator := false
	for _, section := range imports {
		if len(section) == 0 {
			continue
		}
		if needSeperator {
			out = append(out, '\n')
		}
		sort.Slice(section, func(a, b int) bool {
			return section[a].path < section[b].path
		})
		for _, s := range section {
			out = append(out, inLines[s.index]...)
			out = append(out, '\n')
		}
		needSeperator = true
	}

	return out
}

func genFile(in []byte, sections []string) ([]byte, error) {
	out := make([]byte, 0, len(in)+3) // Add some fudge to avoid re-allocation

	for {
		const importLine = "\nimport (\n"
		const importLineLen = len(importLine)
		importStart := bytes.Index(in, []byte(importLine))
		if importStart == -1 {
			break
		}
		// Save to `out` everything up to and including "import(\n"
		out = append(out, in[:importStart+importLineLen]...)
		in = in[importStart+importLineLen:]
		importLen := bytes.Index(in, []byte("\n)\n"))
		if importLen == -1 {
			return nil, errors.New(`parsing error: missing ")"`)
		}
		// Sort body of "import" and write it to `out`
		out = append(out, sortImports(in[:importLen], sections)...)
		out = append(out, []byte(")")...)
		in = in[importLen+2:]
	}
	// Write everything leftover to out
	out = append(out, in...)
	return out, nil
}

// returns true if the file changed
func ProcessFile(filename string, writeFile, listDiffFiles bool, sections []string) (bool, error) {
	in, err := ioutil.ReadFile(filename)
	if err != nil {
		return false, err
	}
	out, err := genFile(in, sections)
	if err != nil {
		return false, err
	}

	equal := bytes.Equal(in, out)
	if listDiffFiles {
		return !equal, nil
	}
	if !writeFile {
		os.Stdout.Write(out)
		return !equal, nil
	}

	if equal {
		return false, nil
	}
	temp, err := ioutil.TempFile(filepath.Dir(filename), filepath.Base(filename))
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(temp.Name())
	s, err := os.Stat(filename)
	if err != nil {
		return false, err
	}
	if _, err = temp.Write(out); err != nil {
		return false, err
	}
	if err := temp.Close(); err != nil {
		return false, err
	}
	if err := os.Chmod(temp.Name(), s.Mode()); err != nil {
		return false, err
	}
	if err := os.Rename(temp.Name(), filename); err != nil {
		return false, err
	}

	return true, nil
}

// maps directory to vcsRoot
var vcsRootCache = make(map[string]string)

func VcsRootImportPath(f string) (string, error) {
	path, err := filepath.Abs(f)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if root, ok := vcsRootCache[dir]; ok {
		return root, nil
	}
	gopath := build.Default.GOPATH
	var root string
	_, root, err = vcs.FromDir(dir, filepath.Join(gopath, "src"))
	if err != nil {
		return "", err
	}
	vcsRootCache[dir] = root
	return root, nil
}

func Process(filepath,
	filename string,
	writeFile *bool,
	listDiffFiles *bool,
	sections Multistring,
	pattern *regexp.Regexp) error {
	if pattern != nil {
		if match := pattern.MatchString(filename); match {
			return nil
		}
	}

	diff, err := ProcessFile(filepath, *writeFile, *listDiffFiles, sections)
	if err != nil {
		return err
	}
	if *listDiffFiles && diff {
		fmt.Println(filepath)
	}
	return nil
}

func Visit(writeFile *bool,
	listDiffFiles *bool,
	sections Multistring,
	patterns *regexp.Regexp) func(path string, info os.FileInfo, err error) error {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// only Visit go files
		if filepath.Ext(path) != ".go" {
			return nil
		}
		if err := Process(path,
			info.Name(),
			writeFile,
			listDiffFiles,
			sections,
			patterns); err != nil {
			fmt.Fprintf(os.Stderr, "error while proccessing file %q: %s", path, err)
		}
		return nil
	}
}

type Multistring []string

func (m *Multistring) String() string {
	return strings.Join(*m, ", ")
}
func (m *Multistring) Set(s string) error {
	*m = append(*m, s)
	return nil
}

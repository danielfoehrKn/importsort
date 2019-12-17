// Copyright (c) 2017 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/danielfoehrKn/importsort/pkg"
)

func main() {
	writeFile := flag.Bool("w", false, "write result to file instead of stdout")
	listDiffFiles := flag.Bool("l", false, "list files whose formatting differs from importsort")
	pattern := flag.String("exclude", "", "exclude filenames matching these regex pattern")
	var sections pkg.Multistring
	flag.Var(&sections, "s", "package `prefix` to define an import section,"+
		` ex: "cvshub.com/company". May be specified multiple times.`+
		" If not specified the repository root is used.")

	flag.Parse()

	var compiledPattern *regexp.Regexp
	if len(*pattern) > 0 {
		compiledPattern = regexp.MustCompile(*pattern)
	}

	checkVCSRoot := sections == nil
	for _, f := range flag.Args() {
		if checkVCSRoot {
			root, err := pkg.VcsRootImportPath(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error determining VCS root for file %q: %s", f, err)
				continue
			} else {
				sections = pkg.Multistring{root}
			}
		}
		// check if input argument is a directory
		info, err := os.Stat(f)
		if err == nil && info.IsDir() {
			err := filepath.Walk(f, pkg.Visit(writeFile, listDiffFiles, sections, compiledPattern))
			if err != nil {
				log.Println(err)
			}
			continue
		}

		if err := pkg.Process(f,
			info.Name(),
			writeFile,
			listDiffFiles,
			sections,
			compiledPattern); err != nil {
			fmt.Fprintf(os.Stderr, "error while proccessing file %q: %s", f, err)
		}
	}
}

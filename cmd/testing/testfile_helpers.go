package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ------------------------------------------------------------------------------------------------
// use Glob to list all the files in the test dir.
// For now we ignore sub-dirs, but in the future I want to make it recursive so I can better
// organise tests. Right now I just want to get it working.
//
// TODO: Add a flag to make it also consider sub-dirs so test can be grouped better.
func findAllTestFiles(patterns []string) ([]string, error) {
	var files []string
	found := make(map[string]bool)

	for _, pattern := range patterns {
		var matches []string
		if strings.ContainsAny(pattern, "*?[") {
			var err error
			matches, err = filepath.Glob(pattern)
			if err != nil {
				return nil, fmt.Errorf("glob pattern %q: %w", pattern, err)
			}
		} else {
			matches = []string{pattern}
		}

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				return nil, fmt.Errorf("stat %q: %w", match, err)
			}

			if info.IsDir() {
				entries, err := os.ReadDir(match)
				if err != nil {
					return nil, fmt.Errorf("read directory %q: %w", match, err)
				}
				for _, entry := range entries {
					if isNotOfInterest(entry) {
						continue
					}
					if hasAcceptableSuffix(entry.Name()) {
						p := filepath.Join(match, entry.Name())
						if !found[p] {
							found[p] = true
							files = append(files, p)
						}
					}
				}
			} else {
				if !found[match] {
					found[match] = true
					files = append(files, match)
				}
			}
		}
	}

	if len(files) == 0 {
		return nil, errors.New("no test specification files found")
	}

	sort.Strings(files)
	return files, nil
}

// ------------------------------------------------------------------------------------------------
// Don't care about dirs or hidden files
func isNotOfInterest(entry os.DirEntry) bool {
	return entry.IsDir() || strings.HasPrefix(entry.Name(), ".")
}

// ------------------------------------------------------------------------------------------------
// Accepts .txt or .spec files
func hasAcceptableSuffix(name string) bool {
	return strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".spec")
}

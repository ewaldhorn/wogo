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
// findAllTestFiles scans sub-dirs to find all test files.
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
				err := filepath.WalkDir(match, func(p string, d os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if d.IsDir() {
						if strings.HasPrefix(d.Name(), ".") && p != match {
							return filepath.SkipDir
						}
						return nil
					}
					if isTestFile(d.Name()) && !found[p] {
						found[p] = true
						files = append(files, p)
					}
					return nil
				})
				if err != nil {
					return nil, fmt.Errorf("walk directory %q: %w", match, err)
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
// Accepts .txt or .spec files
func isTestFile(name string) bool {
	return strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".spec")
}

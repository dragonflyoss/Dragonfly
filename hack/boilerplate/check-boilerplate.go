/*
 * Copyright The Dragonfly Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var (
	start       = " * Copyright "
	end         = " * limitations under the License."
	boilerplate = []string{
		` * Copyright The Dragonfly Authors.`,
		` *`,
		` * Licensed under the Apache License, Version 2.0 (the "License");`,
		` * you may not use this file except in compliance with the License.`,
		` * You may obtain a copy of the License at`,
		` *`,
		` *      http://www.apache.org/licenses/LICENSE-2.0`,
		` *`,
		` * Unless required by applicable law or agreed to in writing, software`,
		` * distributed under the License is distributed on an "AS IS" BASIS,`,
		` * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.`,
		` * See the License for the specific language governing permissions and`,
		` * limitations under the License.`,
	}
)

// checkBoilerplate checks if the input string contains the boilerplate.
func checkBoilerplate(content string) error {
	// ignore generated files
	if strings.Contains(content, "DO NOT EDIT") {
		return nil
	}

	index := 0
	foundStart := false
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// find the start of the boilerplate
		bpLine := boilerplate[index]
		if strings.Contains(line, start) {
			foundStart = true
		}

		// match line by line
		if foundStart {
			if line != bpLine {
				return fmt.Errorf("boilerplate line %d does not match\nexpected: %q\ngot: %q", index+1, bpLine, line)
			}
			index++
			// exit after the last line is found
			if strings.Index(line, end) == 0 {
				break
			}
		}
	}

	if !foundStart {
		return fmt.Errorf("the file is missing a boilerplate")
	}
	if index < len(boilerplate) {
		return fmt.Errorf("boilerplate has missing lines")
	}
	return nil
}

// verifyFile verifies if a file contains the boilerplate.
func verifyFile(filePath string) error {
	if len(filePath) == 0 {
		return fmt.Errorf("empty file name")
	}

	// check file extension is go
	idx := strings.LastIndex(filePath, ".")
	if idx == -1 {
		return nil
	}

	// check if the file has a supported extension
	ext := filePath[idx : idx+len(filePath)-idx]
	if ext != ".go" {
		return nil
	}

	// read the file
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	return checkBoilerplate(string(b))
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run check-boilerplate.go <path-to-file> <path-to-file> ...")
		os.Exit(1)
	}

	for _, filePath := range os.Args[1:] {
		if err := verifyFile(filePath); err != nil {
			fmt.Fprintf(os.Stderr, "error validating %q: %v\n", filePath, err)
		}
	}
}

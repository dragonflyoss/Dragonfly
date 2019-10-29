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

package digest

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// Sha256 returns the SHA-256 checksum of the data.
func Sha256(value string) string {
	h := sha256.New()
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil))
}

// Sha1 returns the SHA-1 checksum of the contents.
func Sha1(contents []string) string {
	h := sha1.New()
	for _, content := range contents {
		if _, err := io.WriteString(h, content); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write string to io.Write: %v", err)
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

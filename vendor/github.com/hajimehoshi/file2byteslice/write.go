// Copyright 2020 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package file2byteslice

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

// Write writes a Go byte slice literal to w from the bytes of r.
func Write(w io.Writer, r io.Reader, compress bool, buildTags string, packageName string, varName string) error {
	if compress {
		compressed := &bytes.Buffer{}
		cw, err := gzip.NewWriterLevel(compressed, gzip.BestCompression)
		if err != nil {
			return err
		}
		if _, err := io.Copy(cw, r); err != nil {
			return err
		}
		cw.Close()
		r = compressed
	}

	bs, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "// Code generated by file2byteslice. DO NOT EDIT."); err != nil {
		return err
	}
	if buildTags != "" {
		if _, err := fmt.Fprintln(w, "\n//go:build "+strings.Join(strings.Split(buildTags, ","), " && ")); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "// +build "+buildTags); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "package "+packageName); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "var %s = []byte(%q)\n", varName, string(bs)); err != nil {
		return err
	}
	return nil
}

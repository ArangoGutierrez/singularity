// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

func init() {
	if err := removeLines("config_gen.go", 12, 54); err != nil {
		fmt.Println(err)
	}

}
func removeLines(fn string, start, n int) (err error) {
	if start < 1 {
		return errors.New("invalid request.  line numbers start at 1")
	}
	if n < 0 {
		return errors.New("invalid request.  negative number to remove")
	}
	var f *os.File
	if f, err = os.OpenFile(fn, os.O_RDWR, 0); err != nil {
		return
	}
	defer func() {
		if cErr := f.Close(); err == nil {
			err = cErr
		}
	}()
	var b []byte
	if b, err = ioutil.ReadAll(f); err != nil {
		return
	}
	cut, ok := skip(b, start-1)
	if !ok {
		return fmt.Errorf("less than %d lines", start)
	}
	if n == 0 {
		return nil
	}
	tail, ok := skip(cut, n)
	if !ok {
		return fmt.Errorf("less than %d lines after line %d", n, start)
	}
	t := int64(len(b) - len(cut))
	if err = f.Truncate(t); err != nil {
		return
	}
	if len(tail) > 0 {
		_, err = f.WriteAt(tail, t)
	}
	return
}

func skip(b []byte, n int) ([]byte, bool) {
	for ; n > 0; n-- {
		if len(b) == 0 {
			return nil, false
		}
		x := bytes.IndexByte(b, '\n')
		if x < 0 {
			x = len(b)
		} else {
			x++
		}
		b = b[x:]
	}
	return b, true
}

func ParseLine(s string) (d Define) {
	d = Define{
		words: strings.Fields(s),
	}

	return
}

// Define is a struct that contains one line of configuration words.
type Define struct {
	words []string
}

// WriteLine writes a line of configuration.
func (d Define) WriteLine() (s string) {
	s = "const " + d.words[1] + " = " + d.words[2]
	fmt.Println(s)
	if len(d.words) > 3 {
	}

	for _, w := range d.words[3:] {
		s += " + " + w
	}
	return s
}

var confgenTemplate = template.Must(template.New("").Parse(`// Code generated by go generate; DO NOT EDIT.
package buildcfg
{{ range $i, $d := . }}
{{$d.WriteLine -}}
{{end}}
`))

func main() {
	outFile, err := os.Create("config.go")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer outFile.Close()

	inFile, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	header := []Define{}
	s := bufio.NewScanner(bytes.NewReader(inFile))
	for s.Scan() {
		header = append(header, parseLine(s.Text()))
	}

	confgenTemplate.Execute(outFile, header)
}

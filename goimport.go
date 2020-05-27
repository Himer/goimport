package main

import (
	"bufio"
	"bytes"
	"fmt"
	"golang.org/x/tools/imports"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var options = &imports.Options{
	TabWidth:  8,
	TabIndent: true,
	Comments:  true,
	Fragment:  true,
}

/*
  delete the blank line in imports
*/
func removeBlankLine(r io.Reader) ([]byte, error) {
	var out bytes.Buffer
	in := bufio.NewReader(r)
	inImports := false
	done := false
	for {
		s, err := in.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if !inImports && !done && strings.HasPrefix(s, "import") {
			inImports = true
		}
		if inImports && (strings.HasPrefix(s, "var") ||
			strings.HasPrefix(s, "func") ||
			strings.HasPrefix(s, "const") ||
			strings.HasPrefix(s, "type")) {
			done = true
			inImports = false
		}
		if inImports && s == "\n" {
			continue
		}
		_, _ = fmt.Fprint(&out, s)
	}
	return out.Bytes(), nil
}

/*
   fix the imports
*/
func processFile(filename string) error {
	fd, err := os.OpenFile(filename, os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() {
		_ = fd.Close()
	}()
	src, err := ioutil.ReadAll(fd)
	if err != nil {
		return err
	}
	remove, err := removeBlankLine(bytes.NewReader(src))
	if err != nil {
		return err
	}
	res, err := imports.Process(filename, remove, options)
	if err != nil {
		return err
	}
	if bytes.Equal(src, res) {
		return nil
	}
	if err = fd.Truncate(0); err != nil {
		return err
	}
	if _, err = fd.Seek(0, 0); err != nil {
		return err
	}
	if _, err = fd.Write(res); err != nil {
		return err
	}
	log.Println("format:", filename, "success")
	return nil
}

/*
	determine if it is a golang file
*/
func isGolangFile(filename string) bool {
	if !strings.HasSuffix(filename, ".go") {
		return false
	}
	fi, err := os.Stat(filename)

	return err == nil && !fi.IsDir() && fi.Mode().IsRegular()
}

func main() {

	var (
		exclude = kingpin.Flag("exclude-dir", "exclude dir or filename,eg: vendor,test.").Short('e').Default("vendor").String()
		workDir = kingpin.Arg("dir", "format dir.").Required().String()
	)

	kingpin.Parse()

	excludes := map[string]interface{}{}

	for _, item := range strings.Split(*exclude, ",") {
		if item := strings.TrimSpace(item); item != "" {
			excludes[item] = nil
		}
	}

	absPath, err := filepath.Abs(*workDir)
	if err != nil {
		log.Printf("get abs path for %s fail,%s", *workDir, err)
		os.Exit(-1)
	}
	err = filepath.Walk(*workDir, func(filename string, info os.FileInfo, err error) error {
		if !isGolangFile(filename) {
			return nil
		}

		for item, _ := range excludes {
			if strings.Contains(strings.TrimPrefix(filename, absPath), item) {
				fmt.Println(filename)
				return nil
			}
		}
		return processFile(filename)
	})
	if err != nil {
		log.Printf("fail on doing format,%s", err)
		os.Exit(-1)
	}
}

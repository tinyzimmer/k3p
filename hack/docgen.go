package main

import (
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra/doc"

	"github.com/tinyzimmer/k3p/pkg/cmd"
	"github.com/tinyzimmer/k3p/pkg/log"
)

func main() {
	if err := genMarkdownDocs(); err != nil {
		log.Fatal(err)
	}
}

func genMarkdownDocs() error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	username := u.Username

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := doc.GenMarkdownTree(cmd.GetRootCommand(), tmpDir); err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll("doc", 0755); err != nil {
		log.Fatal(err)
	}

	return filepath.Walk(tmpDir, func(file string, fileInfo os.FileInfo, lastErr error) error {
		if lastErr != nil {
			return lastErr
		}
		if fileInfo.IsDir() {
			return nil
		}
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		sanitized := strings.Replace(string(data), username, "<user>", -1)
		if err := ioutil.WriteFile(path.Join("doc", strings.TrimPrefix(file, tmpDir+"/")), []byte(sanitized), 0644); err != nil {
			return err
		}
		return nil
	})

}

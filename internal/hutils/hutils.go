package hutils

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"
)

const (
	CompactDateTimeFormat = "060102150405"
)

func FindDirectory(dir string, alt string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("%s cannot be empty", alt)
	} else if stat, err := os.Stat(dir); err != nil {
		return err
	} else if !stat.IsDir() {
		return fmt.Errorf("%s is a not a directory", dir)
	} else {
		return nil
	}
}

func PackageFromDirectory(dir string) (string, string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", "", err
	}

	var pkgName string
	switch n := len(pkgs); n {
	case 0:
		pkgName = filepath.Base(dir)
		var buf bytes.Buffer
		for _, r := range pkgName {
			if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' {
				_, _ = buf.WriteRune(r)
				continue
			}
			_ = buf.WriteByte('_')
		}
		pkgName = strings.Trim(buf.String(), "_")
		if pkgName == "" {
			return "", "", fmt.Errorf("failed to convert %s to a package name", dir)
		}
	case 1:
		for _, pkg := range pkgs {
			if pkg.Name == "" {
				return "", "", fmt.Errorf("empty package name. dir: %s", dir)
			}
			pkgName = pkg.Name
		}
	default:
		return "", "", fmt.Errorf("%q contains %d packages", dir, n)
	}

	var a1 []string
	for absDir != "" {
		f := filepath.Join(absDir, "go.mod")
		if _, err := os.Stat(f); err != nil {
			a1 = append(a1, filepath.Base(absDir))
			absDir = filepath.Dir(absDir)
			continue
		}

		data, err := ioutil.ReadFile(f)
		if err != nil {
			return "", "", err
		}
		mod, err := modfile.ParseLax("go.mod", data, nil)
		if err != nil {
			return "", "", err
		}
		if mod.Module == nil {
			return "", "", errors.New("failed to parse go.mod [1]")
		}
		if mod.Module.Mod.Path == "" {
			return "", "", errors.New("failed to parse go.mod [2]")
		}

		for i, j := 0, len(a1)-1; i < j; i, j = i+1, j-1 {
			a1[i], a1[j] = a1[j], a1[i]
		}
		a2 := []string{mod.Module.Mod.Path}
		a3 := append(a2, a1...)
		return pkgName, path.Join(a3...), nil
	}

	return "", "", errors.New("cannot find go.mod")
}

func Gofmt(file string) error {
	cmd := exec.Command("gofmt", "-w", file)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func BuildPlugin(t *testing.T, hotswap string, pluginDir, outputDir string, args ...string) {
	t.Helper()
	a := []string{"build", pluginDir, outputDir}
	cmd := exec.Command(hotswap, append(a, args...)...)
	cmd.Stderr = os.Stderr
	if testing.Verbose() {
		cmd.Stdout = os.Stdout
	}
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}

func Join(a ...string) string {
	switch n := len(a); n {
	case 0:
		return ""
	case 1:
		return a[0]
	case 2:
		return fmt.Sprintf("%s and %s", a[0], a[1])
	default:
		return fmt.Sprintf("%s and %s", strings.Join(a[:n-1], ", "), a[n-1])
	}
}

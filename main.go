package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/icholy/gomajor/internal/importpaths"
	"github.com/icholy/gomajor/internal/latest"
	"github.com/icholy/gomajor/internal/packages"
)

func main() {
	if err := get(); err != nil {
		log.Fatal(err)
	}
}

func get() error {
	var rewrite, goget bool
	flag.BoolVar(&rewrite, "rewrite", true, "rewrite import paths")
	flag.BoolVar(&goget, "get", true, "run go get")
	flag.Parse()
	if flag.NArg() != 1 {
		return fmt.Errorf("missing package spec")
	}
	// figure out the correct import path
	pkgpath, version := packages.SplitSpec(flag.Arg(0))
	pkg, err := packages.Load(pkgpath)
	if err != nil {
		return err
	}
	// figure out what version to get
	if version == "latest" {
		version, err = latest.Version(pkg.ModPrefix)
		if err != nil {
			return err
		}
	}
	if version != "" && !semver.IsValid(version) {
		return fmt.Errorf("invalid version: %s", version)
	}
	// go get
	if goget {
		spec := pkg.Path(version)
		if version != "" {
			spec += "@" + semver.Canonical(version)
		}
		fmt.Println("go get", spec)
		cmd := exec.Command("go", "get", spec)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	// rewrite imports
	if !rewrite {
		return nil
	}
	return importpaths.Rewrite(".", func(name, path string) (string, bool) {
		modpath, ok := pkg.FindModPath(path)
		if !ok {
			return "", false
		}
		pkgdir := strings.TrimPrefix(path, modpath)
		pkgdir = strings.TrimPrefix(pkgdir, "/")
		if pkg.PkgDir != "" && pkg.PkgDir != pkgdir {
			return "", false
		}
		newpath := packages.Package{
			PkgDir:    pkgdir,
			ModPrefix: pkg.ModPrefix,
		}.Path(version)
		if newpath == path {
			return "", false
		}
		fmt.Printf("%s: %s -> %s\n", name, path, newpath)
		return newpath, true
	})
}

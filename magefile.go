// +build mage

package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	golangciLintURL = "https://github.com/golangci/golangci-lint/releases/download/v1.24.0/golangci-lint-1.24.0-%s-%s"
	gotestsumURL    = "https://github.com/gotestyourself/gotestsum/releases/download/v0.4.1/gotestsum_0.4.1_%s_%s.tar.gz"
)

var (
	// Default mage target
	Default = All

	projectName = "convcom"

	goexe = "go"

	// tools
	golangcilint = filepath.Join("bin", "golangci-lint")
	gotestsum    = filepath.Join("bin", "gotestsum")

	// commands
	gobuild         = sh.RunCmd(goexe, "build")
	gofmt           = sh.RunCmd(goexe, "fmt")
	golangcilintCmd = sh.RunCmd(golangcilint, "run")
	gotestsumCmd    = sh.RunCmd(gotestsum, "--")
	govet           = sh.RunCmd(goexe, "vet")
	rm              = sh.RunCmd("rm", "-f")
)

func init() {
	// Force use of go modules
	os.Setenv("GO111MODULES", "on")

	if runtime.GOOS == "windows" {
		golangcilint += ".exe"
		golangcilintCmd = sh.RunCmd(golangcilint, "run")
		gotestsum += ".exe"
		gotestsumCmd = sh.RunCmd(gotestsum, "--")
	}
}

// All runs format, lint, vet, build, and test targets
func All(ctx context.Context) {
	mg.SerialCtxDeps(ctx, Format, Lint, Vet, Build, Test)
}

// Benchmark runs the benchmark suite
func Benchmark(ctx context.Context) error {
	return runTests("-run=__absolutelynothing__", "-bench")
}

// Build runs go build
func Build(ctx context.Context) error {
	say("building " + projectName)
	return gobuild("./...")
}

// Clean removes generated files
func Clean(ctx context.Context) error {
	say("cleaning files")
	return rm("-r", "bin", "converage")
}

// Coverage generates coverage reports
func Coverage(ctx context.Context) error {
	mg.CtxDeps(ctx, getGotestsum, coverageDir)

	mode := os.Getenv("COVERAGE_MODE")
	if mode == "" {
		mode = "atomic"
	}
	if err := runTests("-cover", "-covermode", mode, "-coverprofile=coverage/cover.out"); err != nil {
		return err
	}
	if err := sh.Run(goexe, "tool", "cover", "-html=coverage/cover.out", "-o", "coverage/index.html"); err != nil {
		return err
	}
	return nil
}

// Format runs go fmt
func Format(ctx context.Context) error {
	say("running go fmt")
	return gofmt("./...")
}

// Lint runs golangci-lint
func Lint(ctx context.Context) error {
	mg.CtxDeps(ctx, getGolangcilint)
	say("running " + golangcilint)
	return golangcilintCmd()
}

// Test runs the test suite
func Test(ctx context.Context) error {
	mg.CtxDeps(ctx, getGotestsum)
	say("running tests")
	return runTests()
}

// TestRace runs the test suite with race detection
func TestRace(ctx context.Context) error {
	mg.CtxDeps(ctx, getGotestsum)
	say("running race condition tests")
	return runTests("-race")
}

// TestShort runs only tests marked as short
func TestShort(ctx context.Context) error {
	mg.CtxDeps(ctx, getGotestsum)
	say("running short tests")
	return runTests("-short")
}

// Vet runs go vet
func Vet(ctx context.Context) error {
	say("running go vet")
	return govet("./...")
}

func binDir(ctx context.Context) error {
	_, err := os.Stat("bin")
	if os.IsNotExist(err) {
		return os.Mkdir("bin", 0755)
	}
	return err
}

func coverageDir(ctx context.Context) error {
	_, err := os.Stat("coverage")
	if os.IsNotExist(err) {
		return os.Mkdir("coverage", 0755)
	}
	return err
}

func getFileInTarball(ctx context.Context, path, u string) error {
	r, err := getURL(ctx, u)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()
	}()
	gz, err := gzip.NewReader(r.Body)
	if err != nil {
		return err
	}

	tgz := tar.NewReader(gz)
	if err != nil {
		return err
	}

	for {
		hdr, err := tgz.Next()
		if err == io.EOF {
			break // end of archvie
		}
		if err != nil {
			return err
		}
		file := filepath.Base(path)
		if strings.HasSuffix(hdr.Name, file) {
			f, err := os.Create(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(f, tgz); err != nil {
				return err
			}
			return os.Chmod(path, 0700)
		}
	}
	return nil
}

func getFileInZip(ctx context.Context, path, u string) error {
	r, err := getURL(ctx, u)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()
	}()

	f, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, r.Body); err != nil {
		return err
	}
	f.Close()

	z, err := zip.OpenReader(f.Name())
	if err != nil {
		return err
	}

	file := filepath.Base(path)
	for _, zf := range z.File {
		fmt.Println("file: ", zf.Name)
		if strings.HasSuffix(zf.Name, file) {
			inFile, err := zf.Open()
			if err != nil {
				return err
			}
			defer inFile.Close()
			outFile, err := os.Create(path)
			if err != nil {
				return err
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, inFile); err != nil {
				return err
			}
			return os.Chmod(path, 0755)
		}
	}

	return nil
}

func getGolangcilint(ctx context.Context) error {
	mg.CtxDeps(ctx, binDir)
	_, err := os.Stat(golangcilint)
	if os.IsNotExist(err) {
		if runtime.GOOS != "windows" {
			return getFileInTarball(ctx, golangcilint, fmt.Sprintf(golangciLintURL+".tar.gz", runtime.GOOS, runtime.GOARCH))
		}
		return getFileInZip(ctx, golangcilint, fmt.Sprintf(golangciLintURL+".zip", runtime.GOOS, runtime.GOARCH))
	}
	return err
}

func getGotestsum(ctx context.Context) error {
	mg.CtxDeps(ctx, binDir)
	_, err := os.Stat(gotestsum)
	if os.IsNotExist(err) {
		return getFileInTarball(ctx, gotestsum, fmt.Sprintf(gotestsumURL, runtime.GOOS, runtime.GOARCH))
	}
	return err
}

func getURL(ctx context.Context, u string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error %s [%s]: %s", req.URL, req.Method, resp.Status)
	}
	return resp, nil
}

func runTests(testType ...string) error {
	testType = append(testType, "./...")
	return gotestsumCmd(testType...)
}

func say(format string, args ...interface{}) (int, error) {
	format = strings.TrimSpace(format)
	return fmt.Printf("▶ "+format+"…\n", args...)
}

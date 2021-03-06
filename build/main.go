/*
 * Copyright (c) 2018 Filecoin Project
 * Copyright (c) 2022 https://github.com/siegfried415
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); 
 * you may not use this file except in compliance with the License. 
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
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
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/siegfried415/go-crawling-bazaar/utils/version"
)

var lineBreak = "\n"

func init() {
	log.SetFlags(0)
	if runtime.GOOS == "windows" {
		lineBreak = "\r\n"
	}
	// We build with go modules.
	if err := os.Setenv("GO111MODULE", "on"); err != nil {
		fmt.Println("Failed to set GO111MODULE env")
		os.Exit(1)
	}
}

// command is a structure representing a shell command to be run in the
// specified directory
type command struct {
	dir   string
	parts []string
}

// cmd creates a new command using the pwd and its cwd
func cmd(parts ...string) command {
	return cmdWithDir("./", parts...)
}

// cmdWithDir creates a new command using the specified directory as its cwd
func cmdWithDir(dir string, parts ...string) command {
	return command{
		dir:   dir,
		parts: parts,
	}
}

func runCmd(c command) {
	parts := c.parts
	if len(parts) == 1 {
		parts = strings.Split(parts[0], " ")
	}

	name := strings.Join(parts, " ")
	cmd := exec.Command(parts[0], parts[1:]...) // #nosec
	cmd.Dir = c.dir
	log.Println(name)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if _, err = io.Copy(os.Stderr, stderr); err != nil {
			panic(err)
		}
	}()
	go func() {
		defer wg.Done()
		if _, err = io.Copy(os.Stdout, stdout); err != nil {
			panic(err)
		}
	}()

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	wg.Wait()
	if err := cmd.Wait(); err != nil {
		log.Fatalf("Command '%s' failed: %s\n", name, err)
	}
}

func runCapture(name string) string {
	args := strings.Split(name, " ")
	cmd := exec.Command(args[0], args[1:]...) // #nosec
	log.Println(name)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Command '%s' failed: %s\n", name, err)
	}

	return strings.Trim(string(output), lineBreak)
}

// deps installs all dependencies
func deps() {
	runCmd(cmd("pkg-config --version"))

	log.Println("Installing dependencies...")

	cmds := []command{
		// Download all go modules. While not strictly necessary (go
		// will do this automatically), this:
		//  1. Makes it easier to cache dependencies in CI.
		//  2. Makes it possible to fetch all deps ahead of time for
		//     offline development.
		cmd("go mod download"),
		// Download and build proofs.

		//cmd("./scripts/install-go-bls-sigs.sh"),
		//cmd("./scripts/install-go-sectorbuilder.sh"),

		cmd("./scripts/install-filecoin-parameters.sh"),
	}

	for _, c := range cmds {
		runCmd(c)
	}
}


/* 
// lint runs linting using golangci-lint
func lint(packages ...string) {
	if len(packages) == 0 {
		packages = []string{"./..."}
	}

	log.Printf("Linting %s ...\n", strings.Join(packages, " "))

	runCmd(cmd("go", "run", "github.com/golangci/golangci-lint/cmd/golangci-lint", "run"))
}
*/

func build() {
	buildGcb()
	buildGengen()
	buildFaucet()
	buildGenesisFileServer()
	generateGenesis()
	buildMigrations()
	buildPrereleaseTool()
}

func forcebuild() {
	forceBuildFC()
	buildGengen()
	buildFaucet()
	buildGenesisFileServer()
	generateGenesis()
	buildMigrations()
	buildPrereleaseTool()
}

func forceBuildFC() {
	log.Println("Force building go-crawling-bazaar...")

	runCmd(cmd([]string{
		"go", "build",
		"-ldflags", fmt.Sprintf("-X github.com/siegfried415/go-crawling-bazaar/flags.Commit=%s", getCommitSha()),
		"-a", "-v", "-o", "gcd", ".",
	}...))
}

// cleanDirectory removes the child of a directly wihtout removing the directory itself, unlike `RemoveAll`.
// There is also an additional parameter to ignore dot files which is important for directories which are normally
// empty. Git has no concept of directories, so for a directory to automatically be created on checkout, a file must
// exist in side of it. We use this pattern in a few places, so the need to keep the dot files around is impotant.
func cleanDirectory(dir string, ignoredots bool) error {
	if abs := filepath.IsAbs(dir); !abs {
		return fmt.Errorf("Directory %s is not an absolute path, could not clean directory", dir)
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		fname := file.Name()
		if ignoredots && []rune(fname)[0] == '.' {
			continue
		}

		fpath := filepath.Join(dir, fname)

		fmt.Println("Removing", fpath)
		if err := os.RemoveAll(fpath); err != nil {
			return err
		}
	}

	return nil
}

func generateGenesis() {
	log.Println("Generating genesis...")

	liveFixtures, err := filepath.Abs("./fixtures/live")
	if err != nil {
		panic(err)
	}

	if err := cleanDirectory(liveFixtures, true); err != nil {
		panic(err)
	}

	runCmd(cmd([]string{
		"./gengen/gengen",
		"--keypath", liveFixtures,
		"--out-car", filepath.Join(liveFixtures, "genesis.car"),
		"--out-json", filepath.Join(liveFixtures, "gen.json"),
		"--config", "./fixtures/setup.json",
	}...))

	testFixtures, err := filepath.Abs("./fixtures/test")
	if err != nil {
		panic(err)
	}

	if err := cleanDirectory(testFixtures, true); err != nil {
		panic(err)
	}

	runCmd(cmd([]string{
		"./gengen/gengen",
		"--keypath", testFixtures,
		"--out-car", filepath.Join(testFixtures, "genesis.car"),
		"--out-json", filepath.Join(testFixtures, "gen.json"),
		"--config", "./fixtures/setup.json",
		"--test-proofs-mode",
	}...))
}

func buildGcb() {
	log.Println("Building gcb...")

	runCmd(cmd([]string{
		"go", "build",
		"-ldflags", fmt.Sprintf("-X github.com/siegfried415/go-crawling-bazaar/flags.Version=%s", getVersion()),
		"-v", "-o", "bin/gcb", ".",
	}...))
}

func buildGengen() {
	log.Println("Building gengen utils...")

	runCmd(cmd([]string{"go", "build", "-o", "./gengen/gengen", "./gengen"}...))
}

func buildFaucet() {
	log.Println("Building faucet...")

	runCmd(cmd([]string{"go", "build", "-o", "./tools/faucet/faucet", "./tools/faucet/"}...))
}

func buildGenesisFileServer() {
	log.Println("Building genesis file server...")

	runCmd(cmd([]string{"go", "build", "-o", "./tools/genesis-file-server/genesis-file-server", "./tools/genesis-file-server/"}...))
}

func buildMigrations() {
	log.Println("Building migrations...")
	runCmd(cmd([]string{
		"go", "build", "-o", "./tools/migration/go-crawling-bazaar-migrate", "./tools/migration/main.go"}...))
}

func buildPrereleaseTool() {
	log.Println("Building prerelease-tool...")

	runCmd(cmd([]string{"go", "build", "-o", "./tools/prerelease-tool/prerelease-tool", "./tools/prerelease-tool/"}...))
}

func install() {
	log.Println("Installing...")

	runCmd(cmd("go", "install", "-ldflags", fmt.Sprintf("-X github.com/siegfried415/go-crawling-bazaar/flags.Commit=%s", getCommitSha())))
}

// test executes tests and passes along all additional arguments to `go test`.
func test(userArgs ...string) {
	log.Println("Running tests...")

	// Consult environment for test packages, in order to support CI container-level parallelism.
	packages, ok := os.LookupEnv("TEST_PACKAGES")
	if !ok {
		packages = "./..."
	}

	begin := time.Now()
	runCmd(cmd(fmt.Sprintf("go test %s %s",
		strings.Replace(packages, "\n", " ", -1), strings.Join(userArgs, " "))))
	end := time.Now()
	log.Printf("Tests finished in %.1f seconds\n", end.Sub(begin).Seconds())
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		log.Fatalf("Missing command")
	}

	if !version.Check(runtime.Version()) {
		log.Fatalf("Invalid go version: %s", runtime.Version())
	}

	cmd := args[0]

	switch cmd {
	case "deps", "smartdeps":
		deps()

	//case "lint":
	//	lint(args[1:]...)

	case "build-gcb":
		buildGcb()

	case "build-gengen":
		buildGengen()
	case "generate-genesis":
		generateGenesis()
	case "build-migrations":
		buildMigrations()
	case "build":
		build()
	case "fbuild":
		forcebuild()
	case "test":
		test(args[1:]...)
	case "install":
		install()
	case "best":
		build()
		test(args[1:]...)
	case "all":
		deps()
		
		//lint()

		build()
		test(args[1:]...)
	default:
		log.Fatalf("Unknown command: %s\n", cmd)
	}
}

func getCommitSha() string {
	return runCapture("git log -n 1 --format=%H")
}

func getVersion() string {
 	return runCapture("git describe --abbrev --abbrev=0 --tags" ) 
}

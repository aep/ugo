package main

import (
	"fmt"
	"github.com/segmentio/go-prompt"
	"gopkg.in/alecthomas/kingpin.v2"
	git "gopkg.in/src-d/go-git.v4"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

const gopkgFilename = ".gopackage"

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getParentPath(pathstr string) string {
	parentDir, _ := path.Split(strings.TrimSuffix(pathstr, "/"))
	return parentDir
}

func findGopackagePath() (string, error) {
	currentDir, err := os.Getwd()

	if err != nil {
		return "", err
	}

	for ; currentDir != ""; currentDir = getParentPath(currentDir) {
		if pathExists(path.Join(currentDir, gopkgFilename)) {
			return currentDir, nil
		}
	}

	return "", fmt.Errorf("could not find a %s in any parent directory up to the root directory", gopkgFilename)
}

func getGopackage(gopackagePath string) (string, error) {
	content, err := ioutil.ReadFile(path.Join(gopackagePath, gopkgFilename))

	if err != nil {
		return "", err
	}

	return strings.Replace(string(content), "\n", "", 1), nil
}

func gitURLToPackageName(urlStr string) (string, error) {
	urlStr = strings.Replace(urlStr, ".git", "", 1)

	if strings.HasPrefix(urlStr, "git@") {
		urlStr = strings.Replace(strings.TrimPrefix(urlStr, "git@"), ":", "/", 1)
	}

	url, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	return url.Hostname() + url.Path, nil
}

func getPossiblePackageNames() (pkgsToPath map[string]string, err error) {
	currentDir, err := os.Getwd()

	if err != nil {
		return nil, err
	}

	pkgsToPath = map[string]string{}
	for ; currentDir != ""; currentDir = getParentPath(currentDir) {
		repo, err := git.PlainOpen(currentDir)

		if err != nil {
			continue
		}

		conf, err := repo.Config()

		if err != nil {
			continue
		}

		for _, remote := range conf.Remotes {
			url, err := gitURLToPackageName(remote.URL)

			if err != nil {
				continue
			}

			pkgsToPath[url] = currentDir
		}
	}

	return pkgsToPath, nil
}

func createGopackageFile() (string, error) {
	pkgsToPath, err := getPossiblePackageNames()

	if err != nil {
		return "", err
	}

	for pkgName, pkgPath := range pkgsToPath {
		if prompt.Confirm("Do you want to create a %s/%s file with the following content: %s", pkgPath, gopkgFilename, pkgName) {
			err := ioutil.WriteFile(path.Join(pkgPath, gopkgFilename), []byte(pkgName), 0644)

			return pkgPath, err
		}
	}

	if len(pkgsToPath) > 0 {
		return "", fmt.Errorf("no package selected to create %s file, please create the file manually", gopkgFilename)
	}

	return "", fmt.Errorf("no package auto detected to create %s file, please create the file manually", gopkgFilename)
}

func createWorkspace(gopkgPath string) (string, error) {
	packageName, err := getGopackage(gopkgPath)

	if err != nil {
		return "", err
	}

	basePath, dir := path.Split(packageName)

	workspacePath := path.Join(gopkgPath, ".workspace", "src", basePath)

	os.MkdirAll(workspacePath, os.ModePerm)

	symlink := path.Join(workspacePath, dir)

	if pathExists(symlink) {
		os.Remove(symlink)
	}

	os.Symlink(gopkgPath, symlink)

	return symlink, nil
}

func getEnvironment(replace bool, gopkgPath string, pkgPath string) []string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		if i := strings.Index(e, "="); i >= 0 {
			env[e[:i]] = e[i+1:]
		}
	}

	gopath := path.Join(gopkgPath, ".workspace")

	if replace || len(env["GOPATH"]) == 0 {
		env["GOPATH"] = gopath
	} else {
		env["GOPATH"] += ":" + gopath
	}

	// Set the pwd variable to the directory in the workspace
	env["PWD"] = pkgPath

	result := []string{}

	for key, value := range env {
		result = append(result, key+"="+value)
	}

	return result
}

var (
	app           = kingpin.New("ugo", "A tool for manipulating the GOPATH variable")
	replaceGopath = app.Flag("replace", "Replace existing GOPATH instead of adding.").Default("false").Short('r').Bool()
	command       = app.Arg("command", "The command that should be executed by ugo.").Required().String()
	commandArgs   = app.Arg("arguments", "The arguments for the command.").Strings()
)

func main() {
	// The flags need to be present before the positional arguments
	app.Interspersed(false)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	gopkgPath, err := findGopackagePath()

	if err != nil {
		gopkgPath, err = createGopackageFile()

		if err != nil {
			panic(err)
		}
	}

	pkgPath, err := createWorkspace(gopkgPath)

	if err != nil {
		panic(err)
	}

	os.Chdir(pkgPath)

	fullCommandPath, err := exec.LookPath(*command)

	if err != nil {
		panic(err)
	}

	err = syscall.Exec(fullCommandPath, append([]string{*command}, *commandArgs...), getEnvironment(*replaceGopath, gopkgPath, pkgPath))

	panic(err)
}

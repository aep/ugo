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

	return string(content), nil
}

func gitURLToPackageName(urlStr string) (string, error) {
	urlStr = strings.Replace(urlStr, ".git", "", 1)

	if strings.HasPrefix(urlStr, "git@") {
		return strings.Replace(strings.TrimPrefix(urlStr, "git@"), ":", "/", 1), nil
	}

	url, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	url.Scheme = ""
	return url.String(), nil

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

func getEnvironment(replace bool, gopkgPath string) []string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		if i := strings.Index(e, "="); i >= 0 {
			env[e[:i]] = e[i+1:]
		}
	}

	gopath := path.Join(gopkgPath, ".workspace")

	if replace {
		env["GOPATH"] = gopath
	} else {
		env["GOPATH"] += ":" + gopath
	}

	result := []string{}

	for key, value := range env {
		result = append(result, key+"="+value)
	}

	return result
}

var (
	replaceGopath = kingpin.Flag("replace", "Replace existing GOPATH instead of adding.").Default("false").Short('r').Bool()
	command       = kingpin.Arg("command", "The command that should be executed by ugo.").Required().String()
	commandArgs   = kingpin.Arg("arguments", "The arguments for the command.").Strings()
)

func main() {
	kingpin.Parse()

	gopkgPath, err := findGopackagePath()

	if err != nil {
		gopkgPath, err = createGopackageFile()

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	pkgPath, err := createWorkspace(gopkgPath)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cmdstr := fmt.Sprintf("cd %s && %s %s", pkgPath, *command, strings.Join(*commandArgs, " "))

	cmd := exec.Command("sh", "-c", cmdstr)
	cmd.Env = getEnvironment(*replaceGopath, gopkgPath)

	output, err := cmd.CombinedOutput()

	fmt.Printf("%s", output)

	if err != nil {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

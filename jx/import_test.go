package jx

import (
	/*
	"fmt"
	"os"
	"strings"

	*/
	"github.com/DATA-DOG/godog"
	"github.com/stretchr/testify/assert"
	"github.com/jenkins-x/godog-jenkins/utils"
	"io/ioutil"
	"os"
	"strings"
)

type importTest struct {
	Errors    *utils.ErrorSlice
	SourceDir string
	WorkDir   string
	Args      []string
}

func (o *importTest) aDirectoryContainingASpringBootApplication() error {
	tmpDir, err := ioutil.TempDir("", "test-jx-import-")
	if err != nil {
	  return err
	}
	err = os.MkdirAll(tmpDir, utils.DefaultWritePermissions)
	if err != nil {
	  return err
	}
	o.WorkDir = tmpDir
	assert.NotEmpty(o.Errors, o.SourceDir)
	assert.DirExists(o.Errors, o.SourceDir)
	err = utils.RunCommand("", "bash", "-c", "cp -r " + o.SourceDir + " " + o.WorkDir)
	if err != nil {
	  return err
	}
	assert.DirExists(o.Errors, o.WorkDir)
	return o.Errors.Error()
}

func (o *importTest) runningInThatDirectory(commandLine string) error {
	args := strings.Fields(commandLine)
	assert.NotEmpty(o.Errors, args, "not enough arguments")
	cmd := args[0]
	assert.Equal(o.Errors, "jx", cmd)
	remaining := append(args[1:], "-b")
	if len(o.Args) > 0 {
		remaining = append(remaining, o.Args...)
	}
	err := utils.RunCommand(o.WorkDir, cmd, remaining...)
	if err != nil {
	  return err
	}
	return o.Errors.Error()
}

func (o *importTest) thereShouldBeAJenkinsProjectCreate() error {
	return godog.ErrPending
}

func (o *importTest) theApplicationShouldBeBuiltAndPromotedViaCICD() error {
	return godog.ErrPending
}

func ImportFeatureContext(s *godog.Suite) {
	o := &importTest{
		Errors:    utils.CreateErrorSlice(),
		SourceDir: "../examples/example-spring-boot",
		Args:      []string{"--git-provider-url", "http://localhost:3000/"},
	}
	s.Step(`^a directory containing a Spring Boot application$`, o.aDirectoryContainingASpringBootApplication)
	s.Step(`^running "([^"]*)" in that directory$`, o.runningInThatDirectory)
	s.Step(`^there should be a jenkins project create$`, o.thereShouldBeAJenkinsProjectCreate)
	s.Step(`^the application should be built and promoted via CI \/ CD$`, o.theApplicationShouldBeBuiltAndPromotedViaCICD)
}

package jximport

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/DATA-DOG/godog"
	"github.com/jenkins-x/godog-jenkins/utils"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

type importTest struct {
	Factory     cmdutil.Factory
	Interactive bool
	Errors      *utils.ErrorSlice
	SourceDir   string
	WorkDir     string
	Args        []string
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
	err = utils.RunCommand("", "bash", "-c", "cp -r "+o.SourceDir+"/* "+o.WorkDir)
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
	gitProviderURL, err := o.gitProviderURL()
	if err != nil {
		return err
	}
	fmt.Printf("Using git provider URL %s\n", util.ColorInfo(gitProviderURL))
	remaining := append(args[1:], "-b", "--git-provider-url", gitProviderURL)
	if len(o.Args) > 0 {
		remaining = append(remaining, o.Args...)
	}
	err = utils.RunCommandInteractive(o.Interactive, o.WorkDir, cmd, remaining...)
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

func (o *importTest) gitProviderURL() (string, error) {
	gitProviderURL := os.Getenv("GIT_PROVIDER_URL")
	if gitProviderURL != "" {
		return gitProviderURL, nil
	}
	// find the default  load the default one from the current ~/.jx/jenkinsAuth.yaml
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return "", err
	}
	config := authConfigSvc.Config()
	url := config.CurrentServer
	if url != "" {
		return url, nil
	}
	servers := config.Servers
	if len(servers) == 0 {
		return "", fmt.Errorf("No servers in the ~/.jx/gitAuth.yaml file!")
	}
	return servers[0].URL, nil
}

func ImportFeatureContext(s *godog.Suite) {
	o := &importTest{
		Factory:   cmdutil.NewFactory(),
		Interactive: os.Getenv("JX_INTERACTIVE") == "true",
		Errors:    utils.CreateErrorSlice(),
		SourceDir: "../../examples/example-spring-boot",
		Args:      []string{},
	}
	s.Step(`^a directory containing a Spring Boot application$`, o.aDirectoryContainingASpringBootApplication)
	s.Step(`^running "([^"]*)" in that directory$`, o.runningInThatDirectory)
	s.Step(`^there should be a jenkins project create$`, o.thereShouldBeAJenkinsProjectCreate)
	s.Step(`^the application should be built and promoted via CI \/ CD$`, o.theApplicationShouldBeBuiltAndPromotedViaCICD)
}

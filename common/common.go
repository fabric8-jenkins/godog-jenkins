package common

import (
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/jenkins-x/godog-jenkins/jenkins"
	"github.com/jenkins-x/godog-jenkins/utils"
	"github.com/jenkins-x/golang-jenkins"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)


type CommonTest struct {
	Factory       cmdutil.Factory
	JenkinsClient *gojenkins.Jenkins
	Interactive   bool
	Errors        *utils.ErrorSlice
	WorkDir       string
}

// TheApplicationShouldBeBuiltAndPromotedViaCICD asserts that the project
// should be created in Jenkins and that the build should complete successfully
func (o *CommonTest) TheApplicationShouldBeBuiltAndPromotedViaCICD() error {
	_, folderName := filepath.Split(o.WorkDir)
	// TODO hard coded :) - replace with code to use the Jenkins Auth
	// via the o.gitProviderURL()
	userName := "jstrachan"
	jobName := userName + "/" + folderName + "/master"
	if o.JenkinsClient == nil {
		client, err := o.Factory.GetJenkinsClient()
		if err != nil {
		  return err
		}
		o.JenkinsClient = client
	}
	fmt.Printf("Checking that there is a job built successfully for %s\n", jobName)
	return jenkins.ThereShouldBeAJobThatCompletesSuccessfully(jobName, o.JenkinsClient)
}

// GitProviderURL returns the git provider to use
func (o *CommonTest) GitProviderURL() (string, error) {
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
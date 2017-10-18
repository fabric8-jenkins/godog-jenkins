package jenkins

import (
	"fmt"
	"github.com/DATA-DOG/godog"
	"github.com/fabric8-jenkins/godog-jenkins/utils"
	"github.com/fabric8-jenkins/golang-jenkins"
	"net/url"
	"github.com/fabric8-jenkins/godog-jenkins/github"
)

type importFeature struct {
	job gojenkins.Job
}

func (f *importFeature) thereIsAFabricImportJob(arg int) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client %v", err)
	}

	jobName := "fabric8-import"
	f.job, err = jenkins.GetJob(jobName)
	if err != nil {
		jobXML, err := utils.GetFileAsString("resources/import_job.xml")
		if err != nil {
			return err
		}

		err = jenkins.CreateJobWithXML(jobXML, jobName)
		if err != nil {
			return fmt.Errorf("error creating Job %v", err)
		}
		f.job, err = jenkins.GetJob(jobName)
		if err != nil {
			return fmt.Errorf("error creating Job %v", err)
		}
	}
	return nil
}

func (f *importFeature) weImportTheGitHubRepoSelectingPipeline(originalRepoName, pipeline string) error {
	// lets fork the repository first
	fmt.Printf("forking upstream %s\n", originalRepoName)
	forker := &github.ForkFeature{
		GitCommander: github.CreateGitCommander(),
	}
	repository, err := forker.ForkToUsersRepo(originalRepoName)
	if err != nil {
		return err
	}
	fmt.Printf("fork is %s\n", repository)

	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client %v", err)
	}
	params := url.Values{}
	params.Add("repository", repository)
	params.Add("pipeline", pipeline)
	err = jenkins.Build(f.job, params)
	if err != nil {
		return fmt.Errorf("error triggering Job %s %v", f.job.Name, err)
	}
	return nil
}

func (f *importFeature) weMergeThePRWhichIsCreated() error {
	//return godog.ErrPending
	return nil
}

func (f *importFeature) thereShouldBeAJobThatCompletesSuccessfully(jobExpression string) error {
	job := utils.ReplaceEnvVars(jobExpression)
	fmt.Printf("Aseserting there is a job called %s\n", job)
	return godog.ErrPending
}

func FeatureImportContext(s *godog.Suite) {
	f := &importFeature{}

	s.Step(`^there is a fabric(\d+)-import job$`, f.thereIsAFabricImportJob)
	s.Step(`^we import the "([^"]*)" GitHub repo selecting "([^"]*)" pipeline$`, f.weImportTheGitHubRepoSelectingPipeline)
	s.Step(`^we merge the PR which is created$`, f.weMergeThePRWhichIsCreated)
	s.Step(`^there should be a "([^"]*)" job that completes successfully$`, f.thereShouldBeAJobThatCompletesSuccessfully)
}

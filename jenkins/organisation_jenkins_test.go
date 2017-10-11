package jenkins

import (
	"fmt"
	"github.com/DATA-DOG/godog"
	"github.com/fabric8-jenkins/golang-jenkins/utils"
)

func thereAreNoJobsCalled(jobName string) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client", err)
	}

	job, err := jenkins.GetJob(jobName)
	if err != nil {
		return nil
	}
	return fmt.Errorf("error found existing job %s", job.Name)
}

func iImportTheGitHubOrganisation(jobName string) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client", err)
	}

	jobXML, err := utils.GetFileAsString("resources/org_job.xml")
	if err != nil {
		return err
	}

	err = jenkins.CreateJobWithXML(jobXML, jobName)
	if err != nil {
		return fmt.Errorf("error creating organisation Job", err)
	}
	return nil
}

func thereShouldBeAJobAndMoreThanMultibranchJob(jobName string, numberOfMultiBranchProjects int) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client", err)
	}
	job, err := jenkins.GetJob(jobName)
	if err != nil {
		fmt.Errorf("error found existing job %s ", job.Name)
	}
	return nil
}

func ImportOrganisationFeatureContext(s *godog.Suite) {
	s.Step(`^there are no jobs called "([^"]*)"$`, thereAreNoJobsCalled)
	s.Step(`^I import the "([^"]*)" GitHub organisation$`, iImportTheGitHubOrganisation)
	s.Step(`^there should be a "([^"]*)" job and more than (\d+) multibranch job$`, thereShouldBeAJobAndMoreThanMultibranchJob)
}

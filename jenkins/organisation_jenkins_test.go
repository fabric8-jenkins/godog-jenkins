package jenkins

import (
	"os"
	"testing"
	"time"

	"fmt"
	"github.com/DATA-DOG/godog"
	"github.com/fabric8-jenkins/golang-jenkins/utils"
)

func TestMain(m *testing.M) {
	status := godog.RunWithOptions("godogs", func(s *godog.Suite) {
		FeatureContext(s)
	}, godog.Options{
		Format:    "progress",
		Paths:     []string{"features"},
		Randomize: time.Now().UTC().UnixNano(), // randomize scenario execution order
	})

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}

func thereAreNoJobsCalled(jobName string) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client", err)
	}

	job, err := jenkins.GetJob(jobName)
	if err != nil {
		return nil
	}
	return fmt.Errorf("error for existing job ", job.Name)
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

func thereShouldBeAJobAndMoreThanMultibranchJob(arg1 string, arg2 int) error {
	return godog.ErrPending
}

func FeatureContext(s *godog.Suite) {
	s.Step(`^there are no jobs called "([^"]*)"$`, thereAreNoJobsCalled)
	s.Step(`^I import the "([^"]*)" GitHub organisation$`, iImportTheGitHubOrganisation)
	s.Step(`^there should be a "([^"]*)" job and more than (\d+) multibranch job$`, thereShouldBeAJobAndMoreThanMultibranchJob)
}

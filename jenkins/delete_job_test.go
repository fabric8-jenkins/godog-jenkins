package jenkins

import (
	"fmt"
	"github.com/DATA-DOG/godog"
	"github.com/fabric8-jenkins/godog-jenkins/utils"
)

func thereIsAJobCalled(jobName string) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client %v", err)
	}
	job, err := jenkins.GetJob(jobName)
	if err != nil {
		return fmt.Errorf("error finding existing job %s ", job.Name)
	}
	return nil
}

func iDeleteTheJob(jobName string) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client  %v", err)
	}
	job, err := jenkins.GetJob(jobName)
	if err != nil {
		return fmt.Errorf("error finding existing job %s ", job.Name)
	}
	err = jenkins.DeleteJob(job)
	if err != nil {
		return fmt.Errorf("error deleteing job %s ", job.Name)
	}
	return nil
}

func thereShouldNotBeAJob(jobName string) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client  %v", err)
	}

	job, err := jenkins.GetJob(jobName)
	if err != nil {
		return nil
	}
	return fmt.Errorf("error found existing job %s", job.Name)
}

func DeleteJobFeatureContext(s *godog.Suite) {
	s.Step(`^there is a job called "([^"]*)"$`, thereIsAJobCalled)
	s.Step(`^I delete the "([^"]*)" job$`, iDeleteTheJob)
	s.Step(`^there should not be a "([^"]*)" job$`, thereShouldNotBeAJob)
}

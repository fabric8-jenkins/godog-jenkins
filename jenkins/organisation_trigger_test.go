package jenkins

import (
	"fmt"
	"github.com/DATA-DOG/godog"
	"github.com/fabric8-jenkins/godog-jenkins/utils"
)

func thenWaitToCheckTheOrganisationScanForIsSuccessful(jobName string) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client %v", err)
	}

	job, err := jenkins.GetJob(jobName)
	if err != nil {
		return fmt.Errorf("error found existing job %s ", job.Name)
	}

	result, err := jenkins.GetOrganizationScanResult(200, job)

	if err != nil {
		return fmt.Errorf("error getting org scan result %v", err)
	}

	if result == "SUCCESS" {
		return nil
	}
	return fmt.Errorf("error the %s org scan result was %s", jobName, result)
}

func FeatureTriggerContext(s *godog.Suite) {
	s.Step(`^there is a "([^"]*)" job$`, thereIsAJobCalled)
	s.Step(`^I trigger the "([^"]*)" job$`, triggerJob)
	s.Step(`^then wait to check the organisation scan for "([^"]*)" is successful$`, thenWaitToCheckTheOrganisationScanForIsSuccessful)
}

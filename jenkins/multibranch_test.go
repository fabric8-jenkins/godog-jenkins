package jenkins

import (
	"fmt"

	"github.com/DATA-DOG/godog"
	"github.com/jenkins-x/godog-jenkins/utils"
)

func thereIsAJenkinsCredential(arg1 string) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client %v", err)
	}

	err = jenkins.CreateCredential("bdd-test10", "rawlingsj10", "test")
	if err != nil {
		return fmt.Errorf("error creating jenkins credential %s %v", "bdd-test", err)
	}
	return nil
}

func weCreateAMultibranchJobCalled(jobName string) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client %v", err)
	}
	jobXML, err := utils.GetFileAsString("resources/multi_job.xml")
	if err != nil {
		return err
	}
	err = jenkins.CreateJobWithXML(jobXML, jobName)
	if err != nil {
		return fmt.Errorf("error creating Job %v", err)
	}
	return nil
}

func triggerAScanOfTheJob(jobName string) error {

	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client %v", err)
	}

	job, err := jenkins.GetJob(jobName)
	if err != nil {
		return fmt.Errorf("error creating Job %v", err)
	}

	err = jenkins.Build(job, nil)
	if err != nil {
		return fmt.Errorf("error triggering job %s %v", jobName, err)
	}
	return nil

}

func thereShouldBeAJobThatCompletesSuccessfully(jobName string) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client %v", err)
	}
	return ThereShouldBeAJobThatCompletesSuccessfully(jobName, jenkins)
}

func theApplicationIsInTheEnvironment(arg1, arg2, arg3 string) error {
	return godog.ErrPending
}

func MultibranchFeatureContext(s *godog.Suite) {
	s.Step(`^there is a "([^"]*)" jenkins credential$`, thereIsAJenkinsCredential)
	s.Step(`^we create a multibranch job called "([^"]*)"$`, weCreateAMultibranchJobCalled)
	s.Step(`^trigger a scan of the job "([^"]*)"$`, triggerAScanOfTheJob)
	s.Step(`^there should be a "([^"]*)" job that completes successfully$`, thereShouldBeAJobThatCompletesSuccessfully)
	s.Step(`^the "([^"]*)" application is "([^"]*)" in the "([^"]*)" environment$`, theApplicationIsInTheEnvironment)
}

package jenkins

import (
	"fmt"
	"github.com/DATA-DOG/godog"
	"github.com/fabric8-jenkins/godog-jenkins/utils"
)

func thereIsAJob(jobName string) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client  %v", err)
	}
	_, err = jenkins.GetJob(jobName)
	if err != nil {
		return fmt.Errorf("error finding existing job %s ", jobName)
	}
	return nil
}

func iTriggerTheJob(jobName string) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client  %v", err)
	}
	job, err := jenkins.GetJob(jobName)
	if err != nil {
		return fmt.Errorf("error finding existing job %s ", jobName)
	}
	err = jenkins.TriggerJob(job)
	if err != nil {
		return fmt.Errorf("error triggering job %s ", jobName)
	}
	return nil
}

func theJobIsSuccessful(arg1 string) error {
	return godog.ErrPending
}

func FeatureContext(s *godog.Suite) {
	s.Step(`^there is a job "([^"]*)"$`, thereIsAJob)
	s.Step(`^I trigger the job "([^"]*)"$`, iTriggerTheJob)
	s.Step(`^the job "([^"]*)" is successful$`, theJobIsSuccessful)
}

//
//func thereAreIsAJobCalled(jobName string) error {
//	jenkins, err := utils.GetJenkinsClient()
//	if err != nil {
//		return fmt.Errorf("error getting a Jenkins client %v", err)
//	}
//	job, err := jenkins.GetJob(jobName)
//	if err != nil {
//		fmt.Errorf("error found existing job %s ", job.Name)
//	}
//	return nil
//}
//
//func iDeleteTheJob(jobName string) error {
//	jenkins, err := utils.GetJenkinsClient()
//	if err != nil {
//		return fmt.Errorf("error getting a Jenkins client  %v", err)
//	}
//	job, err := jenkins.GetJob(jobName)
//	if err != nil {
//		fmt.Errorf("error finding existing job %s ", job.Name)
//	}
//	err = jenkins.DeleteJob(job)
//	if err != nil {
//		fmt.Errorf("error deleteing job %s ", job.Name)
//	}
//	return nil
//}
//
//func thereShouldNotBeAJob(jobName string) error {
//	jenkins, err := utils.GetJenkinsClient()
//	if err != nil {
//		return fmt.Errorf("error getting a Jenkins client  %v", err)
//	}
//
//	job, err := jenkins.GetJob(jobName)
//	if err != nil {
//		return nil
//	}
//	return fmt.Errorf("error found existing job %s", job.Name)
//}
//
//func DeleteJobFeatureContext(s *godog.Suite) {
//	s.Step(`^there are is a job called "([^"]*)"$`, thereAreIsAJobCalled)
//	s.Step(`^I delete the "([^"]*)" job$`, iDeleteTheJob)
//	s.Step(`^there should not be a "([^"]*)" job$`, thereShouldNotBeAJob)
//}

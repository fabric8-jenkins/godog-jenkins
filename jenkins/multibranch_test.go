package jenkins

import (
	"fmt"
	"github.com/DATA-DOG/godog"
	"github.com/fabric8-jenkins/godog-jenkins/utils"
	"github.com/fabric8-jenkins/golang-jenkins"
	"time"
)

type mutibranchFeature struct {
	parent string
	name   string
	branch string
	job    gojenkins.Job
	client gojenkins.Jenkins
}

func (m *mutibranchFeature) organisationJobContainsAJob(orgJobName, multibranchJobName string) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client %v", err)
	}
	job, err := jenkins.GetMultiBranchJob(orgJobName, multibranchJobName, "master")

	if err != nil {
		return fmt.Errorf("error finding multibranch job %s in organisation job %s ", multibranchJobName, orgJobName)
	}
	m.job = job
	m.parent = orgJobName
	m.name = multibranchJobName
	return nil
}

func (m *mutibranchFeature) iTriggerTheMultibranchJob(multibranchJobName string) error {
	if m.name != multibranchJobName {
		return fmt.Errorf("error matching multi branch Job %s with previously configured job %s", multibranchJobName, m.name)
	}
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client %v", err)
	}
	utils.LogInfof("Triggering Job: %s\n", m.job.Url)
	err = jenkins.Build(m.job, nil)
	if err != nil {
		return fmt.Errorf("error triggering job %s %v", m.job.FullName, err)
	}
	return nil
}

func (m *mutibranchFeature) theJobIsSuccessful(arg1 string) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client %v", err)
	}

	// wait for build to finish
	err = utils.RetryAfter(200, func() error {

		// wait for build to start
		build, err := jenkins.GetLastBuild(m.job)
		if err != nil {
			return fmt.Errorf("error getting last build for job %s %v", m.job.FullName, err)
		}

		if build.Result == "" {
			return fmt.Errorf("build still running")
		}
		return nil
	}, time.Second*5)
	// check result
	build, err := jenkins.GetLastBuild(m.job)
	if err != nil {
		return fmt.Errorf("error getting last build for job %s %v", m.job.FullName, err)
	}

	if build.Result != "SUCCESS" {
		return fmt.Errorf("build result %s", build.Result)
	}

	return nil
}

func FeatureMultiBranchContext(s *godog.Suite) {
	m := &mutibranchFeature{}

	s.Step(`^organisation job "([^"]*)" contains a "([^"]*)" job$`, m.organisationJobContainsAJob)
	s.Step(`^I trigger the multibranch job "([^"]*)"$`, m.iTriggerTheMultibranchJob)
	s.Step(`^the job "([^"]*)" is successful$`, m.theJobIsSuccessful)
}

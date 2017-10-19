package jenkins

import (
	"fmt"
	"strings"
	"time"

	"github.com/fabric8-jenkins/golang-jenkins"
	"github.com/fabric8-jenkins/godog-jenkins/utils"
)


// Is404 returns true if this is a 404 error
func Is404(err error) bool {
	text := fmt.Sprintf("%s", err)
	return strings.HasPrefix(text, "404 ")
}

// TriggerAndWaitForBuildToStart triggers the build and waits for a new Build for the given amount of time
// or returns an error
func TriggerAndWaitForBuildToStart(jenkins *gojenkins.Jenkins, job gojenkins.Job, buildStartWaitTime time.Duration) (*gojenkins.Build, error) {
	previousBuildNumber := 0
	previousBuild, err := jenkins.GetLastBuild(job)
	jobUrl := job.Url
	if err != nil {
		if !Is404(err) {
			//return nil, fmt.Errorf("error finding last build for %s due to %v", job.Name, err)
			utils.LogInfof("Warning: error finding previous build for %s due to %v\n", jobUrl, err)
		}
	} else {
		previousBuildNumber = previousBuild.Number
	}
	err = jenkins.Build(job, nil)
	if err != nil {
		if !Is404(err) {
			return nil, fmt.Errorf("error triggering build %s due to %v", jobUrl, err)
		}
	}
	attempts := 0

	// lets wait for a new build to start
	timeoutAt := time.Now().Add(buildStartWaitTime)
	for {
		buildNumber := 0
		attempts += 1
		build, err := jenkins.GetLastBuild(job)
		if err != nil {
			if !Is404(err) {
				//return nil, fmt.Errorf("error finding last build for %s due to %v", job.Name, err)
				utils.LogInfof("Warning: error finding last build attempt %d for %s due to %v\n", attempts, jobUrl, err)
			}
		} else {
			buildNumber = build.Number
		}
		if previousBuildNumber != buildNumber {
			utils.LogInfof("triggered job %s build #%d\n", jobUrl, buildNumber)
			return &build, nil
		}
		if time.Now().After(timeoutAt) {
			return nil, fmt.Errorf("Timed out waiting for build to start for for %s waited for %s", jobUrl, buildStartWaitTime.String())
		}
		time.Sleep(1 * time.Second)
	}
}

// TriggerAndWaitForBuildToStart triggers the build and waits for a new Build then waits for the Build to finish
// or returns an error
func TriggerAndWaitForBuildToFinish(jenkins *gojenkins.Jenkins, job gojenkins.Job, buildStartWaitTime time.Duration, buildFinishWaitTime time.Duration) (*gojenkins.Build, error) {
	build, err := TriggerAndWaitForBuildToStart(jenkins, job, buildStartWaitTime)
	if err != nil {
		return build, err
	}
	if (!build.Building) {
		return build, nil
	}
	return WaitForBuildToFinish(jenkins, job, build.Number, buildFinishWaitTime)
}

// TriggerAndWaitForBuildToStart triggers the build and waits for a new Build then waits for the Build to finish
// or returns an error
func WaitForBuildToFinish(jenkins *gojenkins.Jenkins, job gojenkins.Job, buildNumber int, buildFinishWaitTime time.Duration) (*gojenkins.Build, error) {
	jobUrl := job.Url
	timeoutAt := time.Now().Add(buildFinishWaitTime)
	utils.LogInfof("waiting for job %s build #%d to finish\n", jobUrl, buildNumber)

	for {
		time.Sleep(1 * time.Second)

		b, err := jenkins.GetBuild(job, buildNumber)
		if err != nil {
			return nil, fmt.Errorf("error finding job %s build #%d status due to %v", jobUrl, buildNumber, err)
		}
		if !b.Building {
			return &b, nil
		}
		if time.Now().After(timeoutAt) {
			return nil, fmt.Errorf("Timed out waiting for job %s build #%d to finish. Waited for %s", jobUrl, buildNumber, buildFinishWaitTime.String())
		}
	}
}

// AssertBuildSucceeded asserts that the given build succeeded
func AssertBuildSucceeded(build *gojenkins.Build, jobName string) error {
	result := build.Result
	utils.LogInfof("Job %s build %d has result %s\n", jobName, build.Number, result)
	if result == "SUCCESS" {
		return nil
	}
	return fmt.Errorf("Job %s build %d has result %s", jobName, build.Number, result)

}
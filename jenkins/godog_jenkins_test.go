
package jenkins

import (
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/fabric8-jenkins/godog-jenkins/utils"
	"fmt"
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
func thereAreJobs(expectedNoOfJobs int) error {
	c, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client", err)
	}

	numOfJobs, err := c.GetJobs()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client", err)
	}
	if len(numOfJobs) != expectedNoOfJobs {
		return fmt.Errorf("expected %d jobs, but found %d", expectedNoOfJobs, len(numOfJobs))
	}

	return nil
}

func iImportTheFabricQuicktarttestOrg(arg1 int) error {
	return godog.ErrPending
}

func thereShouldBeAGitHubOrganisationJobAndMultibranchJobs(arg1 int) error {
	return godog.ErrPending
}

func FeatureContext(s *godog.Suite) {
	s.Step(`^there are (\d+) Jobs$`, thereAreJobs)
	s.Step(`^I import the fabric(\d+)-quicktart-test org$`, iImportTheFabricQuicktarttestOrg)
	s.Step(`^there should be a GitHub organisation job and > (\d+) multibranch jobs$`, thereShouldBeAGitHubOrganisationJobAndMultibranchJobs)
}
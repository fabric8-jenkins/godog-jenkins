package jenkins

import (
	"github.com/DATA-DOG/godog"
)

func theOrganisationScanForCompletes(arg1 string) error {
	return godog.ErrPending
}

func FeatureTriggerContext(s *godog.Suite) {
	s.Step(`^there is a "([^"]*)" job$`, thereIsAJob)
	s.Step(`^I trigger the "([^"]*)" job$`, iTriggerTheJob)
	s.Step(`^the organisation scan for "([^"]*)" completes$`, theOrganisationScanForCompletes)
}

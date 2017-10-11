package github

import (
	"github.com/DATA-DOG/godog"
)

func thereIsNoForkOf(repo string) error {
	return nil
}

func iForkTheGitHubOrganisation(originalRepo string) error {
	return godog.ErrPending
}

func thereShouldBeAForkWhichHasTheSameLastCommitAs(forkedRepo, originalRepo string) error {
	return godog.ErrPending
}

func FeatureContext(s *godog.Suite) {
	s.Step(`^there is no fork of "([^"]*)"$`, thereIsNoForkOf)
	s.Step(`^I fork the "([^"]*)" GitHub organisation$`, iForkTheGitHubOrganisation)
	s.Step(`^there should be a "([^"]*)" fork which has the same last commit as "([^"]*)"$`, thereShouldBeAForkWhichHasTheSameLastCommitAs)
}


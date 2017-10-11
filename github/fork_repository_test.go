package github

import (
	"github.com/DATA-DOG/godog"
	"fmt"
)

func thereIsNoForkOf(repo string) error {
	return nil
}

func iForkTheGitHubOrganisationToUser(originalRepo string, newUser string) error {
	userRepo, err := ParseUserRepositoryName(originalRepo)
	if err != nil {
		return err;
	}
	client, err := CreateGitHubClient()
	if err != nil {
		return err;
	}

	// now lets fork it
	repo, err := ForkRepositoryOrRevertMasterInFork(client, userRepo, newUser)
	if err != nil {
		return err
	}
	gitcmder := CreateGitCommander()
	dir, err := gitcmder.Clone(repo)
	if err == nil {
		fmt.Printf("Cloned to directory: %s\n", dir)
	}
	return err
}

func thereShouldBeAForkWhichHasTheSameLastCommitAs(forkedRepo, originalRepo string) error {
	return godog.ErrPending
}

func FeatureContext(s *godog.Suite) {
	s.Step(`^there is no fork of "([^"]*)"$`, thereIsNoForkOf)
	s.Step(`^I fork the "([^"]*)" GitHub organisation to user "([^"]*)"$`, iForkTheGitHubOrganisationToUser)
	s.Step(`^there should be a "([^"]*)" fork which has the same last commit as "([^"]*)"$`, thereShouldBeAForkWhichHasTheSameLastCommitAs)
}


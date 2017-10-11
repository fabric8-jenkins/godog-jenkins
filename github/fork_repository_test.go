package github

import (
	"fmt"

	"github.com/DATA-DOG/godog"
)

type forkFeature struct {
	GitCommander *GitCommander

	UpstreamDir string
	ForkDir     string
}

func (f *forkFeature) thereIsNoForkOf(repo string) error {
	return nil
}

func (f *forkFeature) iForkTheGitHubOrganisationToUser(originalRepoName string, newUser string) error {
	userRepo, err := ParseUserRepositoryName(originalRepoName)
	if err != nil {
		return err
	}
	client, err := CreateGitHubClient()
	if err != nil {
		return err
	}
	gitcmder := f.GitCommander
	err = gitcmder.DeleteWorkDir()
	if err != nil {
		return err
	}

	upstreamRepo, err := GetRepository(client, userRepo.Organisation, userRepo.Repository)
	if err != nil {
		return err
	}

	// now lets fork it
	repo, err := ForkRepositoryOrRevertMasterInFork(client, userRepo, newUser)
	if err != nil {
		return err
	}
	dir, err := gitcmder.Clone(repo)
	if err == nil {
		fmt.Printf("Cloned to directory: %s\n", dir)
	}
	f.ForkDir = dir

	upstreamCloneURL, err := GetCloneURL(upstreamRepo, true)
	if err != nil {
		return err
	}

	upstreamDir, err := gitcmder.CloneFromURL(upstreamRepo, upstreamCloneURL)
	if err != nil {
		return err
	}
	f.UpstreamDir = upstreamDir

	err = gitcmder.ResetMasterFromUpstream(dir, upstreamCloneURL)
	if err != nil {
		return err
	}

	return nil
}

func (f *forkFeature) thereShouldBeAForkWhichHasTheSameLastCommitAs(forkedRepo, originalRepo string) error {
	gitcmder := f.GitCommander
	upstreamSha, err := gitcmder.GetLastCommitSha(f.UpstreamDir)
	if err != nil {
		return err
	}
	forkSha, err := gitcmder.GetLastCommitSha(f.ForkDir)
	if err != nil {
		return err
	}
	fmt.Printf("upstream last commit is %s\n", upstreamSha)
	fmt.Printf("fork last commit is %s\n", forkSha)

	errors := CreateErrorSlice()
	assert := CreateAssert(errors)

	msg := fmt.Sprintf("The git sha on the fork should be the same as the upstream repository in dir %s and %s", f.ForkDir, f.UpstreamDir)
	assert.Equal(upstreamSha, forkSha, msg)
	return errors.Error()
}

func FeatureContext(s *godog.Suite) {
	f := &forkFeature{
		GitCommander: CreateGitCommander(),
	}

	s.Step(`^there is no fork of "([^"]*)"$`, f.thereIsNoForkOf)
	s.Step(`^I fork the "([^"]*)" GitHub organisation to user "([^"]*)"$`, f.iForkTheGitHubOrganisationToUser)
	s.Step(`^there should be a "([^"]*)" fork which has the same last commit as "([^"]*)"$`, f.thereShouldBeAForkWhichHasTheSameLastCommitAs)
}

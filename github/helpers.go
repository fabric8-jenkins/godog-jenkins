package github

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/github"
)

type UserRepositoryName struct {
	Organisation string
	Repository string
}

func (r *UserRepositoryName) String() string {
	return r.Organisation + "/" + r.Repository
}

// ParseUserRepositoryName parses a repo name of the form `orgName/repoName` or returns a failure if
// the text cannot be parsed
func ParseUserRepositoryName(text string) (*UserRepositoryName, error) {
	values := strings.Split(text, "/")
	if len(values) != 2 {
		return nil, fmt.Errorf("Invalid github repository name. Expected the format `orgName/RepoName` but got %s", text)
	}
	return &UserRepositoryName{
		Organisation: values[0],
		Repository: values[1],
	}, nil
}

// CreateGitHubClient creates a new GitHub client
func CreateGitHubClient() (*github.Client, error) {
	/*
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "... your access token ..."},
	)
	tc := oauth2.NewClient(ctx, ts)
	*/

	user, err := mandatoryEnvVar("GITHUB_USER")
	if err != nil {
		return nil, err
	}
	pwd, err := mandatoryEnvVar("GITHUB_PASSWORD")
	if err != nil {
		return nil, err
	}
	basicAuth := github.BasicAuthTransport{
		Username: user,
		Password: pwd,

	}
	httpClient := basicAuth.Client()
	return github.NewClient(httpClient), nil
}

// ForkRepositoryOrRevertMasterInFork forks the given repository to the new owner or resets the fork
// to the upstream master
func ForkRepositoryOrRevertMasterInFork(client *github.Client, userRepo *UserRepositoryName, newOwner string) (*github.Repository, error) {
	repoOwner := userRepo.Organisation
	repoName := userRepo.Repository
	repo, _, err := client.Repositories.Get(repoOwner, repoName)
	if err != nil {
		return nil, err;
	}
	u := repo.HTMLURL
	if u != nil {
		fmt.Printf("Found repository at %s\n", *u)
	}

	forkRepo, _, err := client.Repositories.Get(newOwner, repoName)
	if err != nil {
		return nil, err;
	}

	if forkRepo == nil {
		fmt.Println("No fork available yet")

		opts := &github.RepositoryCreateForkOptions{
			Organization: newOwner,
		}
		forkRepo, _, err = client.Repositories.CreateFork(repoOwner, repoName, opts)
		if err != nil {
			return nil, fmt.Errorf("Failed to fork repo %s to user %s due to %s", userRepo.String(), newOwner, err)
		}
	}
	return forkRepo, nil
}


func mandatoryEnvVar(name string) (string, error) {
	answer := os.Getenv(name)
	if len(answer) == 0 {
		return "", fmt.Errorf("Missing environment variable value $%s", name);
	}
	return answer, nil
}

package jenkins

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/fabric8-jenkins/godog-jenkins/utils"
	"github.com/fabric8-jenkins/golang-jenkins"
	"github.com/fabric8-jenkins/godog-jenkins/github"

	gh "github.com/google/go-github/github"
)

type importFeature struct {
	job              gojenkins.Job
	GitHubClient     *gh.Client
	Jenkins          *gojenkins.Jenkins
	ForkedRepository string
	ImportJobName    string
	LastBuildNumber  int
}

func (f *importFeature) thereIsAFabricImportJob(arg int) error {
	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client %v", err)
	}

	jobName := f.ImportJobName
	f.job, err = jenkins.GetJob(jobName)
	if err != nil {
		jobXML, err := utils.GetFileAsString("resources/import_job.xml")
		if err != nil {
			return err
		}

		err = jenkins.CreateJobWithXML(jobXML, jobName)
		if err != nil {
			return fmt.Errorf("error creating Job %v", err)
		}
		f.job, err = jenkins.GetJob(jobName)
		if err != nil {
			return fmt.Errorf("error creating Job %v", err)
		}
	}
	return nil
}

func (f *importFeature) weImportTheGitHubRepoSelectingPipeline(originalRepoName, pipeline string) error {
	// lets fork the repository first
	fmt.Printf("forking upstream %s\n", originalRepoName)
	forker := &github.ForkFeature{
		GitCommander: github.CreateGitCommander(),
	}
	repository, err := forker.ForkToUsersRepo(originalRepoName)
	if err != nil {
		return err
	}
	f.ForkedRepository = repository
	fmt.Printf("fork is %s\n", repository)

	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client %v", err)
	}
	f.Jenkins = jenkins

	build, err := jenkins.GetLastBuild(f.job)
	if err != nil {
		return fmt.Errorf("Failed to find last build for job %s due to %v\n", f.job.Name, err)
	}
	f.LastBuildNumber = build.Number

	params := url.Values{}
	params.Add("repository", repository)
	params.Add("pipeline", pipeline)
	err = jenkins.Build(f.job, params)
	if err != nil {
		return fmt.Errorf("error triggering Job %s %v", f.job.Name, err)
	}
	return nil
}

func (f *importFeature) weMergeThePRWhichIsCreated() error {
	repoName, err := github.ParseUserRepositoryName(f.ForkedRepository)
	if err != nil {
		return err
	}
	owner := repoName.Organisation
	name := repoName.Repository

	jenkins := f.Jenkins

	// lets keep polling for pending PRs
	ghc, err := github.CreateGitHubClient()
	if err != nil {
		return err
	}
	f.GitHubClient = ghc


	ctx := context.Background()
	prOpts := &gh.PullRequestListOptions{
		State: "open",
	}
	loggedNotStarted := false

	//  wait for the import to complete merging open PRs if we find any
	for {
		importJob := f.ImportJobName
		job, err := jenkins.GetJob(importJob)
		if err != nil {
			fmt.Printf("WARNING: could not find import job %s due to %v\n", importJob, err)
		}
		build, err := jenkins.GetLastBuild(job)
		if err != nil {
			fmt.Printf("WARNING: could not find last build of job %s due to %v\n", importJob, err)
		} else {
			if build.Number == f.LastBuildNumber {
				if !loggedNotStarted {
					loggedNotStarted = true
					fmt.Printf("import job not started yet. Last build is still #%d\n", build.Number)
				}
				continue
			}
			if !build.Building {
				result := build.Result
				fmt.Printf("Job %s build %d has result %s\n", importJob, build.Number, result)
				if result == "SUCCESS" {
					return nil
				}
				return fmt.Errorf("Job %s build %d has result %s", importJob, build.Number, result)
			}
		}

		prs, _, err := ghc.PullRequests.List(ctx, owner, name, prOpts)
		if err != nil {
			return fmt.Errorf("Failed to poll PullRequests on repository %s/%s due to %v", owner, name, err)
		}
		for _, pr := range prs {
			url := ""
			title := ""
			if pr.HTMLURL != nil {
				url = *pr.HTMLURL
			}
			if pr.Title != nil {
				title = *pr.Title
			}

			if pr.Number != nil {
				n := *pr.Number
				mergeOpts := &gh.PullRequestOptions{
				 	MergeMethod: "rebase",
				}
				//fmt.Printf("Merging PR %s %s\n", url, title)
				r, _, err := ghc.PullRequests.Merge(ctx, owner, name, n, "godog merging", mergeOpts)
				if err != nil {
					return fmt.Errorf("Failed to merge PR %s due to %v", url, err)
				}
				if r.Merged != nil && *r.Merged {
					fmt.Printf("merged PR %s %s\n", url, title)
				}  else {
					return fmt.Errorf("Failed to merge PR %s got result %v", url, r)
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}


func (f *importFeature) thereShouldBeAJobThatCompletesSuccessfully(jobExpression string) error {
	job := utils.ReplaceEnvVars(jobExpression)
	jenkins := f.Jenkins

	paths := strings.Split(job, "/")
	fullPath := gojenkins.FullJobPath(paths...)

	_, err := jenkins.GetJobByPath(paths...)
	if err != nil {
		return fmt.Errorf("Failed to find job %s due to %v", fullPath, err)
	}
	return nil
}

func FeatureImportContext(s *godog.Suite) {
	f := &importFeature{
		ImportJobName: "fabric8-import",
	}

	s.Step(`^there is a fabric(\d+)-import job$`, f.thereIsAFabricImportJob)
	s.Step(`^we import the "([^"]*)" GitHub repo selecting "([^"]*)" pipeline$`, f.weImportTheGitHubRepoSelectingPipeline)
	s.Step(`^we merge the PR which is created$`, f.weMergeThePRWhichIsCreated)
	s.Step(`^there should be a "([^"]*)" job that completes successfully$`, f.thereShouldBeAJobThatCompletesSuccessfully)
}

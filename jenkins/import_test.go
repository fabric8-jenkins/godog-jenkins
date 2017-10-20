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

const (
	maxWaitForImportBuildToComplete = 2 * time.Minute

	maxWaitForBuildToStart     = 20 * time.Second
	maxWaitForBuildToBeCreated = 50 * time.Second
	maxWaitForBuildToComplete  = 40 * time.Minute
)

type importFeature struct {
	job                  gojenkins.Job
	GitHubClient         *gh.Client
	Jenkins              *gojenkins.Jenkins
	ForkedRepository     string
	ImportJobName        string
	LastBuildNumber      int
	TriggeredBuildNumber int
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
	utils.LogInfof("forking upstream %s\n", originalRepoName)
	forker := &github.ForkFeature{
		GitCommander: github.CreateGitCommander(),
	}
	repository, err := forker.ForkToUsersRepo(originalRepoName)
	if err != nil {
		return err
	}
	f.ForkedRepository = repository
	utils.LogInfof("fork is %s\n", repository)

	jenkins, err := utils.GetJenkinsClient()
	if err != nil {
		return fmt.Errorf("error getting a Jenkins client %v", err)
	}
	f.Jenkins = jenkins

	build, err := jenkins.GetLastBuild(f.job)
	if err != nil {
		if Is404(err) {
			f.LastBuildNumber = 0
		} else {
			return fmt.Errorf("Failed to find last build for job %s due to %v\n", f.job.Name, err)
		}
	} else {
		f.LastBuildNumber = build.Number
	}

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

	newBuildNumber := -1

	//  wait for the import to complete merging open PRs if we find any
	for {
		importJob := f.ImportJobName
		job, err := jenkins.GetJob(importJob)
		if err != nil {
			utils.LogInfof("WARNING: could not find import job %s due to %v\n", importJob, err)
		}
		var build gojenkins.Build
		if newBuildNumber < 0 {
			build, err = jenkins.GetLastBuild(job)
			if Is404(err) {
				err = nil
				build = gojenkins.Build{
					Number: 0,
				}
			}
		} else {
			build, err = jenkins.GetBuild(job, newBuildNumber)
		}
		if err != nil {
			utils.LogInfof("WARNING: could not find last build of job %s due to %v\n", importJob, err)
		} else {
			if build.Number == f.LastBuildNumber {
				if !loggedNotStarted {
					loggedNotStarted = true
					utils.LogInfof("import job not started yet. Last build is still #%d\n", build.Number)
				}
				continue
			}
			if newBuildNumber < 0 {
				newBuildNumber = build.Number
				utils.LogInfof("import job started build #%d\n", newBuildNumber)
			}
			if !build.Building {
				return AssertBuildSucceeded(&build, importJob)
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
				//utils.LogInfof("Merging PR %s %s\n", url, title)
				r, _, err := ghc.PullRequests.Merge(ctx, owner, name, n, "godog merging", mergeOpts)
				if err != nil {
					return fmt.Errorf("Failed to merge PR %s due to %v", url, err)
				}
				if r.Merged != nil && *r.Merged {
					utils.LogInfof("merged PR %s %s\n", url, title)
				} else {
					return fmt.Errorf("Failed to merge PR %s got result %v", url, r)
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func (f *importFeature) weTriggerTheJob(jobExpression string) error {
	job, err := f.getJobByExpression(jobExpression)
	if err != nil {
		return err
	}
	jenkins := f.Jenkins
	build, err := TriggerAndWaitForBuildToFinish(jenkins, job, maxWaitForBuildToStart, maxWaitForBuildToComplete)
	if err != nil {
		return err
	}
	f.TriggeredBuildNumber = build.Number
	return nil
}

func (f *importFeature) theScanCompletesSuccessfully(jobExpression string) error {
	// TODO how to wait for the scan to finish?
	//return f.thereShouldBeAJobThatCompletesSuccessfully(jobExpression)
	return nil
}

func (f *importFeature) thereShouldBeAJobThatCompletesSuccessfully(jobExpression string) error {
	job, err := f.waitForJobByExpression(jobExpression, maxWaitForBuildToBeCreated)
	if err != nil {
		return err
	}
	jenkins := f.Jenkins
	build, err := WaitForBuildToFinish(jenkins, job, f.TriggeredBuildNumber, maxWaitForBuildToComplete)
	if err != nil {
		return err
	}
	return AssertBuildSucceeded(build, job.Url)
}

func (f *importFeature) waitForJobByExpression(jobExpression string, timeout time.Duration) (job gojenkins.Job, err error) {
	jobPath := utils.ReplaceEnvVars(jobExpression)
	jenkins := f.Jenkins

	paths := strings.Split(jobPath, "/")
	fullPath := gojenkins.FullJobPath(paths...)

	fn := func() (bool, error) {
		job, err = jenkins.GetJobByPath(paths...)
		if err != nil {
			if !Is404(err) {
				err = fmt.Errorf("Failed to find job %s due to %v", fullPath, err)
				return false, err
			}
		} else {
			return true, nil
		}
		return false, nil
	}

	err = utils.Poll(1 * time.Second, timeout, fn, fmt.Sprintf("build to be created for %s", fullPath))
	return
}

func (f *importFeature) getJobByExpression(jobExpression string) (job gojenkins.Job, err error) {
	jobPath := utils.ReplaceEnvVars(jobExpression)
	jenkins := f.Jenkins

	paths := strings.Split(jobPath, "/")
	fullPath := gojenkins.FullJobPath(paths...)

	job, err = jenkins.GetJobByPath(paths...)
	if err != nil {
		err = fmt.Errorf("Failed to find job %s due to %v", fullPath, err)
	}
	return
}

func FeatureContext(s *godog.Suite) {
}

func FeatureImportContext(s *godog.Suite) {
	f := &importFeature{
		ImportJobName: "fabric8-import",
	}

	s.Step(`^there is a fabric(\d+)-import job$`, f.thereIsAFabricImportJob)
	s.Step(`^we import the "([^"]*)" GitHub repo selecting "([^"]*)" pipeline$`, f.weImportTheGitHubRepoSelectingPipeline)
	s.Step(`^we merge the PR which is created$`, f.weMergeThePRWhichIsCreated)
	s.Step(`^the "([^"]*)" scan completes successfully$`, f.theScanCompletesSuccessfully)
	s.Step(`^we trigger the "([^"]*)" job$`, f.weTriggerTheJob)
	s.Step(`^there should be a "([^"]*)" job that completes successfully$`, f.thereShouldBeAJobThatCompletesSuccessfully)
}

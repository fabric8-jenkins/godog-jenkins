package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"strings"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
)

const (
	DefaultWritePermissions = 0760

	defaultGitIgnoreFile = `
.project
.classpath
.idea
.cache
.DS_Store
*.im?
target
work
`

	// TODO figure out how to pass extra dockerfiles from a draft pack
	defaultDockerfile = `
FROM openjdk:8-jdk-alpine
ENV PORT 8080
EXPOSE 8080
COPY target/*.jar /opt/app.jar
WORKDIR /opt
CMD ["java", "-jar", "app.jar"]
`

	// TODO replace with the jx-pipelines-plugin version when its available
	defaultJenkinsfile = `
pipeline {
  agent {
    label "jenkins-maven"
  }

  environment {
    ORG 		= 'jenkinsx'
    APP_NAME    = '%s'
  }

  stages {

    stage('Build Release') {
      steps {
        container('maven') {
          // ensure we're not on a detached head
          sh "git checkout master"

          // until we switch to the new kubernetes / jenkins credential implementation use git credentials store
          sh "git config credential.helper store"

          // so we can retrieve the version in later steps
          sh "echo \$(jx-release-version) > VERSION"
          sh "mvn versions:set -DnewVersion=\$(jx-release-version)"
        }

        dir ('./charts/%s') {
          container('maven') {
            sh "make tag"
          }
        }

        container('maven') {
          sh 'mvn clean deploy'
          sh "docker build -f Dockerfile.release -t $JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST:$JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT/$ORG/$APP_NAME:\$(cat VERSION) ."
          sh "docker push $JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST:$JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT/$ORG/$APP_NAME:\$(cat VERSION)"
        }
      }
    }
    stage('Deploy Staging') {

      steps {
        dir ('./charts/%s') {
          container('maven') {

            sh 'make release'
            sh 'helm install . --namespace staging --name example-release'
            sh 'exposecontroller --namespace staging --http' // until we switch to git environments where helm hooks will expose services
          }
        }
      }
    }
  }
}
`
)

type ImportOptions struct {
	CommonOptions

	RepoURL string

	Dir                     string
	Organisation            string
	Repository              string
	Credentials             string
	AppName                 string
	GitHub                  bool
	DryRun                  bool
	SelectAll               bool
	DisableDraft            bool
	DisableJenkinsfileCheck bool
	SelectFilter            string
	Jenkinsfile             string

	DisableDotGitSearch bool
	Jenkins             *gojenkins.Jenkins
	GitConfDir          string
	GitProvider         gits.GitProvider
}

var (
	import_long = templates.LongDesc(`
		Imports a git repository or folder into Jenkins X.

		If you specify no other options or arguments then the current directory is imported.
	    Or you can use '--dir' to specify a directory to import.

	    You can specify the git URL as an argument.`)

	import_example = templates.Examples(`
		# Import the current folder
		jx import

		# Import a different folder
		jx import /foo/bar

		# Import a git repository from a URL
		jx import --url https://github.com/jenkins-x/spring-boot-web-example.git

        # Select a number of repositories from a github organisation
		jx import --github --org myname 

        # Import all repositories from a github organisation selecting ones to not import
		jx import --github --org myname --all 

        # Import all repositories from a github organisation which contain the text foo
		jx import --github --org myname --all --filter foo 
		`)
)

func NewCmdImport(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ImportOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "import",
		Short:   "Imports a local project into Jenkins",
		Long:    import_long,
		Example: import_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.RepoURL, "url", "u", "", "The git clone URL to clone into the current directory and then import")
	cmd.Flags().StringVarP(&options.Organisation, "org", "o", "", "Specify the git provider organisation to import the project into (if it is not already in one)")
	cmd.Flags().StringVarP(&options.Repository, "name", "n", "", "Specify the git repository name to import the project into (if it is not already in one)")
	cmd.Flags().StringVarP(&options.Credentials, "credentials", "c", "", "The Jenkins credentials name used by the job")
	cmd.Flags().StringVarP(&options.Jenkinsfile, "jenkinsfile", "j", "", "The name of the Jenkinsfile to use. If not specified then 'Jenkinsfile' will be used")
	cmd.Flags().BoolVarP(&options.DryRun, "dry-run", "d", false, "Performs local changes to the repo but skips the import into Jenkins X")
	cmd.Flags().BoolVarP(&options.GitHub, "github", "", false, "If you wis to pick the repositories from GitHub to import")
	cmd.Flags().BoolVarP(&options.SelectAll, "all", "", false, "If selecting projects to import from a git provider this defaults to selecting them all")
	cmd.Flags().BoolVarP(&options.DisableDraft, "no-draft", "x", false, "Disable Draft from trying to default a Dockerfile and Helm Chart")
	cmd.Flags().BoolVarP(&options.DisableJenkinsfileCheck, "no-jenkinsfile", "", false, "Disable defaulting a Jenkinsfile if its missing")
	cmd.Flags().StringVarP(&options.SelectFilter, "filter", "", "", "If selecting projects to import from a git provider this filters the list of repositories")
	return cmd
}

func (o *ImportOptions) Run() error {
	f := o.Factory
	jenkins, err := f.GetJenkinsClient()
	if err != nil {
		return err
	}
	o.Jenkins = jenkins

	if o.GitHub {
		return o.ImportProjectsFromGitHub()
	}
	if o.Dir == "" {
		args := o.Args
		if len(args) > 0 {
			o.Dir = args[0]
		} else {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			o.Dir = dir
		}
	}
	_, o.AppName = filepath.Split(o.Dir)

	checkForJenkinsfile := o.Jenkinsfile == "" && !o.DisableJenkinsfileCheck
	shouldClone := checkForJenkinsfile || !o.DisableDraft

	if o.RepoURL != "" {
		if shouldClone {
			// lets make sure there's a .git at the end for github URLs
			err = o.CloneRepository()
			if err != nil {
				return err
			}
		}
	} else {
		err = o.DiscoverGit()
		if err != nil {
			return err
		}

		if o.RepoURL == "" {
			err = o.DiscoverRemoteGitURL()
			if err != nil {
				return err
			}
		}
	}

	if !o.DisableDraft {
		err = o.DraftCreate()
		if err != nil {
			return err
		}

		err = o.DefaultDockerfile()
		if err != nil {
			return err
		}
	}

	if checkForJenkinsfile {
		err = o.DefaultJenkinsfile()
		if err != nil {
			return err
		}
	}

	if o.RepoURL == "" {
		err = o.CreateNewRemoteRepository()
		if err != nil {
			return err
		}
	} else {
		if shouldClone {
			err = gits.GitPush(o.Dir)
			if err != nil {
				return err
			}
		}
	}
	if o.DryRun {
		log.Infof("dry-run so skipping import to Jenkins X")
		return nil
	}
	return o.DoImport()
}

func (o *ImportOptions) ImportProjectsFromGitHub() error {
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()
	server := config.GetOrCreateServer(gits.GitHubHost)
	userAuth, err := config.PickServerUserAuth(server, "git user name")
	if err != nil {
		return err
	}
	provider, err := gits.CreateProvider(server, &userAuth)
	if err != nil {
		return err
	}

	username := userAuth.Username
	org := o.Organisation
	if org == "" {
		org, err = gits.PickOrganisation(provider, username)
		if err != nil {
			return err
		}
	}
	repos, err := gits.PickRepositories(provider, org, "Which repositories do you want to import", o.SelectAll, o.SelectFilter)
	if err != nil {
		return err
	}

	o.Printf("Selected repositories\n")
	for _, r := range repos {
		o2 := ImportOptions{
			CommonOptions:           o.CommonOptions,
			Dir:                     o.Dir,
			RepoURL:                 r.CloneURL,
			Organisation:            org,
			Repository:              r.Name,
			Jenkins:                 o.Jenkins,
			GitProvider:             provider,
			DisableJenkinsfileCheck: o.DisableJenkinsfileCheck,
			DisableDraft:            o.DisableDraft,
		}
		o.Printf("Importing repository %s\n", util.ColorInfo(r.Name))
		err = o2.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *ImportOptions) DraftCreate() error {
	args := []string{"create"}

	// TODO this is a workaround of this draft issue:
	// https://github.com/Azure/draft/issues/476
	dir := o.Dir
	pomName := filepath.Join(dir, "pom.xml")
	exists, err := util.FileExists(pomName)
	if err != nil {
		return err
	}
	if exists {
		args = []string{"create", "--pack=github.com/jenkins-x/draft-repo/packs/java"}
	}
	e := exec.Command("draft", args...)
	e.Dir = dir
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err = e.Run()
	if err != nil {
		// lets ignore draft errors as sometimes it can't find a pack - e.g. for environments
		o.Printf(util.ColorWarning("WARNING: Failed to run draft create in %s due to %s"), dir, err)
		//return fmt.Errorf("Failed to run draft create in %s due to %s", dir, err)
	}
	// chart expects folder name to be the same as app name
	oldChartsDir := filepath.Join(dir, "charts", "java")
	newChartsDir := filepath.Join(dir, "charts", o.AppName)
	exists, err = util.FileExists(oldChartsDir)
	if err != nil {
		return err
	}
	if exists {
		os.Rename(oldChartsDir, newChartsDir)
	}

	// now update the chart.yaml
	err = o.addAppNameToGeneratedFile("Chart.yaml", "name: ", o.AppName)
	if err != nil {
		return err
	}

	// now update the makefile
	err = o.addAppNameToGeneratedFile("Makefile", "NAME := ", o.AppName)
	if err != nil {
		return err
	}

	// now update the helm values which contains the image name
	err = o.addAppNameToGeneratedFile("values.yaml", "  repository: ", fmt.Sprintf("%s/%s", "jenkinsx", o.AppName))
	if err != nil {
		return err
	}

	err = gits.GitAdd(dir, "*")
	if err != nil {
		return err
	}
	err = gits.GitCommitIfChanges(dir, "Draft create")
	if err != nil {
		return err
	}
	return nil
}

func (o *ImportOptions) DefaultJenkinsfile() error {

	dir := o.Dir
	name := filepath.Join(dir, "Jenkinsfile")
	exists, err := util.FileExists(name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	data := []byte(fmt.Sprintf(defaultJenkinsfile, o.AppName, o.AppName, o.AppName))
	err = ioutil.WriteFile(name, data, DefaultWritePermissions)
	if err != nil {
		return fmt.Errorf("Failed to write %s due to %s", name, err)
	}
	err = gits.GitAdd(dir, "Jenkinsfile")
	if err != nil {
		return err
	}
	err = gits.GitCommitIfChanges(dir, "Added default Jenkinsfile pipeline")
	if err != nil {
		return err
	}
	return nil
}

func (o *ImportOptions) DefaultDockerfile() error {

	dir := o.Dir
	name := filepath.Join(dir, "Dockerfile.release")
	exists, err := util.FileExists(name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	data := []byte(defaultDockerfile)
	err = ioutil.WriteFile(name, data, DefaultWritePermissions)
	if err != nil {
		return fmt.Errorf("Failed to write %s due to %s", name, err)
	}
	err = gits.GitAdd(dir, "Dockerfile.release")
	if err != nil {
		return err
	}
	err = gits.GitCommitIfChanges(dir, "Added Release Dockerfile pipeline")
	if err != nil {
		return err
	}
	return nil
}

func (o *ImportOptions) CreateNewRemoteRepository() error {
	f := o.Factory
	authConfigSvc, err := f.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	dir := o.Dir
	_, defaultRepoName := filepath.Split(dir)

	details, err := gits.PickNewGitRepository(o.Out, authConfigSvc, defaultRepoName)
	if err != nil {
		return err
	}
	repo, err := details.CreateRepository()
	if err != nil {
		return err
	}
	o.GitProvider = details.GitProvider

	o.RepoURL = repo.CloneURL
	pushGitURL, err := gits.GitCreatePushURL(repo.CloneURL, details.User)
	if err != nil {
		return err
	}
	err = gits.GitCmd(dir, "remote", "add", "origin", pushGitURL)
	if err != nil {
		return err
	}
	err = gits.GitCmd(dir, "push", "-u", "origin", "master")
	if err != nil {
		return err
	}
	o.Printf("Pushed git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))
	return nil
}

func (o *ImportOptions) CloneRepository() error {
	url := o.RepoURL
	if url == "" {
		return fmt.Errorf("No git repository URL defined!")
	}
	gitInfo, err := gits.ParseGitURL(url)
	if err != nil {
		return fmt.Errorf("Failed to parse git URL %s due to: %s", url, err)
	}
	if gitInfo.Host == gits.GitHubHost && strings.HasPrefix(gitInfo.Scheme, "http") {
		if !strings.HasSuffix(url, ".git") {
			url += ".git"
		}
		o.RepoURL = url
	}
	cloneDir, err := util.CreateUniqueDirectory(o.Dir, gitInfo.Name, util.MaximumNewDirectoryAttempts)
	if err != nil {
		return err
	}
	err = gits.GitClone(url, cloneDir)
	if err != nil {
		return err
	}
	o.Dir = cloneDir
	return nil
}

// DiscoverGit checks if there is a git clone or prompts the user to import it
func (o *ImportOptions) DiscoverGit() error {
	if !o.DisableDotGitSearch {
		root, gitConf, err := gits.FindGitConfigDir(o.Dir)
		if err != nil {
			return err
		}
		if root != "" {
			o.Dir = root
			o.GitConfDir = gitConf
			return nil
		}
	}

	dir := o.Dir
	if dir == "" {
		return fmt.Errorf("No directory specified!")
	}

	// lets prompt the user to initiialse the git repository
	o.Printf("The directory %s is not yet using git\n", util.ColorInfo(dir))
	flag := false
	prompt := &survey.Confirm{
		Message: "Would you like to initialise git now?",
		Default: true,
	}
	err := survey.AskOne(prompt, &flag, nil)
	if err != nil {
		return err
	}
	if !flag {
		return fmt.Errorf("Please initialise git yourself then try again")
	}
	err = gits.GitInit(dir)
	if err != nil {
		return err
	}
	o.GitConfDir = filepath.Join(dir, ".git/config")
	err = o.DefaultGitIgnore()
	if err != nil {
		return err
	}
	err = gits.GitAdd(dir, ".gitignore")
	if err != nil {
		return err
	}
	err = gits.GitAdd(dir, "*")
	if err != nil {
		return err
	}

	err = gits.GitStatus(dir)
	if err != nil {
		return err
	}

	message := ""
	messagePrompt := &survey.Input{
		Message: "Commit message: ",
		Default: "Initial import",
	}
	err = survey.AskOne(messagePrompt, &message, nil)
	if err != nil {
		return err
	}
	err = gits.GitCommitIfChanges(dir, message)
	if err != nil {
		return err
	}
	o.Printf("\nGit repository created\n")
	return nil
}

// DiscoverGit checks if there is a git clone or prompts the user to import it
func (o *ImportOptions) DefaultGitIgnore() error {
	name := filepath.Join(o.Dir, ".gitignore")
	exists, err := util.FileExists(name)
	if err != nil {
		return err
	}
	if !exists {
		data := []byte(defaultGitIgnoreFile)
		err = ioutil.WriteFile(name, data, DefaultWritePermissions)
		if err != nil {
			return fmt.Errorf("Failed to write %s due to %s", name, err)
		}
	}
	return nil
}

// DiscoverRemoteGitURL finds the git url by looking in the directory
// and looking for a .git/config file
func (o *ImportOptions) DiscoverRemoteGitURL() error {
	gitConf := o.GitConfDir
	if gitConf == "" {
		return fmt.Errorf("No GitConfDir defined!")
	}
	cfg := gitcfg.NewConfig()
	data, err := ioutil.ReadFile(gitConf)
	if err != nil {
		return fmt.Errorf("Failed to load %s due to %s", gitConf, err)
	}

	err = cfg.Unmarshal(data)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal %s due to %s", gitConf, err)
	}
	remotes := cfg.Remotes
	if len(remotes) == 0 {
		return nil
	}
	url := gits.GetRemoteUrl(cfg, "origin")
	if url == "" {
		url = gits.GetRemoteUrl(cfg, "upstream")
		if url == "" {
			url, err = o.pickRemoteURL(cfg)
			if err != nil {
				return err
			}
		}
	}
	if url != "" {
		o.RepoURL = url
	}
	return nil
}

func (o *ImportOptions) DoImport() error {
	if o.Jenkins == nil {
		jenkins, err := o.Factory.GetJenkinsClient()
		if err != nil {
			return err
		}
		o.Jenkins = jenkins
	}
	gitURL := o.RepoURL
	gitProvider := o.GitProvider
	if gitProvider == nil {
		p, err := o.gitProviderForURL(gitURL, "user name to register webhook")
		if err != nil {
			return err
		}
		gitProvider = p
	}

	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	jenkinsfile := o.Jenkinsfile
	if jenkinsfile == "" {
		jenkinsfile = jenkins.DefaultJenkinsfile
	}
	return jenkins.ImportProject(o.Out, o.Jenkins, gitURL, jenkinsfile, o.Credentials, false, gitProvider, authConfigSvc)
}

func (o *ImportOptions) pickRemoteURL(config *gitcfg.Config) (string, error) {
	urls := []string{}
	if config.Remotes != nil {
		for _, r := range config.Remotes {
			if r.URLs != nil {
				for _, u := range r.URLs {
					urls = append(urls, u)
				}
			}
		}
	}
	if len(urls) == 1 {
		return urls[0], nil
	}
	url := ""
	if len(urls) > 1 {
		prompt := &survey.Select{
			Message: "Choose a remote git URL:",
			Options: urls,
		}
		err := survey.AskOne(prompt, &url, nil)
		if err != nil {
			return "", err
		}
	}
	return url, nil
}
func (o *ImportOptions) addAppNameToGeneratedFile(filename, field, value string) error {
	dir := filepath.Join(o.Dir, "charts", o.AppName)
	file := filepath.Join(dir, filename)
	exists, err := util.FileExists(file)
	if err != nil {
		return err
	}
	if !exists {
		// no file so lets ignore this
		return nil
	}
	input, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, field) {
			lines[i] = fmt.Sprintf("%s%s", field, value)
		}
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(file, []byte(output), 0644)
	if err != nil {
		return err
	}
	return nil
}

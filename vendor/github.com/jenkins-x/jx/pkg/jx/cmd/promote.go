package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/blang/semver"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

const (
	optionEnvironment = "environment"
)

// PromoteOptions containers the CLI options
type PromoteOptions struct {
	CommonOptions

	Namespace         string
	Environment       string
	Application       string
	Version           string
	LocalHelmRepoName string
	HelmRepositoryURL string
	Preview           bool
	NoHelmUpdate      bool
	AllAutomatic      bool
}

var (
	promote_long = templates.LongDesc(`
		Promotes a version of an application to an environment.
`)

	promote_example = templates.Examples(`
		# Promote a version of an application to staging
		jx promote myapp --version 1.2.3 --env staging
	`)
)

// NewCmdPromote creates the new command for: jx get prompt
func NewCmdPromote(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &PromoteOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "promote [application]",
		Short:   "Promotes a version of an application to an environment",
		Long:    promote_long,
		Example: promote_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The Namespace to promote to")
	cmd.Flags().StringVarP(&options.Environment, optionEnvironment, "e", "", "The Environment to promote to")
	cmd.Flags().StringVarP(&options.Application, "app", "a", "", "The Application to promote")
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The Version to promote")
	cmd.Flags().StringVarP(&options.LocalHelmRepoName, "helm-repo-name", "r", kube.LocalHelmRepoName, "The name of the helm repository that contains the app")
	cmd.Flags().StringVarP(&options.HelmRepositoryURL, "helm-repo-url", "u", helm.DefaultHelmRepositoryURL, "The Helm Repository URL to use for the App")
	cmd.Flags().BoolVarP(&options.Preview, "preview", "p", false, "Whether to create a new Preview environment for the app")
	cmd.Flags().BoolVarP(&options.NoHelmUpdate, "no-helm-update", "", false, "Allows the 'helm repo update' command if you are sure your local helm cache is up to date with the version you wish to promote")
	cmd.Flags().BoolVarP(&options.AllAutomatic, "all-auto", "", false, "Promote to all automatic environments in order")

	return cmd
}

// Run implements this command
func (o *PromoteOptions) Run() error {
	app := o.Application
	if app == "" {
		args := o.Args
		if len(args) == 0 {
			var err error
			app, err = o.discoverAppName()
			if err != nil {
				return err
			}
		} else {
			app = args[0]
		}
	}
	o.Application = app

	if o.AllAutomatic {
		return o.PromoteAllAutomatic()

	}
	targetNS, env, err := o.GetTargetNamespace(o.Namespace, o.Environment)
	if err != nil {
		return err
	}
	return o.Promote(targetNS, env, true)
}

func (o *PromoteOptions) PromoteAllAutomatic() error {
	kubeClient, currentNs, err := o.KubeClient()
	if err != nil {
		return err
	}
	team, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return err
	}
	jxClient, _, err := o.JXClient()
	if err != nil {
		return err
	}
	envs, err := jxClient.JenkinsV1().Environments(team).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	environments := envs.Items
	if len(environments) == 0 {
		return fmt.Errorf("No Environments have been created yet in team %s. Please create some via 'jx create env'", team)
	}
	kube.SortEnvironments(environments)

	for _, env := range environments {
		if env.Spec.PromotionStrategy == v1.PromotionStrategyTypeAutomatic {
			ns := env.Spec.Namespace
			err = o.Promote(ns, &env, false)
			if err != nil {
				return err
			}
			err = o.WaitForPromotion(ns, &env)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *PromoteOptions) Promote(targetNS string, env *v1.Environment, warnIfAuto bool) error {
	app := o.Application
	version := o.Version
	info := util.ColorInfo
	if version == "" {
		o.Printf("Promoting latest version of app %s to namespace %s\n", info(app), info(targetNS))
	} else {
		o.Printf("Promoting app %s version %s to namespace %s\n", info(app), info(version), info(targetNS))
	}

	if warnIfAuto && env != nil && env.Spec.PromotionStrategy == v1.PromotionStrategyTypeAutomatic {
		o.Printf("%s", util.ColorWarning("WARNING: The Environment %s is setup to promote automatically as part of the CI / CD Pipelines.\n\n", env.Name))

		confirm := &survey.Confirm{
			Message: "Do you wish to promote anyway? :",
			Default: false,
		}
		flag := false
		err := survey.AskOne(confirm, &flag, nil)
		if err != nil {
			return err
		}
		if !flag {
			return nil
		}
	}
	if env != nil {
		source := &env.Spec.Source
		if source.URL != "" {
			return o.PromoteViaPullRequest(env)
		}
	}
	fullAppName := app
	if o.LocalHelmRepoName != "" {
		fullAppName = o.LocalHelmRepoName + "/" + app
	}

	// lets do a helm update to ensure we can find the latest version
	if !o.NoHelmUpdate {
		o.Printf("Updating the helm repositories to ensure we can find the latest versions...")
		err := o.runCommand("helm", "repo", "update")
		if err != nil {
			return err
		}
	}
	releaseName := targetNS + "-" + app
	if version != "" {
		return o.runCommand("helm", "upgrade", "--install", "--namespace", targetNS, "--version", version, releaseName, fullAppName)
	}
	return o.runCommand("helm", "upgrade", "--install", "--namespace", targetNS, releaseName, fullAppName)
}

func (o *PromoteOptions) PromoteViaPullRequest(env *v1.Environment) error {
	source := &env.Spec.Source
	gitURL := source.URL
	if gitURL == "" {
		return fmt.Errorf("No source git URL")
	}
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return err
	}

	environmentsDir, err := util.EnvironmentsDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(environmentsDir, gitInfo.Organisation, gitInfo.Name)

	// now lets clone the fork and push it...
	exists, err := util.FileExists(dir)
	if err != nil {
		return err
	}
	app := o.Application
	version := o.Version
	versionName := version
	if versionName == "" {
		versionName = "latest"
	}

	branchName := gits.ConvertToValidBranchName("promote-" + app + "-" + versionName)
	base := source.Ref
	if base == "" {
		base = "master"
	}

	if exists {
		// lets check the git remote URL is setup correctly
		err = gits.SetRemoteURL(dir, "origin", gitURL)
		if err != nil {
			return err
		}
		err = gits.GitCmd(dir, "stash")
		if err != nil {
			return err
		}
		err = gits.GitCmd(dir, "checkout", base)
		if err != nil {
			return err
		}
		err = gits.GitCmd(dir, "pull")
		if err != nil {
			return err
		}
	} else {
		err := os.MkdirAll(dir, DefaultWritePermissions)
		if err != nil {
			return fmt.Errorf("Failed to create directory %s due to %s", dir, err)
		}
		err = gits.GitClone(gitURL, dir)
		if err != nil {
			return err
		}
		if base != "master" {
			err = gits.GitCmd(dir, "checkout", base)
			if err != nil {
				return err
			}
		}

		// TODO lets fork if required???
		/*
			pushGitURL, err := gits.GitCreatePushURL(gitURL, details.User)
			if err != nil {
				return err
			}
			err = gits.GitCmd(dir, "remote", "add", "upstream", forkEnvGitURL)
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
		*/
	}
	branchNames, err := gits.GitGetRemoteBranchNames(dir, "remotes/origin/")
	if err != nil {
		return fmt.Errorf("Failed to load remote branch names: %s", err)
	}
	o.Printf("Found remote branch names %s\n", strings.Join(branchNames, ", "))
	if util.StringArrayIndex(branchNames, branchName) >= 0 {
		// lets append a UUID as the branch name already exists
		branchName += "-" + string(uuid.NewUUID())
	}
	err = gits.GitCmd(dir, "branch", branchName)
	if err != nil {
		return err
	}
	err = gits.GitCmd(dir, "checkout", branchName)
	if err != nil {
		return err
	}

	requirementsFile, err := helm.FindRequirementsFileName(dir)
	if err != nil {
		return err
	}
	requirements, err := helm.LoadRequirementsFile(requirementsFile)
	if err != nil {
		return err
	}
	if version == "" {
		version, err = o.findLatestVersion(app)
		if err != nil {
			return err
		}
	}
	requirements.SetAppVersion(app, version, o.HelmRepositoryURL)
	err = helm.SaveRequirementsFile(requirementsFile, requirements)

	err = gits.GitCmd(dir, "add", "*", "*/*")
	if err != nil {
		return err
	}
	changed, err := gits.HasChanges(dir)
	if err != nil {
		return err
	}
	if !changed {
		o.Printf("%s\n", util.ColorWarning("No changes made to the GitOps Environment source code. Must be already on version!"))
		return nil
	}
	message := fmt.Sprintf("Promote %s to version %s", app, versionName)
	err = gits.GitCommit(dir, message)
	if err != nil {
		return err
	}
	err = gits.GitPush(dir)
	if err != nil {
		return err
	}

	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	provider, err := gitInfo.PickOrCreateProvider(authConfigSvc, "user name to submit the Pull Request")
	if err != nil {
		return err
	}

	gha := &gits.GitPullRequestArguments{
		Owner: gitInfo.Organisation,
		Repo:  gitInfo.Name,
		Title: app + " to " + versionName,
		Body:  message,
		Base:  base,
		Head:  branchName,
	}

	pr, err := provider.CreatePullRequest(gha)
	if err != nil {
		return err
	}
	o.Printf("Created Pull Request: %s\n\n", util.ColorInfo(pr.URL))
	return nil
}

func (o *PromoteOptions) GetTargetNamespace(ns string, env string) (string, *v1.Environment, error) {
	kubeClient, currentNs, err := o.KubeClient()
	if err != nil {
		return "", nil, err
	}
	team, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return "", nil, err
	}

	jxClient, _, err := o.JXClient()
	if err != nil {
		return "", nil, err
	}

	m, envNames, err := kube.GetEnvironments(jxClient, team)
	if err != nil {
		return "", nil, err
	}
	if len(envNames) == 0 {
		return "", nil, fmt.Errorf("No Environments have been created yet in team %s. Please create some via 'jx create env'", team)
	}

	var envResource *v1.Environment
	targetNS := currentNs
	if env != "" {
		envResource = m[env]
		if envResource == nil {
			return "", nil, util.InvalidOption(optionEnvironment, env, envNames)
		}
		targetNS = envResource.Spec.Namespace
		if targetNS == "" {
			return "", nil, fmt.Errorf("Environment %s does not have a namspace associated with it!", env)
		}
	} else if ns != "" {
		targetNS = ns
	}

	labels := map[string]string{}
	annotations := map[string]string{}
	err = kube.EnsureNamespaceCreated(kubeClient, targetNS, labels, annotations)
	if err != nil {
		return "", nil, err
	}
	return targetNS, envResource, nil
}

func (options *PromoteOptions) discoverAppName() (string, error) {
	answer := ""
	dir, err := os.Getwd()
	if err != nil {
		return answer, err
	}

	root, gitConf, err := gits.FindGitConfigDir(dir)
	if err != nil {
		return answer, err
	}
	if root != "" {
		url, err := gits.DiscoverRemoteGitURL(gitConf)
		if err != nil {
			return answer, err
		}
		gitInfo, err := gits.ParseGitURL(url)
		if err != nil {
			return answer, err
		}
		answer = gitInfo.Name
	}
	return answer, nil
}

func (options *PromoteOptions) WaitForPromotion(ns string, env *v1.Environment) error {
	// TODO
	return nil
}

func (o *PromoteOptions) findLatestVersion(app string) (string, error) {
	output, err := o.getCommandOutput("", "helm", "search", app, "--versions")
	if err != nil {
		return "", err
	}
	var maxSemVer *semver.Version
	maxString := ""
	for i, line := range strings.Split(output, "\n") {
		if i == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 1 {
			v := fields[1]
			if v != "" {
				sv, err := semver.Parse(v)
				if err != nil {
					o.warnf("Invalid semantic version: %s %s\n", v, err)
				} else {
					if maxSemVer == nil || maxSemVer.Compare(sv) > 0 {
						maxSemVer = &sv
					}
				}
				if maxString == "" || strings.Compare(v, maxString) > 0 {
					maxString = v
				}
			}
		}
	}
	if maxSemVer != nil {
		return maxSemVer.String(), nil
	}
	if maxString == "" {
		return "", fmt.Errorf("Could not find a version of app %s in the helm repositories", app)
	}
	return maxString, nil
}

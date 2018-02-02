package cmd

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"os"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

// CommonOptions contains common options and helper methods
type CommonOptions struct {
	Factory   cmdutil.Factory
	Out       io.Writer
	Err       io.Writer
	Cmd       *cobra.Command
	Args      []string
	BatchMode bool

	// common cached clients
	kubeClient       *kubernetes.Clientset
	currentNamespace string
	jxClient         *versioned.Clientset
	jenkinsClient    *gojenkins.Jenkins
}

func (c *CommonOptions) CreateTable() table.Table {
	return c.Factory.CreateTable(c.Out)
}

func (c *CommonOptions) Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(c.Out, format, a...)
}

func (o *CommonOptions) runCommandFromDir(dir, name string, args ...string) error {
	e := exec.Command(name, args...)
	if dir != "" {
		e.Dir = dir
	}
	e.Stdout = o.Out
	e.Stderr = o.Err
	err := e.Run()
	if err != nil {
		o.Printf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err
}

func (o *CommonOptions) runCommand(name string, args ...string) error {
	e := exec.Command(name, args...)
	e.Stdout = o.Out
	e.Stderr = o.Err
	err := e.Run()
	if err != nil {
		o.Printf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err
}

func (o *CommonOptions) runCommandInteractive(interactive bool, name string, args ...string) error {
	e := exec.Command(name, args...)
	e.Stdout = o.Out
	e.Stderr = o.Err
	if interactive {
		e.Stdin = os.Stdin
	}
	err := e.Run()
	if err != nil {
		o.Printf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err
}

// getCommandOutput evaluates the given command and returns the trimmed output
func (o *CommonOptions) getCommandOutput(dir string, name string, args ...string) (string, error) {
	e := exec.Command(name, args...)
	if dir != "" {
		e.Dir = dir
	}
	data, err := e.CombinedOutput()
	text := string(data)
	text = strings.TrimSpace(text)
	return text, err
}

func (options *CommonOptions) addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&options.BatchMode, "batch-mode", "b", false, "In batch mode the command never prompts for user input")
	options.Cmd = cmd
}

func (o *CommonOptions) KubeClient() (*kubernetes.Clientset, string, error) {
	if o.kubeClient == nil {
		kubeClient, currentNs, err := o.Factory.CreateClient()
		if err != nil {
			return nil, "", err
		}
		o.kubeClient = kubeClient
		o.currentNamespace = currentNs
	}
	return o.kubeClient, o.currentNamespace, nil
}

func (o *CommonOptions) JXClient() (*versioned.Clientset, string, error) {
	if o.jxClient == nil {
		jxClient, ns, err := o.Factory.CreateJXClient()
		if err != nil {
			return nil, ns, err
		}
		o.jxClient = jxClient
		if o.currentNamespace == "" {
			o.currentNamespace = ns
		}
	}
	return o.jxClient, o.currentNamespace, nil
}

func (o *CommonOptions) JenkinsClient() (*gojenkins.Jenkins, error) {
	if o.jenkinsClient == nil {
		jenkins, err := o.Factory.GetJenkinsClient()
		if err != nil {
			return nil, err
		}
		o.jenkinsClient = jenkins
	}
	return o.jenkinsClient, nil
}

func (o *CommonOptions) TeamAndEnvironmentNames() (string, string, error) {
	kubeClient, currentNs, err := o.KubeClient()
	if err != nil {
		return "", "", err
	}
	return kube.GetDevNamespace(kubeClient, currentNs)
}

// warnf generates a warning
func (o *CommonOptions) warnf(format string, a ...interface{}) {
	o.Printf(util.ColorWarning("WARNING: "+format), a...)
}

// gitProviderForURL returns a GitProvider for the given git URL
func (o *CommonOptions) gitProviderForURL(gitURL string, message string) (gits.GitProvider, error) {
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return nil, err
	}
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return nil, err
	}
	return gitInfo.PickOrCreateProvider(authConfigSvc, message)
}

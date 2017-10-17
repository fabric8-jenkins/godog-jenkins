package utils

import (
	"errors"
	"fmt"
	"github.com/fabric8-jenkins/golang-jenkins"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func GetJenkinsClient() (*gojenkins.Jenkins, error) {
	url := os.Getenv("BDD_JENKINS_URL")
	if url == "" {
		return nil, errors.New("no BDD_JENKINS_URL env var set")
	}
	username := os.Getenv("BDD_JENKINS_USERNAME")
	token := os.Getenv("BDD_JENKINS_TOKEN")

	bearerToken := os.Getenv("BDD_JENKINS_BEARER_TOKEN")
	if bearerToken == "" && (token == "" || username == "") {
		return nil, errors.New("no BDD_JENKINS_TOKEN or BDD_JENKINS_BEARER_TOKEN && BDD_JENKINS_USERNAME env var set")
	}

	auth := &gojenkins.Auth{
		Username:    username,
		ApiToken:    token,
		BearerToken: bearerToken,
	}
	return gojenkins.NewJenkins(auth, url), nil
}

func GetFileAsString(path string) (string, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("No file found at path %s", path)
	}

	return string(buf), nil
}

type MultiError struct {
	Errors []error
}

func RetryAfter(attempts int, callback func() error, d time.Duration) (err error) {
	m := MultiError{}
	for i := 0; i < attempts; i++ {
		err = callback()
		if err == nil {
			return nil
		}
		m.Collect(err)
		time.Sleep(d)
	}
	return m.ToError()
}

func (m *MultiError) Collect(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

func (m MultiError) ToError() error {
	if len(m.Errors) == 0 {
		return nil
	}

	errStrings := []string{}
	for _, err := range m.Errors {
		errStrings = append(errStrings, err.Error())
	}
	return fmt.Errorf(strings.Join(errStrings, "\n"))
}

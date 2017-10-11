package utils

import (
	"errors"
	"fmt"
	"github.com/fabric8-jenkins/golang-jenkins"
	"io/ioutil"
	"os"
)

func GetJenkinsClient() (*gojenkins.Jenkins, error) {
	url := os.Getenv("BDD_JENKINS_URL")
	if url == "" {
		return nil, errors.New("no BDD_JENKINS_URL env var set")
	}
	username := os.Getenv("BDD_JENKINS_USERNAME")
	if username == "" {
		return nil, errors.New("no BDD_JENKINS_USERNAME env var set")
	}
	token := os.Getenv("BDD_JENKINS_TOKEN")
	if token == "" {
		return nil, errors.New("no BDD_JENKINS_TOKEN env var set")
	}

	auth := &gojenkins.Auth{
		Username: username,
		ApiToken: token,
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

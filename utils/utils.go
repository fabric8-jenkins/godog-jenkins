package utils

import (
	"github.com/fabric8io/golang-jenkins"
	"os"
	"errors"
)

func GetJenkinsClient() (*gojenkins.Jenkins, error){
	url := os.Getenv("BDD_JENKINS_URL")
	if url == ""{
		return nil, errors.New("no BDD_JENKINS_URL env var set")
	}
	username := os.Getenv("BDD_JENKINS_USERNAME")
	if username == ""{
		return nil, errors.New("no BDD_JENKINS_USERNAME env var set")
	}
	token := os.Getenv("BDD_JENKINS_TOKEN")
	if token == ""{
		return nil, errors.New("no BDD_JENKINS_TOKEN env var set")
	}

	auth := &gojenkins.Auth{
		Username: username,
		ApiToken: token,
	}
	return gojenkins.NewJenkins(auth, url), nil
}
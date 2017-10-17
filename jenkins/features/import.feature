Feature: import GitHub Repo
  In order to provide quick fabric8 jenkins experience
  As a project admin
  I need to be able to import a Github repo into Jenkins

  Scenario: Import repo and run sample pipeline
    Given there is a fabric8-import job
    When we import the "fabric8-quickstarts-tests/spring-boot-http-booster" GitHub repo selecting "ReleaseAndStage" pipeline
    And we merge the PR which is created
    Then there should be a "fabric8-quickstarts-tests/spring-boot-http-booster" job that completes successfully

  Scenario: Delete organisation
    Given there is a job called "fabric8-quickstarts-tests"
    When I delete the "fabric8-quickstarts-tests" job
    Then there should not be a "fabric8-quickstarts-tests" job
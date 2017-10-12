Feature: import GitHub Repo
  In order to provide CI/CD for a GitHub organisation
  As a project admin
  I need to be able to import Github repos into Jenkins

  Scenario: Import organisation
    Given there are no jobs called "fabric8-quickstarts-tests"
    When I import the "fabric8-quickstarts-tests" GitHub organisation
    Then there should be a "fabric8-quickstarts-tests" job and more than 1 multibranch job

  Scenario: Trigger an organisation scan
    Given there is a "fabric8-quickstarts-tests" job
    When I trigger the "fabric8-quickstarts-tests" job
    Then the organisation scan for "fabric8-quickstarts-tests" completes

#  Scenario: Multi branch job
#    Given there is a job "spring-boot-http-booster1"
#    When I trigger the job "spring-boot-http-booster1"
#    Then the job "spring-boot-http-booster1" is successful

  Scenario: Delete organisation
    Given there are is a job called "fabric8-quickstarts-tests"
    When I delete the "fabric8-quickstarts-tests" job
    Then there should not be a "fabric8-quickstarts-tests" job
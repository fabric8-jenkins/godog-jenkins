Feature: import GitHub Repo
  In order to provide CI/CD for a GitHub organisation
  As a project admin
  I need to be able to import Github repos into Jenkins

  Scenario: Import organisation
    Given there are no jobs called "fabric8-quicktart-test"
    When I import the "fabric8-quicktart-test" GitHub organisation
    Then there should be a "fabric8-quicktart-test" job and more than 1 multibranch job

  Scenario: Delete organisation
    Given there are is a job called "fabric8-quicktart-test"
    When I delete the "fabric8-quicktart-test" job
    Then there should not be a "fabric8-quicktart-test" job
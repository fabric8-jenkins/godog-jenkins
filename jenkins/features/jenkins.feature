Feature: import GitHub Repo
  In order to provide CI/CD for a GitHub organisation
  As a project admin
  I need to be able to import Github repos into Jenkins

  Scenario: Import organisation
    Given there are 0 Jobs
    When I import the fabric8-quicktart-test org
    Then there should be a GitHub organisation job and > 1 multibranch jobs
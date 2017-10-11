Feature: fork GitHub Repo
  In order to test importing a quickstart
  As a tester
  I need to be able to fork a quickstart GitHub repository to a clean fork

  Scenario: Fork repository
    Given there is no fork of "fabric8-quickstarts/spring-boot-webmvc"
    When I fork the "fabric8-quickstarts/spring-boot-webmvc" GitHub organisation
    Then there should be a "jstrachan-testing/spring-boot-webmvc" fork which has the same last commit as "fabric8-quickstarts/spring-boot-webmvc"


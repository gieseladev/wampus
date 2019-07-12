workflow "Publish Docker image on release" {
  resolves = ["Push docker image"]
  on = "release"
}

action "Build Docker image" {
  uses = "actions/docker/cli@master"
  args = "build --tag gieseladev/wampus ."
}

action "Docker Registry login" {
  uses = "actions/docker/login@master"
  needs = ["Build Docker image"]
  secrets = ["DOCKER_PASSWORD", "DOCKER_USERNAME"]
}

action "Push docker image" {
  uses = "actions/docker/cli@master"
  needs = ["Docker Registry login"]
  args = "push gieseladev/wampus"
}

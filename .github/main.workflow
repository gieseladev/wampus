workflow "Publish Docker image on release" {
  resolves = [
    "Docker push latest",
    "Docker push release",
  ]
  on = "release"
}

action "Docker login" {
  uses = "actions/docker/login@master"
  secrets = ["DOCKER_PASSWORD", "DOCKER_USERNAME"]
}

action "Docker build" {
  uses = "actions/docker/cli@master"
  needs = ["Docker login"]
  args = "build --tag wampus --file build/Dockerfile ."
}

action "Docker tag latest" {
  uses = "actions/docker/tag@master"
  needs = ["Docker build"]
  args = "wampus gieseladev/wampus:latest"
}

action "Docker tag release" {
  uses = "actions/docker/tag@master"
  needs = ["Docker build"]
  args = "wampus gieseladev/wampus:$GITHUB_REF"
}

action "Docker push latest" {
  uses = "actions/docker/cli@master"
  needs = ["Docker tag latest"]
  args = "push gieseladev/wampus:latest"
}

action "Docker push release" {
  uses = "actions/docker/cli@master"
  needs = ["Docker tag release"]
  args = "push gieseladev/wampus:$GITHUB_REF"
}

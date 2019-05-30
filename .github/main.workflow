workflow "Release" {
  on = "push"
  resolves = ["goreleaser"]
}

action "is-tag" {
  uses = "actions/bin/filter@master"
  args = "tag"
}

action "goreleaser" {
  uses = "docker://goreleaser/goreleaser"
  secrets = [
    "GITHUB_TOKEN",
  ]

  # either GITHUB_TOKEN or GORELEASER_GITHUB_TOKEN is required

  args = "release"
  needs = ["is-tag"]
}

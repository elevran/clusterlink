workflow "Hugo Link Check" {
  resolves = "linkcheck"
  on = ["pull_request", "workflow_dispatch"]

}

action "filter-to-pr-open-synced" {
  uses = "actions/bin/filter@master"
  args = "action 'opened|synchronize'"
}

action "linkcheck" {
  uses = "marccampbell/hugo-linkcheck-action@v0.1.3"
  needs = "filter-to-pr-open-synced"
  secrets = ["GITHUB_TOKEN"]
  env = {
    HUGO_CONFIG = "./website/config.toml"
    HUGO_ROOT = "./website"
    HUGO_CONTENT_ROOT = "./website/content"
    HUGO_FINAL_URL = "https://clusterlink.net"
  }
}
resource "digitalocean_app" "domaininfo" {
  spec {
    name   = "domaininfo"
    region = "nyc3"
    domain {
      name = "whoiswrapper.com"
    }

    service {
      name                  = "domaininfo-service"
      build_command         = "go test && go build ."
      run_command           = "./domainInfo"

      git {
        repo_clone_url = "https://github.com/Sackbuoy/domainInfo"
        branch         = "main"
      }
    }
  }
}
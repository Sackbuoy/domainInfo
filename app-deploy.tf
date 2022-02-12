resource "digitalocean_app" "domaininfo" {
  spec {
    name   = "domaininfo"
    region = "nyc3"

    service {
      name                  = "domaininfo-service"
      build_command         = "go test && go build ."
      run_command           = "./domainInfo"

      github {
        repo           = "Sackbuoy/domainInfo"
        branch         = "main"
        deploy_on_push = true
      }
    }
  }
}
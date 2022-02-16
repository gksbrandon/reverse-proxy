provider "helm" {
  kubernetes {
    host                   = data.aws_eks_cluster.dev-cluster.endpoint
    cluster_ca_certificate = base64decode(data.aws_eks_cluster.dev-cluster.certificate_authority.0.data)
    exec {
      api_version = "client.authentication.k8s.io/v1alpha1"
      args        = ["eks", "get-token", "--cluster-name", data.aws_eks_cluster.dev-cluster.name]
      command     = "aws"
    }
  }
}

resource "helm_release" "reverse-proxy" {
  name       = "reverse-proxy"
  repository = "https://gksbrandon.github.io/reverse-proxy/charts"
  chart      = "reverse-proxy"
}

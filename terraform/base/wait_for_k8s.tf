
# Readiness Probe for Kubernetes API
# This resource waits for the Kubernetes API server to become responsive
# before allowing dependent resources (like Helm charts) to be created.

resource "null_resource" "wait_for_k8s_api" {
  depends_on = [module.talos_cluster]

  provisioner "local-exec" {
    command = <<-EOT
      end_time=$(( $(date +%s) + 300 )) # 5 minutes from now
      while true; do
        http_code=$(curl -k -s -o /dev/null -w "%%{http_code}" "$K8S_ENDPOINT/version")
        if [ "$http_code" = "401" ] || [ "$http_code" = "200" ]; then
          echo "Kubernetes API is ready at $K8S_ENDPOINT (HTTP status: $http_code)."
          break
        fi

        if [ "$(date +%s)" -gt "$end_time" ]; then
          echo "Timeout: Kubernetes API at $K8S_ENDPOINT did not become ready after 5 minutes. Last HTTP code: $http_code"
          exit 1
        fi

        echo "Waiting for Kubernetes API at $K8S_ENDPOINT to be ready... (last HTTP code: $http_code)"
        sleep 10
      done
    EOT

    environment = {
      K8S_ENDPOINT = module.talos_cluster.kubernetes_host
    }
  }
}

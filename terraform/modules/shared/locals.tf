# ============================================================================
# SHARED MODULE - DRY CONFIGURATIONS
# ============================================================================
# This module centralizes all reusable configurations to eliminate duplication
# across Terraform codebase. Used by all modules (talos, cilium, argocd).

locals {
  # --------------------------------------------------------------------------
  # CHART VERSIONS - Single source of truth
  # --------------------------------------------------------------------------
  chart_versions = {
    cilium       = "1.18.3"
    argocd       = "7.7.7"
    traefik      = "25.0.0"
    cert_manager = "v1.14.4"
  }

  # --------------------------------------------------------------------------
  # CONTROL PLANE TOLERATIONS - Used by Hubble, ArgoCD, etc.
  # --------------------------------------------------------------------------
  control_plane_tolerations = [
    {
      key      = "node-role.kubernetes.io/control-plane"
      operator = "Exists"
      effect   = "NoSchedule"
    }
  ]

  # --------------------------------------------------------------------------
  # CILIUM CONFIGURATION
  # --------------------------------------------------------------------------
  cilium_config = {
    # Agent capabilities (required by Talos)
    # Keep all capabilities from working configuration to ensure Cilium starts
    agent_capabilities = {
      add = [
        "CHOWN",
        "KILL",
        "NET_ADMIN",
        "NET_RAW",
        "IPC_LOCK",
        "SYS_ADMIN",
        "SYS_RESOURCE",
        "DAC_OVERRIDE",
        "FOWNER",
        "SETGID",
        "SETUID"
      ]
      drop = ["ALL"]
    }

    # Clean state capabilities
    clean_capabilities = {
      add  = ["NET_ADMIN", "SYS_ADMIN", "SYS_RESOURCE"]
      drop = ["ALL"]
    }

    # Hubble relay capabilities
    hubble_relay_capabilities = {
      drop = ["ALL"]
    }

    # Hubble UI capabilities
    hubble_ui_capabilities = {
      drop = ["ALL"]
    }

    # Operator capabilities
    operator_capabilities = {
      drop = ["ALL"]
    }
  }

  # --------------------------------------------------------------------------
  # ENVIRONMENT CONFIGURATION
  # --------------------------------------------------------------------------
  env_config = {
    dev = {
      vlan_services = 208
      vlan_internal = 111
      domain        = "dev.truxonline.com"
    }
    test = {
      vlan_services = 209
      vlan_internal = 111
      domain        = "test.truxonline.com"
    }
    staging = {
      vlan_services = 210
      vlan_internal = 111
      domain        = "staging.truxonline.com"
    }
    prod = {
      vlan_services = 201
      vlan_internal = 111
      domain        = "truxonline.com"
    }
  }

  # --------------------------------------------------------------------------
  # COMMON LABELS
  # --------------------------------------------------------------------------
  common_labels = {
    environment = var.environment
    managed_by  = "terraform"
    project     = "vixens"
  }

  # --------------------------------------------------------------------------
  # NETWORK DEFAULTS
  # --------------------------------------------------------------------------
  network = {
    pod_subnet     = "10.244.0.0/16"
    service_subnet = "10.96.0.0/12"
  }

  # --------------------------------------------------------------------------
  # SECURITY DEFAULTS
  # --------------------------------------------------------------------------
  security = {
    run_as_non_root            = true
    run_as_user                = 65532
    allow_privilege_escalation = false
    read_only_root_filesystem  = true
  }

  # --------------------------------------------------------------------------
  # TIMEOUTS
  # --------------------------------------------------------------------------
  timeouts = {
    helm_install = 1200 # 20 minutes (Cilium needs time to pull images and start on fresh cluster)
    helm_upgrade = 900  # 15 minutes
    k8s_api_wait = 300  # 5 minutes
    cilium_ready = 300  # 5 minutes
  }
}

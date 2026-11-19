# Centrifugo Real-time Chat Service Terraform Infrastructure Entry

resource "kubernetes_secret" "centrifugo_secrets" {
  metadata {
    name      = "centrifugo-secrets"
    namespace = var.namespace
    labels = {
      "app.kubernetes.io/name"    = "centrifugo"
      "app.kubernetes.io/part-of" = "techguild-chat"
    }
  }

  data = {
    api_key               = var.centrifugo_api_key
    token_hmac_secret_key = var.centrifugo_token_hmac_secret_key
  }

  type = "Opaque"
}

resource "kubernetes_config_map" "centrifugo_config" {
  metadata {
    name      = "centrifugo-config"
    namespace = var.namespace
    labels = {
      "app.kubernetes.io/name"    = "centrifugo"
      "app.kubernetes.io/part-of" = "techguild-chat"
    }
  }

  data = {
    "config.json" = jsonencode({
      token_hmac_secret_key      = "${var.centrifugo_token_hmac_secret_key}"
      api_key                    = "${var.centrifugo_api_key}"
      engine                     = "redis"
      redis_address              = var.redis_address
      allowed_origins            = ["https://techguild.io", "https://*.techguild.io", "http://localhost:3000"]
      allow_subscribe_for_client = false
      proxy_subscribe_endpoint   = var.backend_subscribe_proxy_url
      proxy_subscribe_timeout    = "5s"
      proxy_http_headers         = ["Authorization"]
      namespaces = [
        {
          name            = "quest"
          proxy_subscribe = true
          presence        = true
          join_leave      = true
          history_size    = 0
          history_ttl     = "0s"
        },
        {
          name            = "party"
          proxy_subscribe = true
          presence        = true
          join_leave      = true
          history_size    = 0
          history_ttl     = "0s"
        },
        {
          name            = "dm"
          proxy_subscribe = true
          presence        = true
          join_leave      = true
          history_size    = 0
          history_ttl     = "0s"
        }
      ]
    })
  }
}

resource "kubernetes_deployment" "centrifugo" {
  metadata {
    name      = "centrifugo"
    namespace = var.namespace
    labels = {
      "app.kubernetes.io/name"    = "centrifugo"
      "app.kubernetes.io/part-of" = "techguild-chat"
    }
  }

  spec {
    replicas = 3

    selector {
      match_labels = {
        "app.kubernetes.io/name" = "centrifugo"
      }
    }

    template {
      metadata {
        labels = {
          "app.kubernetes.io/name" = "centrifugo"
        }
      }

      spec {
        container {
          name              = "centrifugo"
          image             = "centrifugo/centrifugo:v5"
          image_pull_policy = "IfNotPresent"
          args              = ["centrifugo", "--config=/etc/centrifugo/config.json"]

          port {
            name           = "http"
            container_port = 8000
          }

          env {
            name  = "REDIS_ADDRESS"
            value = var.redis_address
          }

          env {
            name = "CENTRIFUGO_TOKEN_HMAC_SECRET_KEY"
            value_from {
              secret_key_ref {
                name = kubernetes_secret.centrifugo_secrets.metadata[0].name
                key  = "token_hmac_secret_key"
              }
            }
          }

          env {
            name = "CENTRIFUGO_API_KEY"
            value_from {
              secret_key_ref {
                name = kubernetes_secret.centrifugo_secrets.metadata[0].name
                key  = "api_key"
              }
            }
          }

          volume_mount {
            name       = "config-volume"
            mount_path = "/etc/centrifugo"
          }

          liveness_probe {
            http_get {
              path = "/health"
              port = 8000
            }
            initial_delay_seconds = 10
            period_seconds        = 15
          }

          readiness_probe {
            http_get {
              path = "/health"
              port = 8000
            }
            initial_delay_seconds = 5
            period_seconds        = 10
          }

          resources {
            requests = {
              cpu    = "100m"
              memory = "128Mi"
            }
            limits = {
              cpu    = "500m"
              memory = "512Mi"
            }
          }
        }

        volume {
          name = "config-volume"
          config_map {
            name = kubernetes_config_map.centrifugo_config.metadata[0].name
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "centrifugo" {
  metadata {
    name      = "centrifugo"
    namespace = var.namespace
    labels = {
      "app.kubernetes.io/name" = "centrifugo"
    }
  }

  spec {
    selector = {
      "app.kubernetes.io/name" = "centrifugo"
    }

    port {
      name        = "http"
      port        = 8000
      target_port = 8000
      protocol    = "TCP"
    }

    type = "ClusterIP"
  }
}

resource "kubernetes_ingress_v1" "centrifugo_ingress" {
  metadata {
    name      = "centrifugo-ingress"
    namespace = var.namespace
    annotations = {
      "kubernetes.io/ingress.class"                      = "nginx"
      "cert-manager.io/cluster-issuer"                   = "letsencrypt-prod"
      "nginx.ingress.kubernetes.io/websocket-services"   = "centrifugo"
      "nginx.ingress.kubernetes.io/proxy-read-timeout"    = "3600"
      "nginx.ingress.kubernetes.io/proxy-send-timeout"    = "3600"
      "nginx.ingress.kubernetes.io/proxy-connect-timeout" = "60"
      "nginx.ingress.kubernetes.io/proxy-http-version"    = "1.1"
      "nginx.ingress.kubernetes.io/configuration-snippet" = <<-EOT
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
      EOT
    }
  }

  spec {
    tls {
      hosts       = [var.domain]
      secret_name = "chat-techguild-tls"
    }

    rule {
      host = var.domain

      http {
        path {
          path      = "/connection/websocket"
          path_type = "ImplementationSpecific"
          backend {
            service {
              name = kubernetes_service.centrifugo.metadata[0].name
              port {
                number = 8000
              }
            }
          }
        }

        path {
          path      = "/"
          path_type = "Prefix"
          backend {
            service {
              name = kubernetes_service.centrifugo.metadata[0].name
              port {
                number = 8000
              }
            }
          }
        }
      }
    }
  }
}

output "centrifugo_websocket_url" {
  value       = "wss://${var.domain}/connection/websocket"
  description = "Public WebSocket endpoint for Centrifugo real-time connection"
}


terraform {
  required_version = ">= 1.5.0"
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.25.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12.0"
    }
  }
}

variable "namespace" {
  type        = string
  default     = "techguild"
  description = "Kubernetes namespace where Centrifugo and backend are deployed"
}

variable "domain" {
  type        = string
  default     = "chat.techguild.io"
  description = "Domain name for Centrifugo WebSocket service"
}

variable "redis_address" {
  type        = string
  default     = "redis.techguild.svc.cluster.local:6379"
  description = "Redis address for Centrifugo horizontal pub/sub engine"
}

variable "centrifugo_api_key" {
  type        = string
  sensitive   = true
  description = "API key used by the Go backend to call Centrifugo Server API"
}

variable "centrifugo_token_hmac_secret_key" {
  type        = string
  sensitive   = true
  description = "HMAC secret key used for signing and verifying connection JWTs"
}

variable "backend_subscribe_proxy_url" {
  type        = string
  default     = "http://techguild-backend.techguild.svc.cluster.local:8080/v1/communication/centrifugo/subscribe-proxy"
  description = "Backend endpoint called by Centrifugo to verify channel membership"
}


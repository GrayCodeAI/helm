# HELM Deployment Guide

## Overview

This guide covers deploying HELM in various environments.

## Docker Deployment

### Build Image

```bash
docker build -t helm:latest .
```

### Run with Docker

```bash
docker run -d \
  --name helm \
  -p 8080:8080 \
  -p 9090:9090 \
  -v helm_data:/data \
  -v helm_config:/config \
  -e HELM_LOG_LEVEL=info \
  -e HELM_METRICS_ENABLED=true \
  helm:latest
```

### Docker Compose

```bash
# Start with just HELM
docker-compose up -d

# Start with monitoring stack
docker-compose --profile with-metrics up -d

# Start with Redis caching
docker-compose --profile with-redis up -d
```

## Kubernetes Deployment

### Apply manifests

```bash
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/deployment.yaml
kubectl apply -f deploy/k8s/service.yaml
```

### Verify deployment

```bash
kubectl get pods -n helm-system
kubectl get svc -n helm-system
kubectl logs -n helm-system -l app=helm
```

### Access the dashboard

```bash
kubectl port-forward -n helm-system svc/helm 8080:8080
open http://localhost:8080
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HELM_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `HELM_LOG_FORMAT` | `text` | Log format (text, json) |
| `HELM_DATA_DIR` | `~/.local/share/helm` | Data directory |
| `HELM_CONFIG_DIR` | `~/.config/helm` | Config directory |
| `HELM_SERVER_PORT` | `8080` | Web server port |
| `HELM_METRICS_ENABLED` | `false` | Enable Prometheus metrics |
| `HELM_METRICS_PORT` | `9090` | Metrics port |

### Provider Configuration

Set provider API keys as environment variables:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-..."
export GOOGLE_API_KEY="..."
export OPENROUTER_API_KEY="..."
```

Or mount a config file:

```yaml
# In Kubernetes
volumeMounts:
  - name: config
    mountPath: /config
volumes:
  - name: config
    configMap:
      name: helm-config
```

## Monitoring

### Health Checks

```bash
# Overall health
curl http://localhost:8080/health

# Readiness probe
curl http://localhost:8080/health/ready

# Liveness probe
curl http://localhost:8080/health/live
```

### Prometheus Metrics

```bash
# Scrape metrics
curl http://localhost:9090/metrics
```

### Grafana Dashboards

1. Start with metrics profile:
   ```bash
   docker-compose --profile with-metrics up -d
   ```

2. Open Grafana at http://localhost:3000 (admin/admin)

3. Import dashboards from `deploy/grafana/`

## Backup and Restore

### Backup

```bash
# Using helm CLI
helm export full backup.tar.gz

# Manual database backup
cp ~/.local/share/helm/helm.db backup.db
```

### Restore

```bash
# Using helm CLI
helm import full backup.tar.gz

# Manual database restore
cp backup.db ~/.local/share/helm/helm.db
```

## Scaling

### Horizontal Scaling

HELM can be scaled horizontally with shared storage:

```yaml
# In Kubernetes, use a shared PVC
volumeClaimTemplates:
  - metadata:
      name: helm-data
    spec:
      accessModes: ["ReadWriteMany"]
      storageClassName: "nfs"
```

### Resource Limits

Recommended resources:
- CPU: 100m - 500m
- Memory: 128Mi - 512Mi
- Storage: 1Gi+

## Security

### Authentication

Enable authentication:

```toml
[server]
enable_auth = true

[auth]
type = "jwt"
jwt_secret = "your-secret-key"
```

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: helm-network-policy
  namespace: helm-system
spec:
  podSelector:
    matchLabels:
      app: helm
  policyTypes:
    - Ingress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: monitoring
      ports:
        - port: 9090
    - from: []
      ports:
        - port: 8080
```

## Troubleshooting

### Check logs

```bash
kubectl logs -n helm-system -l app=helm --tail=100
```

### Check health

```bash
kubectl exec -n helm-system deployment/helm -- helm status health
```

### Database issues

```bash
# Check database file
kubectl exec -n helm-system deployment/helm -- ls -la /data/

# Backup database
kubectl cp helm-system/$(kubectl get pod -n helm-system -l app=helm -o jsonpath='{.items[0].metadata.name}'):/data/helm.db ./backup.db
```

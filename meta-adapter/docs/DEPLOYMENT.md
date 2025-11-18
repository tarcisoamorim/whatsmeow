# Deployment Guide

**Version**: 1.0.0
**Last Updated**: 2025-11-18

---

## Table of Contents

1. [Overview](#overview)
2. [Local Development](#local-development)
3. [Docker Deployment](#docker-deployment)
4. [Kubernetes Deployment](#kubernetes-deployment)
5. [Cloud Deployments](#cloud-deployments)
6. [CI/CD Pipeline](#cicd-pipeline)
7. [Monitoring & Observability](#monitoring--observability)
8. [Backup & Disaster Recovery](#backup--disaster-recovery)
9. [Troubleshooting](#troubleshooting)

---

## 1. Overview

This guide covers deploying the WhatsApp Meta API Adapter to various environments:
- **Local**: Development with Docker Compose
- **Kubernetes**: Production-ready orchestration
- **AWS/GCP/Azure**: Cloud-specific guides
- **CI/CD**: Automated deployment pipelines

### Architecture Overview

```
┌─────────────────────────────────────────────────┐
│ Load Balancer (AWS ALB / NGINX Ingress)        │
└────────────────┬────────────────────────────────┘
                 │
        ┌────────┴────────┐
        │                 │
   ┌────▼────┐      ┌────▼────┐
   │ API     │      │ API     │
   │ Server  │      │ Server  │
   │ (Pod 1) │      │ (Pod 2) │
   └────┬────┘      └────┬────┘
        │                │
        └────────┬────────┘
                 │
    ┌────────────▼──────────────┐
    │                           │
┌───▼──┐  ┌──────┐  ┌────────┐│
│Postgre│  │Redis │  │RabbitMQ││
│SQL    │  │      │  │        ││
└───────┘  └──────┘  └────────┘│
                                │
           ┌────────────────────▼─┐
           │ Instance Manager     │
           │ (Stateful Set)       │
           └──────────────────────┘
```

---

## 2. Local Development

### Prerequisites

- Docker 24+
- Docker Compose 2.20+
- Go 1.24+ (for local dev without Docker)
- Node.js 20+ (for frontend)
- PostgreSQL 15+ (or use Docker)

### Quick Start (Docker Compose)

```bash
# Clone repository
git clone https://github.com/tarcisoamorim/whatsmeow.git
cd whatsmeow/meta-adapter

# Copy environment template
cp .env.example .env

# Edit .env with your configuration
vim .env

# Start all services
docker compose up -d

# Check logs
docker compose logs -f api-server

# API available at http://localhost:8080
# Admin dashboard at http://localhost:3000
```

### Docker Compose Configuration

```yaml
# docker-compose.yml
version: '3.9'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: whatsapp_adapter
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3

  rabbitmq:
    image: rabbitmq:3.12-management-alpine
    environment:
      RABBITMQ_DEFAULT_USER: ${RABBITMQ_USER}
      RABBITMQ_DEFAULT_PASS: ${RABBITMQ_PASSWORD}
    ports:
      - "5672:5672"   # AMQP
      - "15672:15672" # Management UI
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "ping"]
      interval: 30s
      timeout: 10s
      retries: 5

  api-server:
    build:
      context: .
      dockerfile: docker/Dockerfile
      target: production
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      rabbitmq:
        condition: service_healthy
    environment:
      DATABASE_URL: postgresql://postgres:${DB_PASSWORD}@postgres:5432/whatsapp_adapter?sslmode=disable
      REDIS_URL: redis://:${REDIS_PASSWORD}@redis:6379/0
      RABBITMQ_URL: amqp://${RABBITMQ_USER}:${RABBITMQ_PASSWORD}@rabbitmq:5672/
      JWT_SECRET: ${JWT_SECRET}
      PORT: 8080
    ports:
      - "8080:8080"
    volumes:
      - ./data/sessions:/app/data/sessions
      - ./data/media:/app/data/media
    restart: unless-stopped

  instance-manager:
    build:
      context: .
      dockerfile: docker/Dockerfile.instance-manager
    depends_on:
      - api-server
    environment:
      DATABASE_URL: postgresql://postgres:${DB_PASSWORD}@postgres:5432/whatsapp_adapter?sslmode=disable
      REDIS_URL: redis://:${REDIS_PASSWORD}@redis:6379/0
      MAX_INSTANCES: 100
    volumes:
      - ./data/sessions:/app/data/sessions
    restart: unless-stopped
    deploy:
      replicas: 2

  webhook-worker:
    build:
      context: .
      dockerfile: docker/Dockerfile
      target: worker
    depends_on:
      - rabbitmq
    environment:
      RABBITMQ_URL: amqp://${RABBITMQ_USER}:${RABBITMQ_PASSWORD}@rabbitmq:5672/
      DATABASE_URL: postgresql://postgres:${DB_PASSWORD}@postgres:5432/whatsapp_adapter?sslmode=disable
    restart: unless-stopped
    deploy:
      replicas: 3

  admin-dashboard:
    build:
      context: ./web
      dockerfile: Dockerfile
    ports:
      - "3000:80"
    environment:
      REACT_APP_API_URL: http://localhost:8080/v1
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
  rabbitmq_data:
```

### Running Migrations

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path migrations \
  -database "postgresql://postgres:password@localhost:5432/whatsapp_adapter?sslmode=disable" \
  up

# Rollback one migration
migrate -path migrations \
  -database "postgresql://postgres:password@localhost:5432/whatsapp_adapter?sslmode=disable" \
  down 1
```

---

## 3. Docker Deployment

### Multi-Stage Dockerfile

```dockerfile
# docker/Dockerfile

# Stage 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o /build/server \
    ./cmd/server/main.go

# Stage 2: Runtime
FROM alpine:3.19

# Install CA certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=app:app /build/server /app/server

# Create data directories
RUN mkdir -p /app/data/sessions /app/data/media && \
    chown -R app:app /app/data

# Switch to non-root user
USER app

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/server", "healthcheck"] || exit 1

EXPOSE 8080

ENTRYPOINT ["/app/server"]
```

### Build and Push

```bash
# Build image
docker build -t whatsapp-adapter:1.0.0 -f docker/Dockerfile .

# Tag for registry
docker tag whatsapp-adapter:1.0.0 gcr.io/project-id/whatsapp-adapter:1.0.0

# Push to registry
docker push gcr.io/project-id/whatsapp-adapter:1.0.0

# Or use Docker Hub
docker tag whatsapp-adapter:1.0.0 username/whatsapp-adapter:1.0.0
docker push username/whatsapp-adapter:1.0.0
```

---

## 4. Kubernetes Deployment

### Namespace

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: whatsapp-adapter
  labels:
    name: whatsapp-adapter
```

### ConfigMap

```yaml
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: whatsapp-adapter
data:
  PORT: "8080"
  LOG_LEVEL: "info"
  MAX_INSTANCES_PER_MANAGER: "100"
  RATE_LIMIT_REQUESTS_PER_MINUTE: "60"
```

### Secrets

```yaml
# k8s/secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: app-secrets
  namespace: whatsapp-adapter
type: Opaque
stringData:
  DATABASE_URL: postgresql://user:password@postgres:5432/whatsapp_adapter
  REDIS_URL: redis://:password@redis:6379/0
  RABBITMQ_URL: amqp://user:password@rabbitmq:5672/
  JWT_SECRET: your-256-bit-secret
  WEBHOOK_SECRET: your-webhook-secret
```

**Note**: Use sealed-secrets or external-secrets operator for production!

### PostgreSQL StatefulSet

```yaml
# k8s/postgres-statefulset.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: whatsapp-adapter
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        ports:
        - containerPort: 5432
          name: postgres
        env:
        - name: POSTGRES_DB
          value: whatsapp_adapter
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: username
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        - name: PGDATA
          value: /var/lib/postgresql/data/pgdata
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        livenessProbe:
          exec:
            command:
            - pg_isready
            - -U
            - postgres
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          exec:
            command:
            - pg_isready
            - -U
            - postgres
          initialDelaySeconds: 5
          periodSeconds: 5
  volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      storageClassName: standard
      resources:
        requests:
          storage: 50Gi
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: whatsapp-adapter
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
  clusterIP: None
```

### API Server Deployment

```yaml
# k8s/api-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-server
  namespace: whatsapp-adapter
  labels:
    app: api-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api-server
  template:
    metadata:
      labels:
        app: api-server
    spec:
      containers:
      - name: api-server
        image: gcr.io/project-id/whatsapp-adapter:1.0.0
        ports:
        - containerPort: 8080
          name: http
        envFrom:
        - configMapRef:
            name: app-config
        - secretRef:
            name: app-secrets
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        volumeMounts:
        - name: media-storage
          mountPath: /app/data/media
      volumes:
      - name: media-storage
        persistentVolumeClaim:
          claimName: media-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: api-server
  namespace: whatsapp-adapter
spec:
  selector:
    app: api-server
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
```

### Ingress (NGINX)

```yaml
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-ingress
  namespace: whatsapp-adapter
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - api.whatsapp-adapter.example.com
    secretName: api-tls
  rules:
  - host: api.whatsapp-adapter.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: api-server
            port:
              number: 80
```

### HorizontalPodAutoscaler

```yaml
# k8s/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-server-hpa
  namespace: whatsapp-adapter
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-server
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### Deploy to Kubernetes

```bash
# Create namespace
kubectl apply -f k8s/namespace.yaml

# Create secrets (use sealed-secrets in production!)
kubectl apply -f k8s/secrets.yaml

# Deploy database
kubectl apply -f k8s/postgres-statefulset.yaml

# Deploy Redis
kubectl apply -f k8s/redis-deployment.yaml

# Deploy RabbitMQ
kubectl apply -f k8s/rabbitmq-statefulset.yaml

# Deploy API server
kubectl apply -f k8s/api-deployment.yaml

# Deploy ingress
kubectl apply -f k8s/ingress.yaml

# Deploy HPA
kubectl apply -f k8s/hpa.yaml

# Check status
kubectl get pods -n whatsapp-adapter

# View logs
kubectl logs -f deployment/api-server -n whatsapp-adapter
```

---

## 5. Cloud Deployments

### 5.1 AWS (Elastic Kubernetes Service)

```bash
# Create EKS cluster
eksctl create cluster \
  --name whatsapp-adapter \
  --region us-east-1 \
  --nodegroup-name standard-workers \
  --node-type t3.medium \
  --nodes 3 \
  --nodes-min 3 \
  --nodes-max 10 \
  --managed

# Configure kubectl
aws eks update-kubeconfig --region us-east-1 --name whatsapp-adapter

# Install AWS Load Balancer Controller
kubectl apply -k "github.com/aws/eks-charts/stable/aws-load-balancer-controller//crds?ref=master"

helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName=whatsapp-adapter

# Deploy application
kubectl apply -f k8s/
```

**RDS for PostgreSQL**:
```bash
# Create RDS instance
aws rds create-db-instance \
  --db-instance-identifier whatsapp-adapter-db \
  --db-instance-class db.t3.medium \
  --engine postgres \
  --engine-version 15.4 \
  --master-username postgres \
  --master-user-password <password> \
  --allocated-storage 100 \
  --storage-encrypted \
  --backup-retention-period 7 \
  --multi-az \
  --publicly-accessible false
```

**ElastiCache for Redis**:
```bash
aws elasticache create-replication-group \
  --replication-group-id whatsapp-adapter-redis \
  --replication-group-description "Redis for WhatsApp Adapter" \
  --engine redis \
  --cache-node-type cache.t3.medium \
  --num-cache-clusters 2 \
  --automatic-failover-enabled
```

---

### 5.2 GCP (Google Kubernetes Engine)

```bash
# Create GKE cluster
gcloud container clusters create whatsapp-adapter \
  --region us-central1 \
  --num-nodes 3 \
  --machine-type n1-standard-2 \
  --enable-autoscaling \
  --min-nodes 3 \
  --max-nodes 10 \
  --enable-autorepair \
  --enable-autoupgrade

# Configure kubectl
gcloud container clusters get-credentials whatsapp-adapter --region us-central1

# Deploy application
kubectl apply -f k8s/
```

**Cloud SQL for PostgreSQL**:
```bash
gcloud sql instances create whatsapp-adapter-db \
  --database-version=POSTGRES_15 \
  --tier=db-custom-2-7680 \
  --region=us-central1 \
  --backup \
  --backup-start-time=03:00
```

**Memorystore for Redis**:
```bash
gcloud redis instances create whatsapp-adapter-redis \
  --size=5 \
  --region=us-central1 \
  --redis-version=redis_7_0
```

---

### 5.3 Azure (Azure Kubernetes Service)

```bash
# Create resource group
az group create --name whatsapp-adapter --location eastus

# Create AKS cluster
az aks create \
  --resource-group whatsapp-adapter \
  --name whatsapp-adapter-cluster \
  --node-count 3 \
  --node-vm-size Standard_D2s_v3 \
  --enable-addons monitoring \
  --generate-ssh-keys

# Configure kubectl
az aks get-credentials --resource-group whatsapp-adapter --name whatsapp-adapter-cluster

# Deploy application
kubectl apply -f k8s/
```

**Azure Database for PostgreSQL**:
```bash
az postgres flexible-server create \
  --resource-group whatsapp-adapter \
  --name whatsapp-adapter-db \
  --location eastus \
  --admin-user postgres \
  --admin-password <password> \
  --sku-name Standard_D2s_v3 \
  --tier GeneralPurpose \
  --version 15
```

---

## 6. CI/CD Pipeline

### GitHub Actions Workflow

```yaml
# .github/workflows/deploy.yml
name: Build and Deploy

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

env:
  REGISTRY: gcr.io
  IMAGE_NAME: project-id/whatsapp-adapter

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.24'

    - name: Run tests
      run: |
        go test ./... -v -cover -coverprofile=coverage.out
        go tool cover -func=coverage.out

    - name: Security scan
      run: |
        go install github.com/securego/gosec/v2/cmd/gosec@latest
        gosec ./...

    - name: Vulnerability check
      run: |
        go install golang.org/x/vuln/cmd/govulncheck@latest
        govulncheck ./...

  build:
    needs: test
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
    - uses: actions/checkout@v4

    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3

    - name: Login to GCR
      uses: docker/login-action@v3
      with:
        registry: ${{ env.REGISTRY }}
        username: _json_key
        password: ${{ secrets.GCR_JSON_KEY }}

    - name: Build and push
      uses: docker/build-push-action@v5
      with:
        context: .
        file: docker/Dockerfile
        push: true
        tags: |
          ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.sha }}
          ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:latest
        cache-from: type=registry,ref=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:buildcache
        cache-to: type=registry,ref=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:buildcache,mode=max

  deploy:
    needs: build
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Set up Cloud SDK
      uses: google-github-actions/setup-gcloud@v1
      with:
        service_account_key: ${{ secrets.GCP_SA_KEY }}

    - name: Get GKE credentials
      run: |
        gcloud container clusters get-credentials whatsapp-adapter \
          --region us-central1 \
          --project project-id

    - name: Deploy to Kubernetes
      run: |
        kubectl set image deployment/api-server \
          api-server=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.sha }} \
          -n whatsapp-adapter

        kubectl rollout status deployment/api-server -n whatsapp-adapter

    - name: Notify Slack
      if: always()
      uses: 8398a7/action-slack@v3
      with:
        status: ${{ job.status }}
        text: Deployment to production ${{ job.status }}
      env:
        SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK }}
```

---

## 7. Monitoring & Observability

### Prometheus Operator

```bash
# Install Prometheus Operator
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace

# ServiceMonitor for API
kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: api-server-metrics
  namespace: whatsapp-adapter
spec:
  selector:
    matchLabels:
      app: api-server
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
EOF
```

### Grafana Dashboards

```bash
# Access Grafana
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80

# Default credentials: admin / prom-operator

# Import dashboard: https://grafana.com/grafana/dashboards/
# - 15661: Kubernetes cluster monitoring
# - 13770: Go application metrics
```

### Loki for Logs

```bash
# Install Loki
helm install loki grafana/loki-stack \
  --namespace monitoring \
  --set promtail.enabled=true \
  --set grafana.enabled=false

# Configure Grafana data source
# URL: http://loki:3100
```

---

## 8. Backup & Disaster Recovery

### Database Backup (Automated)

```yaml
# k8s/backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
  namespace: whatsapp-adapter
spec:
  schedule: "0 3 * * *"  # Daily at 3 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:15-alpine
            command:
            - /bin/sh
            - -c
            - |
              pg_dump -h postgres -U postgres whatsapp_adapter | \
              gzip > /backup/backup-$(date +%Y%m%d-%H%M%S).sql.gz

              # Upload to S3
              aws s3 cp /backup/backup-$(date +%Y%m%d-%H%M%S).sql.gz \
                s3://backups-bucket/postgres/

              # Cleanup old backups (keep 30 days)
              find /backup -name "backup-*.sql.gz" -mtime +30 -delete
            env:
            - name: PGPASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: password
            volumeMounts:
            - name: backup-storage
              mountPath: /backup
          volumes:
          - name: backup-storage
            persistentVolumeClaim:
              claimName: backup-pvc
          restartPolicy: OnFailure
```

### Disaster Recovery Plan

**RTO (Recovery Time Objective)**: 4 hours
**RPO (Recovery Point Objective)**: 24 hours (daily backups)

**Recovery Steps**:
```bash
# 1. Restore database from latest backup
aws s3 cp s3://backups-bucket/postgres/backup-20251118-030000.sql.gz .
gunzip backup-20251118-030000.sql.gz

psql -h postgres -U postgres whatsapp_adapter < backup-20251118-030000.sql

# 2. Verify data integrity
psql -h postgres -U postgres -d whatsapp_adapter -c "SELECT COUNT(*) FROM tenants;"

# 3. Restart services
kubectl rollout restart deployment/api-server -n whatsapp-adapter

# 4. Verify health
kubectl get pods -n whatsapp-adapter
curl https://api.whatsapp-adapter.example.com/health
```

---

## 9. Troubleshooting

### Common Issues

**Issue**: Pods in CrashLoopBackOff

```bash
# Check pod logs
kubectl logs -f pod/api-server-xxx -n whatsapp-adapter

# Describe pod
kubectl describe pod api-server-xxx -n whatsapp-adapter

# Common causes:
# - Missing environment variables
# - Database connection failure
# - Insufficient resources
```

**Issue**: High memory usage

```bash
# Check resource usage
kubectl top pods -n whatsapp-adapter

# Increase memory limits
kubectl set resources deployment api-server \
  --limits=memory=2Gi \
  -n whatsapp-adapter
```

**Issue**: Database connection timeout

```bash
# Check PostgreSQL logs
kubectl logs -f statefulset/postgres -n whatsapp-adapter

# Verify connectivity
kubectl run -it --rm debug --image=postgres:15-alpine --restart=Never -- \
  psql -h postgres.whatsapp-adapter.svc.cluster.local -U postgres

# Check connection pool settings
# Increase max_connections in PostgreSQL
```

---

## Appendix: Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | API server port | 8080 | No |
| `DATABASE_URL` | PostgreSQL connection string | - | Yes |
| `REDIS_URL` | Redis connection string | - | Yes |
| `RABBITMQ_URL` | RabbitMQ connection string | - | Yes |
| `JWT_SECRET` | JWT signing secret | - | Yes |
| `LOG_LEVEL` | Logging level | info | No |
| `MAX_INSTANCES` | Max instances per manager | 100 | No |
| `RATE_LIMIT_RPM` | Requests per minute | 60 | No |
| `WEBHOOK_TIMEOUT` | Webhook timeout (seconds) | 30 | No |
| `S3_BUCKET` | S3 bucket for media | - | No |
| `AWS_REGION` | AWS region | us-east-1 | No |

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-11-18 | Initial deployment guide |

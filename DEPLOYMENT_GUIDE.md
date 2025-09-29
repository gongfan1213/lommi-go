# Go版本部署指南

## 🚀 部署方式概览

Go版本支持多种部署方式，包括Docker、Kubernetes等，满足不同环境和需求。

## 📦 Docker部署

### 1. 构建镜像

```bash
# 构建生产环境镜像
docker build -t loomi-go:latest -f Dockerfile .

# 构建开发环境镜像
docker build -t loomi-go:dev -f Dockerfile.dev .
```

### 2. 运行容器

#### 生产环境
```bash
# 使用docker-compose
docker-compose up -d

# 或者单独运行
docker run -d \
  --name loomi-go \
  -p 8080:8080 \
  -p 8081:8081 \
  -e ENV=production \
  -e LOG_LEVEL=info \
  -v $(pwd)/logs:/app/logs \
  -v $(pwd)/uploads:/app/uploads \
  loomi-go:latest
```

#### 开发环境
```bash
# 使用docker-compose
docker-compose -f docker-compose.dev.yml up -d

# 或者单独运行
docker run -d \
  --name loomi-go-dev \
  -p 8080:8080 \
  -p 8081:8081 \
  -e ENV=development \
  -e LOG_LEVEL=debug \
  -v $(pwd):/app \
  loomi-go:dev
```

### 3. 验证部署

```bash
# 检查容器状态
docker ps

# 查看日志
docker logs loomi-go

# 健康检查
curl http://localhost:8080/health
```

## ☸️ Kubernetes部署

### 1. 环境准备

```bash
# 确保kubectl已配置
kubectl cluster-info

# 创建命名空间
kubectl apply -f k8s/namespace.yaml
```

### 2. 配置管理

```bash
# 部署配置和密钥
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml
```

### 3. 部署服务

```bash
# 部署Redis
kubectl apply -f k8s/redis.yaml

# 部署应用
kubectl apply -f k8s/app.yaml

# 部署Ingress
kubectl apply -f k8s/ingress.yaml

# 部署监控
kubectl apply -f k8s/monitoring.yaml

# 部署HPA
kubectl apply -f k8s/hpa.yaml
```

### 4. 使用Kustomize部署

```bash
# 生产环境
kubectl apply -k k8s/

# 开发环境
kubectl apply -k k8s/overlays/dev/
```

### 5. 验证部署

```bash
# 检查Pod状态
kubectl get pods -n loomi

# 检查Service状态
kubectl get services -n loomi

# 检查Ingress状态
kubectl get ingress -n loomi

# 检查HPA状态
kubectl get hpa -n loomi

# 查看应用日志
kubectl logs -f deployment/loomi-go -n loomi
```

## 🛠️ 自动化部署脚本

### 使用部署脚本

```bash
# 构建镜像
./deploy.sh build v1.0.0

# 推送到镜像仓库
./deploy.sh push myregistry.com v1.0.0

# 部署到生产环境
./deploy.sh deploy production

# 部署到开发环境
./deploy.sh deploy development

# 验证部署
./deploy.sh verify loomi

# 查看日志
./deploy.sh logs loomi

# 扩展应用
./deploy.sh scale loomi 5

# 回滚部署
./deploy.sh rollback loomi
```

## 🔧 环境配置

### 生产环境配置

```yaml
# .env.prod
ENV=production
LOG_LEVEL=info
DATABASE_TYPE=supabase
REDIS_ENABLED=true
ALERT_ENABLED=true

# API配置
OPENAI_API_KEY=your-openai-key
CLAUDE_API_KEY=your-claude-key
GEMINI_API_KEY=your-gemini-key

# 数据库配置
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_KEY=your-supabase-key

# Redis配置
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=your-redis-password

# 告警配置
FEISHU_WEBHOOK_URL=https://open.feishu.cn/openapi/webhook
SMTP_PASSWORD=your-smtp-password
```

### 开发环境配置

```yaml
# .env.dev
ENV=development
LOG_LEVEL=debug
DATABASE_TYPE=inmemory
REDIS_ENABLED=false
ALERT_ENABLED=false

# 使用测试密钥
OPENAI_API_KEY=test-key
CLAUDE_API_KEY=test-key
GEMINI_API_KEY=test-key
```

## 📊 监控配置

### Prometheus监控

```yaml
# monitoring/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'loomi-go'
    static_configs:
      - targets: ['loomi-go:8081']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Grafana仪表板

访问 `http://localhost:3000` 查看Grafana仪表板
- 用户名: admin
- 密码: admin123

## 🔐 安全配置

### SSL证书配置

```bash
# 生成自签名证书（仅用于测试）
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout nginx/ssl/key.pem \
  -out nginx/ssl/cert.pem

# 生产环境使用Let's Encrypt
certbot certonly --nginx -d api.loomi.com
```

### 密钥管理

```bash
# 创建密钥
kubectl create secret generic loomi-secrets \
  --from-literal=jwt-secret=your-jwt-secret \
  --from-literal=master-key=your-master-key \
  --from-literal=openai-api-key=your-openai-key

# 更新密钥
kubectl patch secret loomi-secrets -p='{"data":{"jwt-secret":"new-secret"}}'
```

## 🚀 性能优化

### 资源配置

```yaml
# Kubernetes资源配置
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "1Gi"
    cpu: "500m"
```

### 水平扩展

```bash
# 手动扩展
kubectl scale deployment loomi-go --replicas=5 -n loomi

# 自动扩展（HPA）
kubectl apply -f k8s/hpa.yaml
```

### 缓存配置

```yaml
# Redis配置
redis:
  max_connections: 50
  connection_timeout: 10
  max_memory: 256mb
  max_memory_policy: allkeys-lru
```

## 🔄 更新部署

### 滚动更新

```bash
# 更新镜像
kubectl set image deployment/loomi-go loomi-go=loomi-go:v1.1.0 -n loomi

# 查看更新状态
kubectl rollout status deployment/loomi-go -n loomi

# 回滚更新
kubectl rollout undo deployment/loomi-go -n loomi
```

### 蓝绿部署

```bash
# 部署新版本
kubectl apply -f k8s/app-v2.yaml

# 切换流量
kubectl patch service loomi-go -p '{"spec":{"selector":{"version":"v1.1.0"}}}'
```

## 🧪 测试部署

### 运行测试

```bash
# 运行单元测试
./run_tests.sh -u

# 运行集成测试
./run_tests.sh -i

# 运行性能测试
./run_tests.sh -p

# 运行所有测试
./run_tests.sh
```

### 健康检查

```bash
# 应用健康检查
curl http://localhost:8080/health

# 就绪检查
curl http://localhost:8080/ready

# 指标检查
curl http://localhost:8081/metrics
```

## 📝 故障排除

### 常见问题

1. **Pod启动失败**
   ```bash
   kubectl describe pod <pod-name> -n loomi
   kubectl logs <pod-name> -n loomi
   ```

2. **服务无法访问**
   ```bash
   kubectl get services -n loomi
   kubectl get endpoints -n loomi
   ```

3. **Ingress配置问题**
   ```bash
   kubectl describe ingress loomi-ingress -n loomi
   ```

4. **资源不足**
   ```bash
   kubectl top nodes
   kubectl top pods -n loomi
   ```

### 日志查看

```bash
# 查看应用日志
kubectl logs -f deployment/loomi-go -n loomi

# 查看系统日志
kubectl logs -f deployment/redis -n loomi

# 查看监控日志
kubectl logs -f deployment/prometheus -n loomi
```

## 🔧 维护操作

### 备份

```bash
# 备份配置
kubectl get configmap loomi-config -o yaml > backup/configmap.yaml
kubectl get secret loomi-secrets -o yaml > backup/secret.yaml

# 备份数据
kubectl exec -it redis-0 -n loomi -- redis-cli BGSAVE
```

### 清理

```bash
# 清理测试环境
./deploy.sh cleanup loomi-dev

# 清理镜像
docker system prune -a
```

## 📚 相关文档

- [Docker部署文档](https://docs.docker.com/)
- [Kubernetes部署文档](https://kubernetes.io/docs/)
- [Prometheus监控文档](https://prometheus.io/docs/)
- [Grafana仪表板文档](https://grafana.com/docs/)
- [Nginx配置文档](https://nginx.org/en/docs/)

## 🆘 支持

如果遇到问题，请：

1. 查看日志文件
2. 检查配置文件
3. 验证网络连接
4. 查看资源使用情况
5. 联系技术支持

---

**部署完成后，您的Go版本Loomi应用就可以在生产环境中运行了！** 🎉

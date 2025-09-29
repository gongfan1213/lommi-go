# Loomi Go - 智能内容创作平台

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-blue.svg)](Dockerfile)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-blue.svg)](k8s/)

Loomi Go是一个基于Go语言开发的企业级智能内容创作平台，提供完整的AI Agent系统、多模态处理、内容生成和智能分析功能。项目采用云原生架构，支持Docker容器化部署和Kubernetes编排。

## 🚀 核心特性

### 🤖 智能Agent系统
- **多Agent协作架构** - 支持Supervisor + Researcher Agents模式
- **Nova3智能编排** - 智能任务调度和决策引擎
- **Loomi内容专家** - 专业内容创作和优化
- **实时状态管理** - 完整的Agent生命周期管理

### 🎯 内容创作功能
- **受众画像分析** - 深度用户画像和偏好分析
- **TikTok口播稿创作** - 专业的短视频脚本生成
- **多模态内容处理** - 支持文本、图像、音频等多种格式
- **智能内容优化** - 基于AI的内容质量提升

### 🔧 技术架构
- **微服务架构** - 模块化设计，易于扩展和维护
- **云原生部署** - 支持Docker和Kubernetes部署
- **高可用设计** - 多副本部署和自动故障恢复
- **可观测性** - 完整的监控、日志和告警体系

## 🛠️ 技术栈

### 后端技术
- **Go 1.21+** - 主要编程语言
- **Gin Gonic** - Web框架
- **GORM** - ORM数据库操作
- **Redis** - 缓存和会话管理
- **PostgreSQL/Supabase** - 主数据库

### AI/ML技术
- **OpenAI GPT-4** - 文本生成和分析
- **Claude 3** - 智能对话和内容创作
- **Gemini Pro** - 多模态内容处理
- **百度AI** - 图像识别和OCR
- **智谱AI** - 中文内容生成

### 基础设施
- **Docker** - 容器化部署
- **Kubernetes** - 容器编排
- **Prometheus** - 监控系统
- **Grafana** - 可视化仪表板
- **Loki** - 日志聚合
- **Nginx** - 反向代理和负载均衡

### 存储和缓存
- **Redis** - 内存缓存和会话存储
- **Supabase** - 云数据库和实时订阅
- **阿里云OSS** - 对象存储服务
- **本地文件系统** - 临时文件存储

## 📁 项目结构

```
loomi-go/
├── cmd/loomi/                 # 主程序入口
├── internal/loomi/            # 核心业务逻辑
│   ├── agents/               # AI Agent实现
│   │   ├── persona_agent.go  # 受众画像Agent
│   │   └── tiktok_script_agent.go # TikTok脚本Agent
│   ├── api/                  # API路由和处理器
│   ├── config/               # 配置管理
│   ├── database/             # 数据库接口和实现
│   ├── llm/                  # LLM客户端接口
│   ├── monitoring/           # 监控系统
│   ├── pool/                 # 连接池管理
│   ├── tools/                # 工具类集合
│   │   ├── multimodal/       # 多模态处理工具
│   │   └── search/           # 搜索工具
│   └── utils/                # 通用工具类
├── k8s/                      # Kubernetes部署配置
├── monitoring/               # 监控配置文件
├── nginx/                    # Nginx配置
├── docker-compose.yml        # Docker编排配置
├── Dockerfile                # Docker镜像构建
└── deploy.sh                 # 自动化部署脚本
```

## 🏗️ 系统架构

### 整体架构图
```
┌─────────────────────────────────────────────────────────────┐
│                    Client Layer                             │
│  Web UI │ Mobile App │ API Client │ Third-party Integration │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway Layer                        │
│  Nginx │ Load Balancer │ SSL Termination │ Rate Limiting    │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│                  Application Layer                          │
│  Loomi Go Service │ Authentication │ Authorization │ Logging│
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│                   Business Logic Layer                      │
│  Agent System │ Content Generation │ Multi-modal Processing │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│                    Data Layer                               │
│  Supabase │ Redis │ OSS │ File System │ External APIs      │
└─────────────────────────────────────────────────────────────┘
```

### Agent系统架构
```
┌─────────────────────────────────────────────────────────────┐
│                    Supervisor Agent                         │
│  Task Planning │ Resource Allocation │ Quality Control     │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│                   Researcher Agents                         │
│  Insight Agent │ Profile Agent │ Hitpoint Agent │ Writing   │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│                    Loomi Agents                             │
│  Persona Agent │ TikTok Script Agent │ Content Optimization│
└─────────────────────────────────────────────────────────────┘
```

## 🚀 快速开始

### 环境要求
- Go 1.21+
- Docker & Docker Compose
- Redis 7.0+
- Node.js 18+ (前端开发)

### 本地开发

1. **克隆项目**
```bash
git clone <repository-url>
cd loomi-go
```

2. **安装依赖**
```bash
go mod tidy
```

3. **配置环境变量**
```bash
cp .env.example .env.dev
# 编辑 .env.dev 文件，配置必要的环境变量
```

4. **启动开发环境**
```bash
# 使用Docker Compose启动开发环境
docker-compose -f docker-compose.dev.yml up -d

# 或者直接运行Go程序
go run ./cmd/loomi
```

5. **验证部署**
```bash
curl http://localhost:8080/health
```

### 生产环境部署

#### Docker部署
```bash
# 构建镜像
docker build -t loomi-go:latest .

# 启动服务
docker-compose up -d
```

#### Kubernetes部署
```bash
# 部署到K8s集群
./deploy.sh deploy production

# 验证部署
./deploy.sh verify loomi
```

## ⚙️ 配置说明

### 环境变量配置

#### 必需配置
```bash
# 应用配置
ENV=production
LOG_LEVEL=info
API_HOST=0.0.0.0
API_PORT=8080

# LLM配置
OPENAI_API_KEY=your-openai-key
CLAUDE_API_KEY=your-claude-key
GEMINI_API_KEY=your-gemini-key

# 数据库配置
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_KEY=your-supabase-key

# Redis配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your-redis-password
```

#### 可选配置
```bash
# 告警配置
FEISHU_WEBHOOK_URL=https://open.feishu.cn/openapi/webhook
SMTP_PASSWORD=your-smtp-password

# OSS配置
OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
OSS_BUCKET_NAME=your-bucket
OSS_ACCESS_KEY_ID=your-access-key-id
OSS_ACCESS_KEY_SECRET=your-access-key-secret

# 百度AI配置
BAIDU_API_KEY=your-baidu-api-key
BAIDU_SECRET_KEY=your-baidu-secret-key
```

### 配置文件

项目支持YAML配置文件，优先级：环境变量 > 配置文件 > 默认值

```yaml
# config/app_config.yaml
app:
  name: "loomi-go"
  version: "1.0.0"
  environment: "production"

api:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30
  write_timeout: 30

security:
  jwt_secret: "your-jwt-secret"
  cors_origins: ["*"]
  rate_limit_enabled: true
```

## 📊 监控和运维

### 监控系统
- **Prometheus** - 指标收集和存储
- **Grafana** - 可视化仪表板
- **Loki** - 日志聚合
- **AlertManager** - 告警管理

### 健康检查
```bash
# 应用健康检查
curl http://localhost:8080/health

# 就绪检查
curl http://localhost:8080/ready

# 指标接口
curl http://localhost:8081/metrics
```

### 日志查看
```bash
# Docker环境
docker logs -f loomi-go

# Kubernetes环境
kubectl logs -f deployment/loomi-go -n loomi
```

## 🧪 测试

### 运行测试
```bash
# 运行所有测试
./run_tests.sh

# 运行特定测试
./run_tests.sh -u  # 单元测试
./run_tests.sh -i  # 集成测试
./run_tests.sh -p  # 性能测试
./run_tests.sh -s  # 压力测试
./run_tests.sh -e  # 端到端测试
```

### 测试覆盖率
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📈 性能优化

### 并发优化
- 使用Go的goroutine处理并发任务
- 连接池管理数据库和Redis连接
- 异步处理耗时操作

### 缓存策略
- Redis缓存热点数据
- 内存缓存频繁访问的数据
- 合理的缓存过期策略

### 数据库优化
- 索引优化
- 查询优化
- 连接池配置

## 🔒 安全特性

### 认证授权
- JWT Token认证
- 基于角色的访问控制
- API密钥管理

### 数据安全
- 敏感数据加密存储
- 传输层SSL/TLS加密
- 安全头配置

### 网络安全
- 限流和防DDoS
- CORS配置
- 防火墙规则

## 🤝 贡献指南

### 开发流程
1. Fork项目
2. 创建特性分支
3. 提交更改
4. 创建Pull Request

### 代码规范
- 遵循Go官方代码规范
- 使用gofmt格式化代码
- 编写单元测试
- 添加必要的注释

### 提交规范
```
feat: 新功能
fix: 修复bug
docs: 文档更新
style: 代码格式调整
refactor: 代码重构
test: 测试相关
chore: 构建过程或辅助工具的变动
```

## 📄 许可证

本项目采用MIT许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🆘 支持和反馈

### 问题报告
- 使用GitHub Issues报告bug
- 提供详细的错误信息和复现步骤
- 包含系统环境信息

### 功能请求
- 使用GitHub Issues提出功能建议
- 描述使用场景和预期效果
- 参与讨论和投票

### 社区支持
- 查看[文档](docs/)获取详细说明
- 参与[讨论](discussions/)交流经验
- 关注项目更新和公告

## 🔄 更新日志

### v1.0.0 (2024-01-XX)
- 🎉 初始版本发布
- ✨ 完整的Agent系统
- ✨ 多模态内容处理
- ✨ 云原生部署支持
- ✨ 完整的监控体系

## 📚 相关文档

- [部署指南](DEPLOYMENT_GUIDE.md)
- [API文档](docs/API.md)
- [架构设计](docs/ARCHITECTURE.md)
- [开发指南](docs/DEVELOPMENT.md)
- [故障排除](docs/TROUBLESHOOTING.md)

---

**Loomi Go - 让AI内容创作更简单、更智能、更高效！** 🚀

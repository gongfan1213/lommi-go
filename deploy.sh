#!/bin/bash

# 部署脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# 打印标题
print_title() {
    echo
    print_message $BLUE "=================================="
    print_message $BLUE "$1"
    print_message $BLUE "=================================="
    echo
}

# 检查依赖
check_dependencies() {
    print_title "检查部署依赖"
    
    # 检查Docker
    if ! command -v docker &> /dev/null; then
        print_message $RED "错误: 未找到Docker命令，请确保Docker已安装"
        exit 1
    fi
    
    # 检查kubectl
    if ! command -v kubectl &> /dev/null; then
        print_message $RED "错误: 未找到kubectl命令，请确保kubectl已安装"
        exit 1
    fi
    
    # 检查helm（可选）
    if ! command -v helm &> /dev/null; then
        print_message $YELLOW "警告: 未找到helm命令，跳过helm相关操作"
    fi
    
    print_message $GREEN "依赖检查通过"
}

# 构建Docker镜像
build_docker_image() {
    print_title "构建Docker镜像"
    
    local image_tag=${1:-latest}
    local image_name="loomi-go"
    
    print_message $YELLOW "构建生产环境镜像..."
    docker build -t ${image_name}:${image_tag} -f Dockerfile .
    
    print_message $YELLOW "构建开发环境镜像..."
    docker build -t ${image_name}:dev -f Dockerfile.dev .
    
    print_message $GREEN "Docker镜像构建完成"
}

# 推送Docker镜像
push_docker_image() {
    print_title "推送Docker镜像"
    
    local registry=${1:-"your-registry.com"}
    local image_tag=${2:-latest}
    local image_name="loomi-go"
    
    print_message $YELLOW "标记镜像..."
    docker tag ${image_name}:${image_tag} ${registry}/${image_name}:${image_tag}
    docker tag ${image_name}:dev ${registry}/${image_name}:dev
    
    print_message $YELLOW "推送镜像到仓库..."
    docker push ${registry}/${image_name}:${image_tag}
    docker push ${registry}/${image_name}:dev
    
    print_message $GREEN "Docker镜像推送完成"
}

# 部署到Kubernetes
deploy_to_k8s() {
    print_title "部署到Kubernetes"
    
    local environment=${1:-production}
    local namespace="loomi"
    
    if [ "$environment" = "development" ]; then
        namespace="loomi-dev"
    fi
    
    print_message $YELLOW "部署到环境: $environment"
    print_message $YELLOW "命名空间: $namespace"
    
    # 创建命名空间
    kubectl apply -f k8s/namespace.yaml
    
    # 部署配置
    kubectl apply -f k8s/configmap.yaml
    kubectl apply -f k8s/secret.yaml
    
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
    
    print_message $GREEN "Kubernetes部署完成"
}

# 使用Kustomize部署
deploy_with_kustomize() {
    print_title "使用Kustomize部署"
    
    local environment=${1:-production}
    
    if [ "$environment" = "development" ]; then
        kubectl apply -k k8s/overlays/dev/
    else
        kubectl apply -k k8s/
    fi
    
    print_message $GREEN "Kustomize部署完成"
}

# 验证部署
verify_deployment() {
    print_title "验证部署"
    
    local namespace=${1:-loomi}
    
    print_message $YELLOW "检查Pod状态..."
    kubectl get pods -n $namespace
    
    print_message $YELLOW "检查Service状态..."
    kubectl get services -n $namespace
    
    print_message $YELLOW "检查Ingress状态..."
    kubectl get ingress -n $namespace
    
    print_message $YELLOW "检查HPA状态..."
    kubectl get hpa -n $namespace
    
    # 等待Pod就绪
    print_message $YELLOW "等待Pod就绪..."
    kubectl wait --for=condition=ready pod -l app=loomi-go -n $namespace --timeout=300s
    
    print_message $GREEN "部署验证完成"
}

# 回滚部署
rollback_deployment() {
    print_title "回滚部署"
    
    local namespace=${1:-loomi}
    
    print_message $YELLOW "回滚到上一个版本..."
    kubectl rollout undo deployment/loomi-go -n $namespace
    
    print_message $YELLOW "等待回滚完成..."
    kubectl rollout status deployment/loomi-go -n $namespace
    
    print_message $GREEN "回滚完成"
}

# 清理部署
cleanup_deployment() {
    print_title "清理部署"
    
    local namespace=${1:-loomi}
    
    print_message $YELLOW "删除所有资源..."
    kubectl delete -f k8s/ --ignore-not-found=true
    
    print_message $YELLOW "删除命名空间..."
    kubectl delete namespace $namespace --ignore-not-found=true
    
    print_message $GREEN "清理完成"
}

# 查看日志
view_logs() {
    print_title "查看应用日志"
    
    local namespace=${1:-loomi}
    local pod_name=${2:-""}
    
    if [ -z "$pod_name" ]; then
        pod_name=$(kubectl get pods -n $namespace -l app=loomi-go -o jsonpath='{.items[0].metadata.name}')
    fi
    
    print_message $YELLOW "查看Pod日志: $pod_name"
    kubectl logs -f $pod_name -n $namespace
}

# 进入Pod
exec_into_pod() {
    print_title "进入Pod"
    
    local namespace=${1:-loomi}
    local pod_name=${2:-""}
    
    if [ -z "$pod_name" ]; then
        pod_name=$(kubectl get pods -n $namespace -l app=loomi-go -o jsonpath='{.items[0].metadata.name}')
    fi
    
    print_message $YELLOW "进入Pod: $pod_name"
    kubectl exec -it $pod_name -n $namespace -- /bin/sh
}

# 扩展应用
scale_app() {
    print_title "扩展应用"
    
    local namespace=${1:-loomi}
    local replicas=${2:-3}
    
    print_message $YELLOW "扩展应用到 $replicas 个副本..."
    kubectl scale deployment loomi-go --replicas=$replicas -n $namespace
    
    print_message $YELLOW "等待扩展完成..."
    kubectl rollout status deployment/loomi-go -n $namespace
    
    print_message $GREEN "扩展完成"
}

# 显示帮助信息
show_help() {
    echo "用法: $0 [命令] [选项]"
    echo
    echo "命令:"
    echo "  build [tag]                    构建Docker镜像"
    echo "  push [registry] [tag]          推送Docker镜像"
    echo "  deploy [environment]           部署到Kubernetes"
    echo "  deploy-kustomize [environment] 使用Kustomize部署"
    echo "  verify [namespace]             验证部署"
    echo "  rollback [namespace]           回滚部署"
    echo "  cleanup [namespace]            清理部署"
    echo "  logs [namespace] [pod]         查看日志"
    echo "  exec [namespace] [pod]         进入Pod"
    echo "  scale [namespace] [replicas]   扩展应用"
    echo "  -h, --help                     显示帮助信息"
    echo
    echo "环境:"
    echo "  production (默认)              生产环境"
    echo "  development                    开发环境"
    echo
    echo "示例:"
    echo "  $0 build v1.0.0                # 构建v1.0.0版本镜像"
    echo "  $0 push myregistry.com v1.0.0  # 推送到镜像仓库"
    echo "  $0 deploy production           # 部署到生产环境"
    echo "  $0 deploy development          # 部署到开发环境"
    echo "  $0 verify loomi                # 验证生产环境部署"
    echo "  $0 logs loomi                  # 查看生产环境日志"
    echo "  $0 scale loomi 5               # 扩展到5个副本"
}

# 主函数
main() {
    case "${1:-}" in
        build)
            check_dependencies
            build_docker_image "$2"
            ;;
        push)
            check_dependencies
            push_docker_image "$2" "$3"
            ;;
        deploy)
            check_dependencies
            deploy_to_k8s "$2"
            verify_deployment "$2"
            ;;
        deploy-kustomize)
            check_dependencies
            deploy_with_kustomize "$2"
            verify_deployment "$2"
            ;;
        verify)
            verify_deployment "$2"
            ;;
        rollback)
            rollback_deployment "$2"
            ;;
        cleanup)
            cleanup_deployment "$2"
            ;;
        logs)
            view_logs "$2" "$3"
            ;;
        exec)
            exec_into_pod "$2" "$3"
            ;;
        scale)
            scale_app "$2" "$3"
            ;;
        -h|--help)
            show_help
            ;;
        "")
            show_help
            ;;
        *)
            print_message $RED "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@"

#!/bin/bash

# 测试运行脚本

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

# 检查Go环境
check_go_environment() {
    print_title "检查Go环境"
    
    if ! command -v go &> /dev/null; then
        print_message $RED "错误: 未找到Go命令，请确保Go已安装"
        exit 1
    fi
    
    go_version=$(go version)
    print_message $GREEN "Go版本: $go_version"
    
    # 检查是否在项目根目录
    if [ ! -f "go.mod" ]; then
        print_message $RED "错误: 请在项目根目录运行此脚本"
        exit 1
    fi
    
    print_message $GREEN "Go环境检查通过"
}

# 安装测试依赖
install_test_dependencies() {
    print_title "安装测试依赖"
    
    # 安装测试工具
    print_message $YELLOW "安装测试工具..."
    go install github.com/stretchr/testify/assert@latest
    go install github.com/golang/mock/mockgen@latest
    go install github.com/axw/gocov/gocov@latest
    go install github.com/AlekSi/gocov-xml@latest
    go install github.com/tebeka/go2xunit@latest
    
    # 下载依赖
    print_message $YELLOW "下载项目依赖..."
    go mod tidy
    go mod download
    
    print_message $GREEN "依赖安装完成"
}

# 运行单元测试
run_unit_tests() {
    print_title "运行单元测试"
    
    print_message $YELLOW "运行Go标准测试..."
    go test -v -race -coverprofile=coverage.out ./...
    
    if [ $? -eq 0 ]; then
        print_message $GREEN "单元测试通过"
    else
        print_message $RED "单元测试失败"
        exit 1
    fi
}

# 运行集成测试
run_integration_tests() {
    print_title "运行集成测试"
    
    # 设置测试环境变量
    export TEST_ENV=true
    export DATABASE_TYPE=inmemory
    export REDIS_ENABLED=false
    export ALERT_ENABLED=false
    
    print_message $YELLOW "运行集成测试..."
    go test -v -tags=integration ./internal/loomi/testing/
    
    if [ $? -eq 0 ]; then
        print_message $GREEN "集成测试通过"
    else
        print_message $RED "集成测试失败"
        exit 1
    fi
    
    # 清理环境变量
    unset TEST_ENV
    unset DATABASE_TYPE
    unset REDIS_ENABLED
    unset ALERT_ENABLED
}

# 运行性能测试
run_performance_tests() {
    print_title "运行性能测试"
    
    print_message $YELLOW "运行基准测试..."
    go test -bench=. -benchmem -run=^$ ./...
    
    if [ $? -eq 0 ]; then
        print_message $GREEN "性能测试通过"
    else
        print_message $RED "性能测试失败"
        exit 1
    fi
}

# 运行压力测试
run_stress_tests() {
    print_title "运行压力测试"
    
    print_message $YELLOW "运行压力测试..."
    go test -v -tags=stress -timeout=10m ./internal/loomi/testing/
    
    if [ $? -eq 0 ]; then
        print_message $GREEN "压力测试通过"
    else
        print_message $RED "压力测试失败"
        exit 1
    fi
}

# 运行端到端测试
run_e2e_tests() {
    print_title "运行端到端测试"
    
    print_message $YELLOW "运行端到端测试..."
    go test -v -tags=e2e -timeout=30m ./internal/loomi/testing/
    
    if [ $? -eq 0 ]; then
        print_message $GREEN "端到端测试通过"
    else
        print_message $RED "端到端测试失败"
        exit 1
    fi
}

# 生成覆盖率报告
generate_coverage_report() {
    print_title "生成覆盖率报告"
    
    if [ -f "coverage.out" ]; then
        print_message $YELLOW "生成HTML覆盖率报告..."
        go tool cover -html=coverage.out -o coverage.html
        
        print_message $YELLOW "生成XML覆盖率报告..."
        gocov convert coverage.out | gocov-xml > coverage.xml
        
        print_message $YELLOW "显示覆盖率统计..."
        go tool cover -func=coverage.out
        
        print_message $GREEN "覆盖率报告生成完成"
        print_message $GREEN "HTML报告: coverage.html"
        print_message $GREEN "XML报告: coverage.xml"
    else
        print_message $YELLOW "未找到覆盖率文件，跳过报告生成"
    fi
}

# 运行代码质量检查
run_code_quality_checks() {
    print_title "运行代码质量检查"
    
    # 检查代码格式
    print_message $YELLOW "检查代码格式..."
    if ! gofmt -l . | grep -q .; then
        print_message $GREEN "代码格式检查通过"
    else
        print_message $RED "代码格式不符合规范"
        gofmt -l .
        exit 1
    fi
    
    # 运行静态分析
    print_message $YELLOW "运行静态分析..."
    go vet ./...
    
    if [ $? -eq 0 ]; then
        print_message $GREEN "静态分析通过"
    else
        print_message $RED "静态分析发现问题"
        exit 1
    fi
    
    # 运行linter（如果安装了golangci-lint）
    if command -v golangci-lint &> /dev/null; then
        print_message $YELLOW "运行golangci-lint..."
        golangci-lint run
        
        if [ $? -eq 0 ]; then
            print_message $GREEN "Linter检查通过"
        else
            print_message $RED "Linter检查发现问题"
            exit 1
        fi
    else
        print_message $YELLOW "未安装golangci-lint，跳过linter检查"
    fi
}

# 清理测试文件
cleanup_test_files() {
    print_title "清理测试文件"
    
    print_message $YELLOW "清理临时文件..."
    rm -f coverage.out
    rm -f *.test
    rm -f test-results.xml
    
    print_message $GREEN "清理完成"
}

# 显示帮助信息
show_help() {
    echo "用法: $0 [选项]"
    echo
    echo "选项:"
    echo "  -h, --help              显示帮助信息"
    echo "  -u, --unit              只运行单元测试"
    echo "  -i, --integration       只运行集成测试"
    echo "  -p, --performance       只运行性能测试"
    echo "  -s, --stress            只运行压力测试"
    echo "  -e, --e2e              只运行端到端测试"
    echo "  -q, --quality           只运行代码质量检查"
    echo "  -c, --coverage          只生成覆盖率报告"
    echo "  -a, --all               运行所有测试（默认）"
    echo "  --clean                 清理测试文件"
    echo
    echo "示例:"
    echo "  $0                      # 运行所有测试"
    echo "  $0 -u                   # 只运行单元测试"
    echo "  $0 -u -p                # 运行单元测试和性能测试"
    echo "  $0 --clean              # 清理测试文件"
}

# 主函数
main() {
    local run_unit=false
    local run_integration=false
    local run_performance=false
    local run_stress=false
    local run_e2e=false
    local run_quality=false
    local run_coverage=false
    local run_all=true
    local clean_only=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -u|--unit)
                run_unit=true
                run_all=false
                shift
                ;;
            -i|--integration)
                run_integration=true
                run_all=false
                shift
                ;;
            -p|--performance)
                run_performance=true
                run_all=false
                shift
                ;;
            -s|--stress)
                run_stress=true
                run_all=false
                shift
                ;;
            -e|--e2e)
                run_e2e=true
                run_all=false
                shift
                ;;
            -q|--quality)
                run_quality=true
                run_all=false
                shift
                ;;
            -c|--coverage)
                run_coverage=true
                run_all=false
                shift
                ;;
            -a|--all)
                run_all=true
                shift
                ;;
            --clean)
                clean_only=true
                shift
                ;;
            *)
                print_message $RED "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 如果只是清理，直接清理并退出
    if [ "$clean_only" = true ]; then
        cleanup_test_files
        exit 0
    fi
    
    # 检查Go环境
    check_go_environment
    
    # 安装测试依赖
    install_test_dependencies
    
    # 根据参数运行相应的测试
    if [ "$run_all" = true ] || [ "$run_unit" = true ]; then
        run_unit_tests
    fi
    
    if [ "$run_all" = true ] || [ "$run_integration" = true ]; then
        run_integration_tests
    fi
    
    if [ "$run_all" = true ] || [ "$run_performance" = true ]; then
        run_performance_tests
    fi
    
    if [ "$run_all" = true ] || [ "$run_stress" = true ]; then
        run_stress_tests
    fi
    
    if [ "$run_all" = true ] || [ "$run_e2e" = true ]; then
        run_e2e_tests
    fi
    
    if [ "$run_all" = true ] || [ "$run_quality" = true ]; then
        run_code_quality_checks
    fi
    
    if [ "$run_all" = true ] || [ "$run_coverage" = true ]; then
        generate_coverage_report
    fi
    
    print_title "测试完成"
    print_message $GREEN "所有测试已成功完成！"
}

# 运行主函数
main "$@"

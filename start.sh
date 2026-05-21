#!/bin/bash

# Virtual Company Platform 启动/停止脚本
# 用法: ./start.sh [--force|--stop|--prod]

set -e

# 端口配置
BACKEND_PORT=8181
FRONTEND_PORT=8100
PROD_MODE=false

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查端口是否被占用
check_port() {
    local port=$1
    if lsof -i :$port | grep -q LISTEN; then
        return 0  # 端口被占用
    else
        return 1  # 端口空闲
    fi
}

# 杀掉占用端口的进程
kill_port() {
    local port=$1
    echo -e "${YELLOW}杀掉端口 $port 的进程...${NC}"
    lsof -i :$port | grep LISTEN | awk '{print $2}' | xargs kill -9 2>/dev/null || true
    sleep 1
}

# 停止所有服务
stop_all() {
    echo -e "${YELLOW}停止所有服务...${NC}"
    if check_port $BACKEND_PORT; then
        kill_port $BACKEND_PORT
        echo -e "${GREEN}✓ 后端已停止${NC}"
    else
        echo -e "${YELLOW}后端未运行${NC}"
    fi
    if check_port $FRONTEND_PORT; then
        kill_port $FRONTEND_PORT
        echo -e "${GREEN}✓ 前端已停止${NC}"
    else
        echo -e "${YELLOW}前端未运行${NC}"
    fi
    echo -e "${GREEN}所有服务已停止${NC}"
    exit 0
}

# 启动后端
start_backend() {
    echo -e "${GREEN}启动后端服务 (端口 $BACKEND_PORT)...${NC}"

    # 设置 CORS 允许的来源
    if [ "$PROD_MODE" = true ]; then
        # 生产模式下 nginx 统一代理，同源访问不需要严格 CORS
        # 设置 * 允许所有来源，兼容性更好
        export ALLOWED_ORIGIN="*"
        echo -e "${YELLOW}生产模式: ALLOWED_ORIGIN=*${NC}"
    else
        export ALLOWED_ORIGIN="http://localhost:8100"
    fi

    cd backend
    nohup go run cmd/server/main.go > /tmp/company-backend.log 2>&1 &
    BACKEND_PID=$!
    cd ..
    sleep 2

    if check_port $BACKEND_PORT; then
        echo -e "${GREEN}✓ 后端启动成功 (PID: $BACKEND_PID)${NC}"
        echo "   API: http://localhost:$BACKEND_PORT"
    else
        echo -e "${RED}✗ 后端启动失败，请检查日志: /tmp/company-backend.log${NC}"
        exit 1
    fi
}

# 启动前端
start_frontend() {
    echo -e "${GREEN}启动前端服务 (端口 $FRONTEND_PORT)...${NC}"
    cd frontend

    # 检查是否需要安装依赖
    if [ ! -d "node_modules" ]; then
        echo -e "${YELLOW}安装前端依赖...${NC}"
        npm install
    fi

    # 生产模式设置环境变量并启动
    if [ "$PROD_MODE" = true ]; then
        nohup env PROD_MODE=true npm run dev > /tmp/company-frontend.log 2>&1 &
    else
        nohup npm run dev > /tmp/company-frontend.log 2>&1 &
    fi
    FRONTEND_PID=$!
    cd ..
    sleep 3

    if check_port $FRONTEND_PORT; then
        echo -e "${GREEN}✓ 前端启动成功 (PID: $FRONTEND_PID)${NC}"
        echo "   UI: http://localhost:$FRONTEND_PORT"
    else
        echo -e "${RED}✗ 前端启动失败，请检查日志: /tmp/company-frontend.log${NC}"
        exit 1
    fi
}

# 主流程
echo ""
echo "========================================"
echo "  Virtual Company Platform 启动脚本"
echo "========================================"
echo ""
echo "用法: ./start.sh [--force|--stop|--prod]"
echo "  --force  强制重启（杀掉现有进程）"
echo "  --stop   停止所有服务"
echo "  --prod   生产模式启动（允许所有来源 CORS）"
echo ""

# 处理参数
case "$1" in
    --stop)
        stop_all
        ;;
    --force)
        FORCE=true
        ;;
    --prod)
        FORCE=true
        PROD_MODE=true
        ;;
    *)
        FORCE=false
        ;;
esac

# 检查并处理端口
echo -e "${YELLOW}检查端口状态...${NC}"

if check_port $BACKEND_PORT; then
    if [ "$FORCE" = true ]; then
        kill_port $BACKEND_PORT
    else
        echo -e "${YELLOW}端口 $BACKEND_PORT 已被占用${NC}"
        echo "  使用 ./start.sh --force 强制重启"
        read -p "  是否杀掉并重启? (y/n): " choice
        if [ "$choice" == "y" ]; then
            kill_port $BACKEND_PORT
        else
            echo -e "${GREEN}后端服务已在运行，跳过启动${NC}"
        fi
    fi
fi

if check_port $FRONTEND_PORT; then
    if [ "$FORCE" = true ]; then
        kill_port $FRONTEND_PORT
    else
        echo -e "${YELLOW}端口 $FRONTEND_PORT 已被占用${NC}"
        read -p "  是否杀掉并重启? (y/n): " choice
        if [ "$choice" == "y" ]; then
            kill_port $FRONTEND_PORT
        else
            echo -e "${GREEN}前端服务已在运行，跳过启动${NC}"
        fi
    fi
fi

echo ""

# 启动服务
if ! check_port $BACKEND_PORT; then
    start_backend
fi

if ! check_port $FRONTEND_PORT; then
    start_frontend
fi

echo ""
echo "========================================"
echo -e "${GREEN}  服务启动完成!${NC}"
echo "========================================"
echo ""
echo "访问地址:"
echo "  前端 UI:  http://localhost:$FRONTEND_PORT"
echo "  后端 API: http://localhost:$BACKEND_PORT"
echo ""
echo "日志文件:"
echo "  后端: /tmp/company-backend.log"
echo "  前端: /tmp/company-frontend.log"
echo ""
echo "停止服务: ./start.sh --stop"
echo ""
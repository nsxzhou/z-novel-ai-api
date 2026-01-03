#!/bin/bash
set -e

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( dirname "$SCRIPT_DIR" )"

echo "Running buf generate in $PROJECT_ROOT/api/proto..."

# 进入 api/proto 目录并执行 buf generate
cd "$PROJECT_ROOT/api/proto" && buf generate

echo "gRPC code generation completed successfully."

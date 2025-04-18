#!/usr/bin/env python3
"""
MIT License

Copyright (c) 2024 Bamboo

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
"""

import os
import sys
import argparse
import logging

# 添加项目根目录到路径
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from api.fastapi_app import start_server
from utils.logger import get_logger
from utils.config import get_config, reload_config

logger = get_logger("startup")

def parse_args():
    parser = argparse.ArgumentParser(description="AIOps服务启动脚本")
    parser.add_argument("--config", type=str, help="自定义配置文件路径")
    parser.add_argument("--log-level", type=str, choices=["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"],
                        help="日志级别")
    parser.add_argument("--port", type=int, help="API服务端口")
    return parser.parse_args()

def main():
    """主函数，启动服务"""
    args = parse_args()
    
    # 设置日志级别
    if args.log_level:
        logging.getLogger("aiops").setLevel(getattr(logging, args.log_level))
    
    # 加载自定义配置
    if args.config:
        os.environ["AIOPS_CONFIG"] = args.config
        reload_config()
    
    # 覆盖端口配置
    if args.port:
        os.environ["AIOPS_PORT"] = str(args.port)
    
    logger.info("Starting AIOps services...")
    
    try:
        # 启动API服务器
        start_server()
    except KeyboardInterrupt:
        logger.info("Shutting down AIOps services...")
    except Exception as e:
        logger.error(f"Error starting services: {e}", exc_info=True)
        sys.exit(1)

if __name__ == "__main__":
    main()
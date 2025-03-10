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
import subprocess
import argparse
import logging

# 添加项目根目录到路径
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

logger = logging.getLogger(__name__)

def check_and_install_dependencies():
    """检查并安装必要的依赖"""
    required_packages = [
        "langchain",
        "openai",
        "chromadb",
        "tiktoken",
        "unstructured",
        "pypdf",
        "pandas",
        "requests"
    ]

    optional_packages = {
        "ollama": "使用本地Ollama模型",
        "sentence-transformers": "使用本地嵌入模型",
        "torch": "深度学习支持",
        "transformers": "Hugging Face模型支持"
    }

    logger.info("检查必要依赖...")
    for package in required_packages:
        try:
            __import__(package.replace("-", "_"))
            logger.info(f"✓ {package} 已安装")
        except ImportError:
            logger.warning(f"! {package} 未安装，正在安装...")
            subprocess.check_call([sys.executable, "-m", "pip", "install", package])
            logger.info(f"✓ {package} 安装完成")

    logger.info("\n检查可选依赖...")
    for package, description in optional_packages.items():
        try:
            __import__(package.replace("-", "_"))
            logger.info(f"✓ {package} 已安装 ({description})")
        except ImportError:
            logger.warning(f"! {package} 未安装 ({description})")
            install = input(f"是否安装 {package}? (y/n): ").lower() == 'y'
            if install:
                subprocess.check_call([sys.executable, "-m", "pip", "install", package])
                logger.info(f"✓ {package} 安装完成")
            else:
                logger.info(f"× 跳过安装 {package}")

def create_directories():
    """创建必要的目录"""
    dirs = [
        "./knowledge_docs",
        "./data/storage/vector_store",
        "./models/data/qa_dataset",
        "./models/finetuned-deepseek-qa"
    ]

    logger.info("\n创建必要目录...")
    for d in dirs:
        os.makedirs(d, exist_ok=True)
        logger.info(f"✓ 目录已创建: {d}")

def setup_environment():
    """设置环境变量"""
    env_vars = {
        "LLM_PROVIDER": "使用的LLM提供者 (openai 或 ollama)",
        "OPENAI_API_KEY": "OpenAI API密钥 (如果使用OpenAI)",
        "LLM_MODEL": "使用的模型名称",
        "OLLAMA_HOST": "Ollama服务地址 (如果使用Ollama，默认: http://127.0.0.1:11434)",
        "EMBEDDING_MODEL": "嵌入模型名称",
        "VECTOR_STORE_DIR": "向量存储目录路径",
        "DOCS_DIR": "知识文档目录路径"
    }

    logger.info("\n设置环境变量...")

    # 检查是否已有环境变量配置文件
    env_file = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), ".env")
    existing_vars = {}

    if os.path.exists(env_file):
        logger.info(f"发现现有环境变量配置文件: {env_file}")
        with open(env_file, 'r') as f:
            for line in f:
                if '=' in line and not line.startswith('#'):
                    key, value = line.strip().split('=', 1)
                    existing_vars[key] = value

    # 设置默认值
    default_values = {
        "LLM_PROVIDER": "ollama",
        "LLM_MODEL": "deepseek-r1:8b",
        "OLLAMA_HOST": "http://127.0.0.1:11434",
        "EMBEDDING_MODEL": "nomic-embed-text:latest",
        "VECTOR_STORE_DIR": "./data/storage/vector_store",
        "DOCS_DIR": "./knowledge_docs"
    }

    # 询问用户设置环境变量
    new_vars = {}
    for var, description in env_vars.items():
        current_value = existing_vars.get(var, os.environ.get(var, default_values.get(var, '')))
        print(f"\n{var}: {description}")
        print(f"当前值: {current_value or '未设置'}")

        new_value = input(f"输入新值 (直接回车保持当前值): ")
        if new_value:
            new_vars[var] = new_value
        elif current_value:
            new_vars[var] = current_value

    # 保存环境变量
    with open(env_file, 'w') as f:
        for var, value in new_vars.items():
            f.write(f"{var}={value}\n")

    logger.info(f"环境变量已保存到: {env_file}")
    logger.info("请确保在运行前加载这些环境变量，例如:")
    logger.info(f"source {env_file}")

def main():
    parser = argparse.ArgumentParser(description='设置RAG系统环境')
    parser.add_argument('--skip-deps', action='store_true', help='跳过依赖检查')
    parser.add_argument('--skip-env', action='store_true', help='跳过环境变量设置')
    parser.add_argument('--skip-dirs', action='store_true', help='跳过目录创建')
    args = parser.parse_args()

    print("="*50)
    print("RAG系统环境设置")
    print("="*50)

    if not args.skip_deps:
        check_and_install_dependencies()

    if not args.skip_dirs:
        create_directories()

    if not args.skip_env:
        setup_environment()

    print("\n"+"="*50)
    print("设置完成！现在您可以运行以下命令来测试RAG系统:")
    print("python scripts/rag_demo.py")
    print("或者运行诊断工具检查系统状态:")
    print("python scripts/diagnose_rag.py")
    print("="*50)

if __name__ == "__main__":
    main()
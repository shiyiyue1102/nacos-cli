# Nacos CLI

一个功能强大的 Nacos 命令行工具，支持快速连接、管理和操作 Nacos 配置中心和服务注册中心。

## 特性

- 🚀 快速安装和启动
- 🪶 **轻量级设计** - 无需重量级 SDK，仅使用标准 HTTP 请求
- 💻 交互式终端界面
- 🎨 美观的彩色输出和语法高亮
- 📝 支持配置管理（查看、发布、删除）
- 🔍 支持服务发现和查询
- ⚙️ 灵活的连接参数配置

## 安装

### 从源码安装

```bash
# 克隆仓库
git clone https://github.com/yourusername/nacos-cli.git
cd nacos-cli

# 安装依赖
pip install -r requirements.txt

# 安装到系统
pip install -e .
```

### 使用 pip 安装（待发布）

```bash
pip install nacos-cli
```

## 快速开始

### 连接到本地 Nacos

默认连接到 `127.0.0.1:8848`：

```bash
nacos-cli
```

### 连接到远程 Nacos

```bash
nacos-cli --host 192.168.1.100 --port 8848
```

或使用简写：

```bash
nacos-cli -h 192.168.1.100 -p 8848
```

### 指定用户名和密码

```bash
nacos-cli -h 192.168.1.100 -p 8848 -u admin -pw admin123
```

### 指定命名空间

```bash
nacos-cli -h 192.168.1.100 -p 8848 -n your-namespace-id
```

## 使用指南

连接成功后，会进入交互式终端，支持以下命令：

### 配置管理

#### 列出所有配置

```bash
nacos> list
# 或
nacos> ls
```

#### 获取配置内容

```bash
nacos> get <dataId> [group]

# 示例
nacos> get myconfig
nacos> get myconfig DEFAULT_GROUP
```

#### 发布/更新配置

```bash
nacos> set <dataId> <group> <content>

# 示例
nacos> set myconfig DEFAULT_GROUP 'server.port=8080'
nacos> set app.yml DEFAULT_GROUP 'key: value'
```

#### 删除配置

```bash
nacos> delete <dataId> [group]
# 或
nacos> rm <dataId> [group]

# 示例
nacos> delete myconfig
nacos> rm myconfig DEFAULT_GROUP
```

### 服务管理

#### 列出所有服务

```bash
nacos> services
# 或
nacos> svc
```

#### 查看服务详情

```bash
nacos> service <serviceName> [group]
# 或
nacos> detail <serviceName> [group]

# 示例
nacos> service myservice
nacos> detail myservice DEFAULT_GROUP
```

### 其他命令

```bash
nacos> help        # 显示帮助信息
nacos> clear       # 清空屏幕
nacos> exit        # 退出终端
```

## 命令行参数

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| --host | -h | 127.0.0.1 | Nacos 服务器地址 |
| --port | -p | 8848 | Nacos 服务器端口 |
| --username | -u | nacos | Nacos 用户名 |
| --password | -pw | nacos | Nacos 密码 |
| --namespace | -n | (空) | Nacos 命名空间 ID |
| --version | | | 显示版本信息 |
| --help | | | 显示帮助信息 |

## 依赖

- Python 3.8+
- click - 命令行参数解析
- rich - 美化终端输出
- prompt-toolkit - 交互式终端
- requests - HTTP 请求

**注意**：本项目使用纯 HTTP REST API 与 Nacos 通信，不依赖官方 SDK，更加轻量。

## 开发

### 安装开发环境

```bash
git clone https://github.com/yourusername/nacos-cli.git
cd nacos-cli
pip install -r requirements.txt
pip install -e .
```

### 运行

```bash
python -m nacos_cli.main
```

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License

## 作者

Your Name

## 更新日志

### v0.1.0 (2026-01-27)

- ✨ 初始版本发布
- 支持基本的配置管理功能
- 支持服务查询功能
- 交互式终端界面
- 语法高亮显示

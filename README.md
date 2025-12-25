# Cloud-DDNS

[![Build and Test](https://github.com/NewFuture/Cloud-DDNS/actions/workflows/build.yml/badge.svg)](https://github.com/NewFuture/Cloud-DDNS/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/NewFuture/CloudDDNS)](https://goreportcard.com/report/github.com/NewFuture/CloudDDNS)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

**Cloud-DDNS** 是一个轻量级的中间件服务，旨在连接支持 **GnuDIP** 协议的老旧网络设备（如路由器、DVR、NAS）与现代云 DNS 服务商（阿里云、腾讯云、Cloudflare 等）。它充当一个 GnuDIP 服务端，接收设备的动态 IP 更新请求，并调用云厂商 API 更新 DNS 记录。

[English](#english) | [中文](#中文)

---

## 中文

### 功能特性

- ✅ **GnuDIP 协议支持**：实现标准 GnuDIP 协议（TCP 3495）服务端
- ✅ **HTTP 简易模式**：支持 GnuDIP.HTTP 更新方式
- ✅ **透传式鉴权**：直接使用云厂商 AccessKey/SecretKey 作为用户名/密码
- ✅ **多云厂商支持**：
  - 阿里云 DNS (Aliyun)
  - 腾讯云 DNSPod (Tencent)
  - 可扩展支持更多厂商
- ✅ **智能更新**：IP 未变化时不调用 API，节省成本
- ✅ **Docker 支持**：提供 Docker 镜像，快速部署

### 快速开始

#### 方式一：Docker（推荐）

1. 克隆仓库：
```bash
git clone https://github.com/NewFuture/CloudDDNS.git
cd CloudDDNS
```

2. 创建配置文件：
```bash
cp config.yaml.example config.yaml
# 编辑 config.yaml，填入你的云厂商凭证
```

3. 构建并运行：
```bash
docker build -t cloud-ddns:latest .
docker run -d -p 3495:3495 -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml:ro --name cloud-ddns cloud-ddns:latest
```

#### 方式二：二进制运行

1. 下载编译好的二进制文件（或自行编译）：
```bash
go build -o cloud-ddns main.go
```

2. 创建配置文件 `config.yaml`：
```yaml
server:
  tcp_port: 3495   # GnuDIP 标准端口
  http_port: 8080  # HTTP 兼容端口

users:
  # 阿里云用户示例
  - username: "LTAI4Fxxxxx"     # Aliyun AccessKey ID
    password: "YourSecretKey"   # Aliyun AccessKey Secret
    provider: "aliyun"
  
  # 腾讯云用户示例
  - username: "123456"          # Dnspod ID
    password: "TokenValue"      # Dnspod Token
    provider: "tencent"
```

3. 运行服务：
```bash
./cloud-ddns
```

#### 方式三：Azure Container Apps 部署

1. 创建 Azure Container App：
```bash
# 使用 Azure CLI
az containerapp create \
  --name cloud-ddns \
  --resource-group <your-resource-group> \
  --environment <your-environment> \
  --image <your-registry>/cloud-ddns:latest \
  --target-port 8080 \
  --ingress external \
  --secrets config-yaml=<base64-encoded-config> \
  --env-vars CONFIG_PATH=/app/config.yaml
```

2. 配置文件可通过 Azure Key Vault 或 Container App Secrets 管理
3. 支持自动缩放和高可用性部署

### 客户端配置

在您的路由器、DVR 或 NAS 上配置 DDNS：

| 配置项 | 值 |
|--------|-----|
| **服务商** | GnuDIP 或 Custom |
| **服务器地址** | 您部署 Cloud-DDNS 的服务器 IP/域名 |
| **端口** | 3495 (TCP) 或 8080 (HTTP) |
| **用户名** | 云厂商 AccessKey ID |
| **密码** | 云厂商 AccessKey Secret |
| **域名** | 完整域名，如 `camera.example.com` |

#### 光猫/路由器兼容性

本服务全面兼容华为、中兴等光猫及主流路由器固件的 GnuDIP 协议实现：

**支持的 HTTP 路径：**
- `/` （根路径）
- `/update`
- `/nic/update`
- `/cgi-bin/gdipupdt.cgi`（返回数字响应 0/1/2，支持 reqc 模式）

**支持的参数别名（不区分大小写）：**
- 域名：`hostname`, `host`, `domn`, `domain`
- 用户名：`username`, `user`, `usr`, `name`
- 密码：`password`, `pass`, `pwd`
- IP 地址：`myip`, `ip`, `addr`

**密码格式支持（自动识别）：**
- 明文密码（推荐存储在配置文件中）
- MD5 哈希（32位十六进制，支持大小写）
- SHA256 哈希（64位十六进制，支持大小写）
- Base64 编码

服务器会自动尝试以下验证方式：
1. 直接明文匹配
2. 将配置文件密码进行 MD5/SHA256 编码后与请求密码对比
3. 将请求密码进行 Base64 解码后与配置文件密码对比
4. 将配置文件密码视为哈希值，对请求密码进行哈希后对比

这样可以兼容不同光猫/路由器的密码传输方式。

**响应格式（符合标准 GnuDIP 协议）：**
- 成功：`good <ip>` 
- 认证失败：`badauth`
- 域名无效：`notfqdn`
- 系统错误：`911`

**Reqc 模式（GnuDIP）：**
- `reqc=0`（默认）：按请求 IP 更新；IP 为空时使用客户端源地址
- `reqc=1`：离线模式，记录更新为 `0.0.0.0`，CGI 路径成功时返回数字 `2`
- `reqc=2`：自动检测模式，忽略传入 IP，始终使用客户端源地址

#### HTTP 方式调用示例

```bash
# 标准参数名（适用于大部分设备）
curl "http://your-server:8080/?user=LTAI4Fxxxxx&pass=YourSecret&domn=camera.example.com&addr=1.2.3.4"

# 使用参数别名（兼容不同固件）
curl "http://your-server:8080/nic/update?username=LTAI4Fxxxxx&password=YourSecret&hostname=camera.example.com&myip=1.2.3.4"

# 省略 IP 参数时自动使用客户端 IP
curl "http://your-server:8080/?user=LTAI4Fxxxxx&pass=YourSecret&domn=camera.example.com"
```

### 目录结构

```
CloudDDNS/
├── main.go                 # 程序入口
├── config.yaml.example     # 配置文件示例
├── Dockerfile              # Docker 镜像构建
├── go.mod                  # Go 模块定义
├── go.sum                  # 依赖版本锁定
└── pkg/
    ├── config/             # 配置加载模块
    │   └── config.go
    ├── provider/           # 云厂商适配层
    │   ├── provider.go     # 接口定义
    │   ├── aliyun.go       # 阿里云实现
    │   └── tencent.go      # 腾讯云实现
    └── server/             # 协议处理层
        └── server.go       # TCP & HTTP 服务
```

### 开发

#### 编译

```bash
go build -o cloud-ddns main.go
```

#### 添加新的云厂商支持

1. 在 `pkg/provider/` 下创建新的 provider 文件（如 `cloudflare.go`）
2. 实现 `Provider` 接口的 `UpdateRecord` 方法
3. 在 `pkg/provider/provider.go` 的 `GetProvider` 函数中添加对应的 case

### 测试

运行测试：
```bash
# 运行所有测试
make test

# 运行测试（详细输出）
make test-verbose

# 运行测试并生成覆盖率报告
make test-coverage
```

### 安全说明

- **配置文件安全**：`config.yaml` 包含敏感凭证，已在 `.gitignore` 中排除
- **透传鉴权**：用户名密码直接用作 API 凭证，确保传输安全（建议使用 HTTPS 反向代理）
- **访问控制**：建议配置防火墙规则，仅允许受信任的设备访问

### 许可证

Apache License 2.0

---

## English

### Features

- ✅ **GnuDIP Protocol Support**: Implements standard GnuDIP protocol (TCP 3495) server
- ✅ **HTTP Simple Mode**: Supports GnuDIP.HTTP update method
- ✅ **Pass-through Authentication**: Uses cloud provider AccessKey/SecretKey as username/password directly
- ✅ **Multi-Cloud Provider Support**:
  - Alibaba Cloud DNS (Aliyun)
  - Tencent Cloud DNSPod
  - Extensible for more providers
- ✅ **Smart Updates**: Skips API calls when IP hasn't changed, saving costs
- ✅ **Docker Support**: Provides Docker image for quick deployment

### Quick Start

#### Method 1: Docker (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/NewFuture/CloudDDNS.git
cd CloudDDNS
```

2. Create configuration file:
```bash
cp config.yaml.example config.yaml
# Edit config.yaml and fill in your cloud provider credentials
```

3. Build and run:
```bash
docker build -t cloud-ddns:latest .
docker run -d -p 3495:3495 -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml:ro --name cloud-ddns cloud-ddns:latest
```

#### Method 2: Binary

1. Download the compiled binary (or build it yourself):
```bash
go build -o cloud-ddns main.go
```

2. Create `config.yaml`:
```yaml
server:
  tcp_port: 3495   # GnuDIP standard port
  http_port: 8080  # HTTP compatible port

users:
  # Aliyun user example
  - username: "LTAI4Fxxxxx"     # Aliyun AccessKey ID
    password: "YourSecretKey"   # Aliyun AccessKey Secret
    provider: "aliyun"
  
  # Tencent user example
  - username: "123456"          # Dnspod ID
    password: "TokenValue"      # Dnspod Token
    provider: "tencent"
```

3. Run the service:
```bash
./cloud-ddns
```

#### Method 3: Azure Container Apps Deployment

1. Create Azure Container App:
```bash
# Using Azure CLI
az containerapp create \
  --name cloud-ddns \
  --resource-group <your-resource-group> \
  --environment <your-environment> \
  --image <your-registry>/cloud-ddns:latest \
  --target-port 8080 \
  --ingress external \
  --secrets config-yaml=<base64-encoded-config> \
  --env-vars CONFIG_PATH=/app/config.yaml
```

2. Configuration can be managed via Azure Key Vault or Container App Secrets
3. Supports auto-scaling and high-availability deployment


### Client Configuration

Configure DDNS on your router, DVR, or NAS:

| Setting | Value |
|---------|-------|
| **Service Provider** | GnuDIP or Custom |
| **Server Address** | IP/domain where Cloud-DDNS is deployed |
| **Port** | 3495 (TCP) or 8080 (HTTP) |
| **Username** | Cloud provider AccessKey ID |
| **Password** | Cloud provider AccessKey Secret |
| **Domain** | Full domain name, e.g., `camera.example.com` |

#### Optical Modem / Router Compatibility

This service is fully compatible with GnuDIP protocol implementations in Huawei, ZTE optical modems and mainstream router firmwares:

**Supported HTTP Paths:**
- `/` (root path)
- `/update`
- `/nic/update`
- `/cgi-bin/gdipupdt.cgi` (returns numeric responses 0/1/2 with reqc support)

**Supported Parameter Aliases (case-insensitive):**
- Domain: `hostname`, `host`, `domn`, `domain`
- Username: `username`, `user`, `usr`, `name`
- Password: `password`, `pass`, `pwd`
- IP Address: `myip`, `ip`, `addr`

**Password Format Support (auto-detection):**
- Plaintext (recommended for config file)
- MD5 hash (32-character hex, case-insensitive)
- SHA256 hash (64-character hex, case-insensitive)
- Base64 encoding

The server automatically tries multiple verification methods:
1. Direct plaintext match
2. Hash config password (MD5/SHA256) and compare with request
3. Decode request password (Base64) and compare with config
4. Treat config as hash and hash request password for comparison

This ensures compatibility with various optical modem/router password transmission methods.

**Response Format (standard GnuDIP protocol):**
- Success: `good <ip>`
- Authentication failed: `badauth`
- Invalid domain: `notfqdn`
- System error: `911`

**Reqc Modes (GnuDIP):**
- `reqc=0` (default): update using provided IP; when absent, the client source IP is used
- `reqc=1`: offline mode updates the record to `0.0.0.0`; CGI path returns numeric `2` on success
- `reqc=2`: auto-detect mode ignores the provided IP and always uses the client source IP

#### HTTP Method Examples

```bash
# Standard parameter names (works with most devices)
curl "http://your-server:8080/?user=LTAI4Fxxxxx&pass=YourSecret&domn=camera.example.com&addr=1.2.3.4"

# Using parameter aliases (compatible with different firmwares)
curl "http://your-server:8080/nic/update?username=LTAI4Fxxxxx&password=YourSecret&hostname=camera.example.com&myip=1.2.3.4"

# Auto-detect client IP when IP parameter is omitted
curl "http://your-server:8080/?user=LTAI4Fxxxxx&pass=YourSecret&domn=camera.example.com"
```

### Development

#### Build

```bash
# Using Makefile (recommended)
make build

# Or directly with go
go build -o cloud-ddns main.go
```

#### Testing

Run tests:
```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage report
make test-coverage
```

#### Code Quality Checks

Before committing changes:

```bash
# Check code formatting
make fmt-check

# Auto-format code
make fmt

# Run static analysis
make vet

# Full pre-commit check
make test && make fmt-check && make vet && make build
```

#### Available Make Targets

- `make build` - Build the binary
- `make run` - Build and run the application
- `make test` - Run tests
- `make test-verbose` - Run tests with verbose output
- `make test-coverage` - Run tests with coverage report
- `make fmt` - Format Go code
- `make fmt-check` - Check code formatting without modifying files
- `make vet` - Run go vet
- `make docker` - Build Docker image
- `make clean` - Remove build artifacts
- `make help` - Show all available targets

#### Adding New Cloud Provider Support

1. Create a new provider file in `pkg/provider/` (e.g., `cloudflare.go`)
2. Implement the `UpdateRecord` method of the `Provider` interface
3. Add the corresponding case in the `GetProvider` function in `pkg/provider/provider.go`
4. Add tests for the new provider

### Security Notes

- **Configuration Security**: `config.yaml` contains sensitive credentials and is excluded in `.gitignore`
- **Pass-through Auth**: Username/password are used directly as API credentials, ensure secure transmission (HTTPS reverse proxy recommended)
- **Access Control**: Configure firewall rules to allow only trusted devices

### License

Apache License 2.0

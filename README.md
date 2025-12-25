# Cloud-DDNS

[![Build and Test](https://github.com/NewFuture/Cloud-DDNS/actions/workflows/build.yml/badge.svg)](https://github.com/NewFuture/Cloud-DDNS/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/NewFuture/CloudDDNS)](https://goreportcard.com/report/github.com/NewFuture/CloudDDNS)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

**Cloud-DDNS** 是一个轻量级的中间件服务，旨在连接支持 **GnuDIP** 协议的老旧网络设备（如路由器、DVR、NAS）与现代云 DNS 服务商（阿里云、腾讯云、Cloudflare 等）。它充当一个 GnuDIP 服务端，接收设备的动态 IP 更新请求，并调用云厂商 API 更新 DNS 记录。

[English](#english) | [中文](#中文)

---

## 中文

### 功能特性（面向使用者）

- ✅ **即插即用**：兼容路由器/光猫/NVR 的 GnuDIP、DynDNS/NIC 路径，直接可用
- ✅ **配置简单**：支持 Basic Auth 或 URL 参数，自动取客户端 IP，`reqc` 支持离线/源地址模式
- ✅ **直连云厂商**：阿里云、腾讯云开箱即用，凭证即用户名/密码，可扩展更多厂商
- ✅ **节省调用**：IP 未变不发起 DNS 更新，降低 API 成本
- ✅ **多种部署**：提供 Docker 镜像与二进制，快速上线
### 支持的 DDNS 协议 / 服务商

| 协议/服务商                         | 端点/端口                          | 认证方式                               | 关键参数 (别名)                                                | 响应示例                      |
| ----------------------------------- | ----------------------------------- | -------------------------------------- | -------------------------------------------------------------- | ----------------------------- |
| DynDNS / NIC / EasyDNS / Oray / DtDNS | `/`, `/update`, `/nic/update`       | Basic Auth 或 `user`/`pass`            | 域名：`hostname/host/domn/domain`；IP：`myip/ip/addr`           | `good <ip>` / `badauth` / `notfqdn` / `911` |
| GnuDIP HTTP                         | `/cgi-bin/gdipupdt.cgi`             | 两步：首请求返回 `time/sign`，二次 `md5(user:time:secret)` | `user/pass(sign)/domn/addr`；`reqc`=0/1/2；缺省 IP 用源地址    | 首次返回 meta；后续数字 `0/1/2` |
| GnuDIP TCP                          | TCP 3495                            | MD5 challenge-response                 | 报文：`user:hash:domain:reqc:addr`                             | 数字 `0/1/2`                  |

更多协议兼容（与上表覆盖关系）：

| 协议类 | API 路径/端口 | 认证 | 请求参数（名称=含义） | Response（典型） | 支持服务商 |
|---|---|---|---|---|---|
| DynDNS（DynDNS2 / NIC Update 族） | `/nic/update`（常见）；Oray 变体 `/ph/update`；3322 变体 `/dyndns/update` | HTTP Basic Auth 或 URL 内嵌 `user:pass` | `hostname`=FQDN（部分支持逗号多值）；`myip`=要设置 IP（可省略用源地址）；（3322 常见：`system`=更新系统类型） | `good <ip>` / `nochg <ip>` / `badauth` / `nohost` / `badagent` / `dnserr` / `911`（服务商略有差异） | DynDNS、No‑IP、DNS‑O‑Matic、Oray、3322(qDNS) |
| easyDNS（脚本端点） | `/dyn/tomato.php`，`/dyn/generic.php` | Query 凭据：`username`、`password` | `username`=账号；`password`=token；`hostname`=主机名；`myip`=IP | 兼容 DynDNS 响应 | easyDNS |
| DtDNS（AutoDNS） | `/api/autodns.cfm` | Query 凭据：`pw`（明文，建议 HTTPS） | `id`=FQDN；`pw`=密码；`ip`=IP（可选，缺省取源地址）；`client`=标识（可选） | 兼容 DynDNS 响应（实现上沿用 DynDNS 模式） | DtDNS |

**常用参数别名（不区分大小写）：**
- 用户：`user`,`username`,`usr`,`name` 或 Basic Auth
- 密码：`pass`,`password`,`pwd`
- GnuDIP HTTP 签名：`sign`（用于基于挑战的签名字段，具体计算方式以服务端实现为准）
- 域名：`hostname`,`host`,`domn`,`domain`
- IP：`myip`,`ip`,`addr`（缺省时使用客户端源地址）
- reqc（GnuDIP）：`0` 正常、`1` 离线(0.0.0.0)、`2` 使用源地址

### 快速开始

#### 方式一：Docker（推荐）
```bash
docker run -d -p 3495:3495 -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml:ro --name cloud-ddns cloud-ddns:latest
```

#### 方式二：二进制运行

下载编译好的二进制文件（或自行编译）：
```bash
./cloud-ddns -c config.yaml
```

### 客户端配置

在您的路由器、DVR 或 NAS 上配置 DDNS：

| 配置项 | 值 |
|--------|-----|
| **服务商** | GnuDIP 或 Custom |
| **服务器地址** | 您部署 Cloud-DDNS 的服务器 IP/域名 |
| **端口** | 3495 (TCP) 或 8080 (定义的HTTP） |
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

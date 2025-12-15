# Cloud-DDNS

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

#### 方式一：Docker Compose（推荐）

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

3. 启动服务：
```bash
docker-compose up -d
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

#### HTTP 方式调用示例

```bash
# 使用 URL 参数更新
curl "http://your-server:8080/?user=LTAI4Fxxxxx&pass=YourSecret&domn=camera.example.com&addr=1.2.3.4"

# 省略 addr 参数时自动使用客户端 IP
curl "http://your-server:8080/?user=LTAI4Fxxxxx&pass=YourSecret&domn=camera.example.com"
```

### 目录结构

```
CloudDDNS/
├── main.go                 # 程序入口
├── config.yaml.example     # 配置文件示例
├── Dockerfile              # Docker 镜像构建
├── docker-compose.yml      # Docker Compose 配置
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

### 安全说明

- **配置文件安全**：`config.yaml` 包含敏感凭证，已在 `.gitignore` 中排除
- **透传鉴权**：用户名密码直接用作 API 凭证，确保传输安全（建议使用 HTTPS 反向代理）
- **访问控制**：建议配置防火墙规则，仅允许受信任的设备访问

### 许可证

MIT License

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

#### Method 1: Docker Compose (Recommended)

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

3. Start the service:
```bash
docker-compose up -d
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

#### HTTP Method Example

```bash
# Update with URL parameters
curl "http://your-server:8080/?user=LTAI4Fxxxxx&pass=YourSecret&domn=camera.example.com&addr=1.2.3.4"

# Auto-detect client IP when addr is omitted
curl "http://your-server:8080/?user=LTAI4Fxxxxx&pass=YourSecret&domn=camera.example.com"
```

### Development

#### Build

```bash
go build -o cloud-ddns main.go
```

#### Adding New Cloud Provider Support

1. Create a new provider file in `pkg/provider/` (e.g., `cloudflare.go`)
2. Implement the `UpdateRecord` method of the `Provider` interface
3. Add the corresponding case in the `GetProvider` function in `pkg/provider/provider.go`

### Security Notes

- **Configuration Security**: `config.yaml` contains sensitive credentials and is excluded in `.gitignore`
- **Pass-through Auth**: Username/password are used directly as API credentials, ensure secure transmission (HTTPS reverse proxy recommended)
- **Access Control**: Configure firewall rules to allow only trusted devices

### License

MIT License

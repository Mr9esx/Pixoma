# Pixoma 安装与部署

Pixoma 控制面建议先用安装脚本部署，再通过管理后台完成数据库、对象存储、Telegram 和计算节点配置。当前安装器支持 macOS、Linux，以及通过 Git Bash 运行的 Windows，均覆盖 `amd64` / `arm64`。

## 快速安装控制面

```bash
curl -fsSL https://pixoma.miaoplus.com/install.sh | sh
```

如果希望先检查脚本再执行：

```bash
curl -fsSL https://pixoma.miaoplus.com/install.sh -o pixoma-install.sh
less pixoma-install.sh
sh pixoma-install.sh
```

安装完成后运行：

```bash
pixoma
```

Windows 用户请在 Git Bash 或 MSYS 终端中执行上面的命令。安装后二进制名称是：

```bash
pixoma.exe
```

启动日志会输出管理后台地址和首次启动的默认管理员账密。登录后先修改密码，再进入初始化向导完成配置。

固定版本安装：

```bash
curl -fsSL https://pixoma.miaoplus.com/install.sh | PIXOMA_VERSION=v0.1.0 sh
```

自定义安装目录：

```bash
curl -fsSL https://pixoma.miaoplus.com/install.sh | PIXOMA_INSTALL_DIR="$HOME/.pixoma/bin" sh
```

Windows 默认安装到 `~/.pixoma/bin`（对应 `%USERPROFILE%\.pixoma\bin`）。如果该目录不在 PATH 中，按脚本输出的提示添加。

## 添加计算节点

计算节点不会自动创建。先在管理后台新增节点，复制 `CONTROL_PLANE_URL`、`AGENT_TOKEN` 和 `EDGE_ID`，再在 GPU 机器上执行：

```bash
curl -fsSL https://pixoma.miaoplus.com/install.sh | \
  PIXOMA_INSTALL=edge \
  CONTROL_PLANE_URL=https://pixoma.example.com \
  AGENT_TOKEN=你的节点Token \
  EDGE_ID=你的节点ID \
  sh
```

安装脚本会把 `pixoma-edge-agent` 放入 `PIXOMA_INSTALL_DIR`（默认是 `/usr/local/bin` 或 `~/.pixoma/bin`），并校验下载包的 SHA-256。Edge 只需要能出站访问控制面，不要把本机目录作为远程部署的对象存储。

前台运行：

```bash
pixoma-edge-agent
```

生产环境建议用 `systemd` 或 `launchd` 托管进程，并为控制面配置 HTTPS 反向代理。

## 从源码构建

在仓库根目录执行：

```bash
make build
```

如需生成发布归档：

```bash
scripts/package-release.sh darwin arm64 v0.1.0
```

脚本会构建 `pixoma` 和 `pixoma-edge-agent`，并在 `release/` 目录生成 `tar.gz` 归档与 checksum。Windows 产物是 `pixoma.exe` 和 `pixoma-edge-agent.exe`，例如：

```bash
scripts/package-release.sh windows arm64 v0.1.0
```

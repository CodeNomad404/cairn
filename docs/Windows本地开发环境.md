# Windows 本地开发环境

本文用于帮助 Windows 开发者在本机运行 Cairn，先完成后端与数据库的本地部署，为后续前端开发和接口联调准备环境。

前端技术方案尚未确定，可能选择 Vue、React 或其他方案。因此本文不规定前端框架、目录结构、构建工具和开发端口，只说明当前已经确定的本地基础环境。

## 一、本地环境结构

当前需要在本机运行两个已确定的服务：

```text
浏览器或未来的前端开发服务器
              │
              ▼
Go 后端：http://127.0.0.1:8080
              │
              ▼
PostgreSQL：127.0.0.1:5432
```

PostgreSQL 运行在 Docker Desktop 中，Go 后端直接运行在 Windows 上。

未来确定前端技术后，开发结构预计会变成：

```text
浏览器
  │
  ▼
前端开发服务器：http://localhost:端口待定
  │  调用接口
  ▼
Go 后端：http://127.0.0.1:8080
  │
  ▼
PostgreSQL：127.0.0.1:5432
```

前端框架、开发服务器端口、API 路径、跨域策略和登录方式，需要在技术选型完成后另行确定。

## 二、需要安装的软件

在 Windows 中安装以下软件：

| 软件 | 用途 | 要求 |
| --- | --- | --- |
| [Git for Windows](https://git-scm.com/downloads/win) | 克隆仓库、管理分支 | 当前稳定版 |
| [Go](https://go.dev/doc/install) | 运行后端 | `1.26.3` 或更高 |
| [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows-install/) | 运行 PostgreSQL | 启动 WSL2 后端 |
| [Node.js](https://nodejs.org/) | 为后续前端项目准备 | 当前 LTS 版本 |
| [VS Code](https://code.visualstudio.com/) | 编辑代码 | 可选但推荐 |

Node.js 目前只需要安装并确认可用。前端方案没有确定前，不创建 Vue、React 或其他项目，也不执行脚手架初始化命令。

安装完成后打开 PowerShell，检查：

```powershell
git --version
go version
docker version
docker compose version
node --version
npm --version
```

如果某个命令不存在，关闭并重新打开 PowerShell；仍然不存在时，检查对应软件是否安装以及是否加入 Windows `PATH`。

## 三、获取项目代码

以下命令可以在 PowerShell 或 Git Bash 中执行。

进入准备保存代码的目录，然后克隆仓库：

```powershell
cd $HOME\Documents
git clone git@gitee.com:ccairn/cairn.git
cd cairn
```

如果已经克隆过：

```powershell
cd $HOME\Documents\cairn
git fetch origin
git status
```

检查当前分支：

```powershell
git branch --show-current
```

不要直接在 `master` 上开发。按照项目的《开发协作规范》，切换到分配给自己的 `feature/F序号_需求名` 分支。

例如：

```powershell
git switch feature/F1_windows本地部署
```

如果项目保存在其他位置，将示例路径替换为实际路径。

## 四、启动 Docker Desktop

打开 Docker Desktop，等待状态显示 Docker Engine 正常运行。

回到项目根目录：

```powershell
cd $HOME\Documents\cairn
```

为当前 PowerShell 设置本地数据库密码：

```powershell
$env:DB_PASSWORD = "cairn_dev"
```

启动 PostgreSQL：

```powershell
docker compose up -d
docker compose ps
```

正常情况下可以看到 `db` 容器，稍等片刻后状态会变为 `healthy`。

本地数据库配置为：

| 配置 | 值 |
| --- | --- |
| 地址 | `127.0.0.1` |
| 端口 | `5432` |
| 数据库 | `cairn` |
| 用户 | `cairn` |
| 密码 | `cairn_dev` |

这些配置只用于本地开发，不是生产环境凭据。

## 五、安装后端依赖

在项目根目录执行：

```powershell
go mod download
go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

`migrate` 默认安装到 Go 的用户工具目录。当前 PowerShell 如果找不到该命令，可临时加入路径：

```powershell
$env:Path += ";$HOME\go\bin"
migrate -version
```

安装时必须带 `-tags postgres`，否则执行迁移时可能报 `unknown driver postgres`。

## 六、初始化数据库

确认当前 PowerShell 中已经设置密码：

```powershell
$env:DB_PASSWORD = "cairn_dev"
```

执行数据库迁移：

```powershell
migrate -path migrations -database "postgres:/cairn:$env:DB_PASSWORD@localhost:5432/cairn?sslmode=disable" up
```

查看当前迁移版本：

```powershell
migrate -path migrations -database "postgres://cairn:$($env:DB_PASSWORD)@localhost:5432/cairn?sslmode=disable" version
```

迁移只需要在首次创建数据库或仓库新增迁移文件后执行。重复执行 `up` 不会重复创建已经应用的结构。

## 七、启动 Go 后端

在项目根目录执行：

```powershell
$env:DB_PASSWORD = "cairn_dev"
$env:DEV = "true"
go run ./cmd/server
```

看到服务启动日志后，不要关闭该 PowerShell 窗口。

浏览器访问：

```text
http://localhost:8080
```

另开一个 PowerShell，检查后端健康状态：

```powershell
Invoke-RestMethod http://localhost:8080/healthz
```

正常结果包含：

```text
status  version
------  -------
ok      dev
```

也可以使用：

```powershell
curl.exe http://localhost:8080/healthz
```

预期返回：

```json
{"status":"ok","version":"dev"}
```

## 八、修改代码后的处理

当前 Windows 原生环境使用 `go run` 启动后端，没有启用热重载。

修改 Go 代码或服务端模板后：

1. 在运行后端的 PowerShell 按 `Ctrl+C`。
2. 重新执行：

   ```powershell
   $env:DB_PASSWORD = "cairn_dev"
   $env:DEV = "true"
   go run ./cmd/server
   ```

未来前端技术确定后，前端开发服务器通常会提供自己的热更新；具体启动命令会由选定的框架和构建工具决定。

## 九、未来前端项目需要补充的配置

本地后端跑通后，前端同学可以开始评估 Vue、React 或其他方案。技术选型完成前，不在本文预设具体实现。

选型时至少需要共同确认：

- 前端项目放在当前仓库还是独立仓库。
- 前端目录名称和包管理工具。
- 使用 Vue、React 或其他框架。
- 使用 Vite 或其他构建工具。
- 本地前端开发服务器端口。
- Go 后端提供哪些 API。
- 本地跨域使用 CORS 还是开发服务器代理。
- Cookie、登录状态和 CSRF 如何处理。
- 生产构建产物由 Go、Caddy 还是独立静态服务托管。

在这些事项确定后，再增加前端安装和启动章节，例如：

```text
npm ci
npm run dev
npm run build
```

这些只是常见命令示例，现在还不是 Cairn 的正式前端命令。

## 十、每天启动本地环境

1. 打开 Docker Desktop。
2. 打开 PowerShell 并进入项目根目录。
3. 确认当前功能分支。
4. 启动 PostgreSQL。
5. 启动 Go 后端。

```powershell
cd $HOME\Documents\cairn
git status
$env:DB_PASSWORD = "cairn_dev"
docker compose up -d
$env:DEV = "true"
go run ./cmd/server
```

浏览器访问 <http://localhost:8080>。

## 十一、停止本地环境

停止 Go 后端：在运行 `go run` 的 PowerShell 中按 `Ctrl+C`。

停止 PostgreSQL，但保留本地数据：

```powershell
docker compose down
```

完全删除本地 PostgreSQL 数据：

```powershell
docker compose down -v
```

`down -v` 会删除数据库数据卷。只有确定不需要本地数据时才能执行。

本地开发不需要执行 `task deploy` 或 `deploy/deploy.sh`。它们会连接生产服务器，只能由维护者在明确决定发布时使用。

## 十二、常见问题

| 现象 | 原因与处理 |
| --- | --- |
| `fatal: not a git repository` | 当前目录不是仓库。执行 `Get-Location` 和 `Get-ChildItem`，然后进入正确的 `cairn` 目录。 |
| `docker` 命令不存在 | Docker Desktop 未安装、未启动或终端没有重新打开。 |
| `DB_PASSWORD is required` | 当前 PowerShell 未设置密码，执行 `$env:DB_PASSWORD = "cairn_dev"`。 |
| 数据库连接失败 | 执行 `docker compose ps`，确认 `db` 为 `healthy`。 |
| `migrate` 命令不存在 | 将 `$HOME\go\bin` 加入当前 PowerShell 的 `PATH`。 |
| `unknown driver postgres` | 使用 `-tags postgres` 重新安装 migrate。 |
| 8080 端口被占用 | 关闭占用端口的程序，或与后端协商修改本地端口。 |
| 5432 端口被占用 | 停止本机其他 PostgreSQL，或调整 Docker Compose 端口。 |
| `/healthz` 正常但页面异常 | 后端正在运行，继续检查浏览器控制台、接口响应或页面代码。 |
| 修改代码后没有变化 | 当前使用 `go run`，停止并重新启动后端。 |

## 十三、本地部署完成标准

满足以下条件，就说明 Windows 本地基础环境已经搭建完成：

- [ ] Git、Go、Docker、Node.js 和 npm 命令正常。
- [ ] PostgreSQL 容器状态为 `healthy`。
- [ ] 数据库迁移成功。
- [ ] Go 后端可以启动。
- [ ] `http://localhost:8080/healthz` 返回 `ok`。
- [ ] 浏览器能打开 `http://localhost:8080`。
- [ ] 当前在自己的功能分支，不是直接在 `master` 开发。

完成这些基础步骤后，就可以在不影响生产环境的前提下进行前端技术选型和接口联调。

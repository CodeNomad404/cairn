# Windows 本地开发环境

Cairn 的前端、后端、PostgreSQL 和数据库迁移均在 Docker 中运行。Windows 开发者不需要在宿主机单独安装 Go、Node.js、pnpm、Air、PostgreSQL 或 migrate。

## 1. 安装软件

安装：

- Git for Windows；
- Docker Desktop，并启用 WSL 2 后端；
- VS Code 或其他编辑器。

打开 PowerShell 检查：

```powershell
git --version
docker version
docker compose version
```

## 2. 获取代码

```powershell
cd $HOME\Documents
git clone git@gitee.com:ccairn/cairn.git
cd cairn
```

不要直接在 `master` 开发。按照《开发协作规范》切换到对应功能分支。

## 3. 启动完整环境

启动 Docker Desktop，等待 Docker Engine 就绪，然后在仓库根目录执行：

```powershell
.\dev.ps1
```

也可以直接执行：

```powershell
docker compose up --build
```

首次启动需要下载并构建镜像，之后会复用缓存。Compose 会依次启动数据库、执行迁移、启动后端并启动前端。

`dev.ps1` 会优先检查本机镜像；缺少镜像时，从 DaoCloud 镜像源拉取并添加官方标签，避免直接访问 Docker Hub 超时。Dockerfile 和 Compose 中仍使用官方镜像名。

打开：

```text
前端：http://localhost:5173
后端健康检查：http://localhost:8080/healthz
```

正常健康响应：

```json
{"status":"ok","version":"dev"}
```

## 4. 日常开发

- 修改 `web/src` 后，Vite 自动刷新页面。
- 修改 `cmd` 或 `internal` 下的 Go 文件后，Air 自动重新构建后端。
- 新增迁移文件后，重新执行 `docker compose up`，迁移容器会应用尚未执行的迁移。
- 前端通过 `/api` 代理访问后端，不需要配置 CORS。

查看所有容器：

```powershell
docker compose ps
```

查看日志：

```powershell
docker compose logs -f
```

只看某个服务：

```powershell
docker compose logs -f frontend
docker compose logs -f backend
docker compose logs -f db
```

## 5. 停止与重置

停止环境并保留数据库：

```powershell
.\stop.ps1
```

或者：

```powershell
docker compose down
```

完全删除本地数据库：

```powershell
docker compose down -v
```

`down -v` 会永久删除本地数据库 Volume，只有确定不再需要本地数据时才能执行。

## 6. 常见问题

| 现象 | 处理 |
| --- | --- |
| `docker` 命令不存在 | 安装 Docker Desktop，重新打开 PowerShell。 |
| 无法连接 Docker Engine | 启动 Docker Desktop，等待 WSL 2 后端就绪。 |
| 5173、8080 或 5432 端口被占用 | 停止占用端口的软件，再重新启动 Compose。 |
| 镜像下载失败 | 脚本默认使用 DaoCloud 镜像源；确认网络可访问该镜像源，必要时再配置 Docker Desktop 代理。 |
| 后端启动失败 | 执行 `docker compose logs backend migrate db`。 |
| 修改代码后没有生效 | 检查 `docker compose ps`，然后查看对应容器日志。 |
| 想清空测试数据 | 执行 `docker compose down -v` 后重新启动。 |

## 7. 完成标准

- Docker Compose 中 `db` 和 `backend` 状态为 healthy；
- `migrate` 容器以状态码 0 结束；
- `http://localhost:5173` 可以打开；
- `http://localhost:8080/healthz` 返回 `ok`；
- 修改前端和后端代码时热更新正常；
- 当前位于自己的功能分支，而不是直接修改 `master`。

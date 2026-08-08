# Cairn

Cairn 是一个使用 Vue 3、Go 和 PostgreSQL 构建的个人博客、作品集与工具站。

## 本地开发

需要 Git、Docker 和 Docker Compose：

```bash
docker compose up --build
```

打开：

- 前端：<http://localhost:5173>
- 后端健康检查：<http://localhost:8080/healthz>

停止并保留本地数据库：

```bash
docker compose down
```

详细说明：

- [开发协作规范](docs/开发协作规范.md)
- [Windows 本地开发环境](docs/Windows本地开发环境.md)
- [前端架构设计](docs/前端架构设计.md)
- [生产部署](deploy/README.md)

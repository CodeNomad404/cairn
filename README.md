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

## TODO

- [ ] 在服务器上部署 Prowlarr 搜索 4K 片源，在电脑上运行 qBittorrent 下载和做种。通过浏览器搜索 `英文片名 年份 2160p`，将磁力链接或 `.torrent` 文件交给本机 qBittorrent。电影文件和 BT 传输都留在电脑端，节省服务器磁盘空间和带宽。

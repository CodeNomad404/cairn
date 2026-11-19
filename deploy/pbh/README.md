# Cairn 的 PeerBanHelper 部署模块

PBH 在 Cairn 服务器上作为独立的 Compose 项目运行。管理界面使用 PBH 自带的 WebUI，通过 SSH 隧道访问。服务数据保存在 `cairn-pbh_pbh-data` Docker 卷中，启停 PBH 不需要重启 Cairn 或 PostgreSQL。

镜像使用 [PBH v9.5.4 发布页](https://github.com/PBH-BTN/PeerBanHelper/releases/tag/v9.5.4)列出的国内镜像。要阅读或修改对应版本的源码，在 Cairn 根目录执行：

```bash
git clone --depth 1 --branch v9.5.4 https://github.com/PBH-BTN/PeerBanHelper.git tmp/PeerBanHelper
```

推送 `master` 后，Gitee Go 在 Cairn 健康检查通过之后启动 PBH，再验证两个服务。PBH 启动失败会使流水线报错，已通过检查的 Cairn 版本继续运行。在服务器上也可以单独部署或检查 PBH：

```bash
cd /opt/cairn-repo/deploy/pbh
sudo docker compose config --quiet
sudo docker compose up -d --wait
sudo docker compose ps
curl --fail http://127.0.0.1:9898/
curl --fail http://127.0.0.1:8080/healthz
```

在自己的电脑上运行 `ssh -N -L 127.0.0.1:9898:127.0.0.1:9898 <SSH 主机别名>`，然后打开 `http://127.0.0.1:9898/`，按向导设置 PBH 的访问令牌并添加下载器。Compose 使用桥接网络：容器内的 `127.0.0.1` 指 PBH 自己，不能用作宿主机下载器的地址。下载器还需要访问 PBH 封禁列表时，按[官方 HTTP 配置](https://docs.pbh-btn.com/docs/network/http-server/)确认下载器可达的地址，再决定如何接入。

回滚时运行 `sudo docker compose down`，保留数据卷。

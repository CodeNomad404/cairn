# Cairn 前端协作与 Git 工作流

这份说明面向参与 Cairn 前端开发的同学。仓库地址：

```text
https://gitee.com/ccairn/cairn
```

## 一句话规则

`master` 是唯一主线和可部署分支；不要直接在 `master` 上开发或推送。

每个需求从最新的 `master` 拉出一个 `feature/...` 分支。完成后通过 Gitee 合并请求合入 `master`，由仓库维护者审核并合并。

```text
master ──┬───────────────────────────  线上部署只读取这里
         └── feature/homepage-layout ─  前端需求在这里完成
```

仓库中即使能看到 `develop`、`feature`、`release` 等旧分支，也不要基于它们开发；它们不是当前工作流的一部分。

## 第一次同步仓库

在 Git Bash 中先进入克隆出来的项目目录，而不是停留在 `~` 目录：

```bash
cd ~/cairn
git status
```

如果项目克隆在桌面，常见路径是：

```bash
cd ~/Desktop/cairn
```

同步主线：

```bash
git fetch origin
git switch master
git pull --ff-only origin master
```

本仓库迁移期间，`master` 曾被替换为完整项目历史。如果你很早就克隆了仓库，并且 `git pull --ff-only` 提示没有共同历史，先确认没有自己的未提交工作，再执行：

```bash
git fetch origin
git switch master
git reset --hard origin/master
```

`reset --hard` 会丢弃当前目录未提交的改动；有任何本地工作时先停止并联系维护者。

## 开始一个前端需求

每次开工都先同步 `master`，再创建功能分支。分支名用英文小写和连字符，描述正在做的事：

```bash
git switch master
git pull --ff-only origin master
git switch -c feature/homepage-layout
```

前端通常修改这些目录：

```text
web/templates/       Go HTML 模板
web/static/css/      Tailwind 源样式与生成后的 CSS
web/static/js/       页面脚本
web/static/images/   图片与图标资源
```

项目是 Go 服务端渲染，不使用 React、Vue 或 Next.js。需要新增页面路由、接口、数据库结构或改动 Go 逻辑时，先和维护者确认接口和边界。

## Tailwind CSS

首次安装前端依赖：

```bash
npm install
```

开发时持续监听并生成 CSS：

```bash
npm run watch:css
```

样式完成后，确认生成文件已更新：

```bash
git status
```

`web/static/css/output.css` 已纳入 Git；修改 Tailwind 类名或 `input.css` 后，若它变化，需要一并提交。

## 提交与推送

提交应小而聚焦，一次只解决一个需求或一个明确问题：

```bash
git status
git add web/templates/home.html web/static/css/input.css web/static/css/output.css
git commit -m "完善首页首屏布局"
git push -u origin feature/homepage-layout
```

不要提交 `.env`、密码、私钥、部署配置或本机临时文件。提交前也不要使用 `git add .` 作为习惯动作，先检查将要提交的文件。

## 合并请求与发布

推送分支后，在 Gitee 创建合并请求：

- 源分支：你的 `feature/...` 分支
- 目标分支：`master`
- 说明：做了什么、如何预览、是否有待确认的交互或视觉细节

维护者审核并合并后，功能才进入主线。合并到 `master` 不等于立刻自动上线：目前生产发布由维护者执行 `task deploy` 触发，服务器只从 Gitee 的 `master` 拉取代码、测试、构建并重启服务。

## 常用自检

```bash
# 看当前分支和是否有未提交改动
git status

# 看当前分支相对远端的提交
git log --oneline --decorate -5

# 拉取远端分支信息，不修改工作区文件
git fetch origin
```

遇到冲突、无法推送、分支不确定，或需要改动 Go/数据库/部署文件时，先不要强制推送或重置，联系维护者一起处理。

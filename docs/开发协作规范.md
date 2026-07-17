# Cairn Git 架构与开发流程

仓库地址：

```text
https://gitee.com/ccairn/cairn
```

## Git 架构

`master` 是唯一主线，也是唯一可部署分支。它始终保存经过审核、可以发布的代码。

```text
master ──┬───────────────────────────  主线；生产部署只读取这里
         ├── feature/F001_首页布局     首页改版
         └── feature/F002_文章卡片     文章卡片优化
```

- 不直接在 `master` 上开发或推送。
- 每个需求从最新的 `master` 创建一个 `feature/F序号_需求名` 分支。
- 功能完成后，通过 Gitee 合并请求把功能分支合入 `master`。
- 只有仓库维护者合并 `master`；合并后才具备部署资格。
- 不使用 `develop`、`release` 或仓库中其他旧分支作为开发起点。

## 日常开发流程

### 1. 同步主线

每次开始新需求前，先进入项目目录并同步远端 `master`：

```bash
git switch master
git pull --ff-only origin master
```

### 2. 创建功能分支

分支名格式固定为 `feature/F序号_需求名`。序号按需求递增，需求名使用简短中文描述：

```bash
git switch -c feature/F001_首页布局
```

一个分支只做一个明确需求。不要在同一个分支混入不相关的页面、重构或修复。

### 3. 开发并提交

完成一个可说明的小步骤后提交。提交前先检查改动范围：

```bash
git status
git add <实际修改的文件>
git commit -m "完善首页首屏布局"
```

不要直接使用 `git add .`，也不要提交密码、私钥、部署配置或本机临时文件。

### 4. 推送分支并创建合并请求

```bash
git push -u origin feature/F001_首页布局
```

随后在 Gitee 创建合并请求：

- 源分支：`feature/F001_首页布局`
- 目标分支：`master`
- 描述：说明做了什么、如何验证，以及需要确认的事项。

### 5. 审核、合并与部署

维护者审核合并请求；需要修改时，继续在原功能分支提交并推送，合并请求会自动更新。

审核通过后，维护者将其合入 `master`。当前生产发布由维护者手动触发，服务器只从 Gitee 的 `master` 拉取代码。

## 已克隆仓库但未同步时

如果本地没有自己的改动，执行：

```bash
git fetch origin
git switch master
git pull --ff-only origin master
```

若提示历史无关，说明仓库是在迁移前克隆的。确认没有未提交工作后，联系维护者处理；不要自行强制推送、强制拉取或删除分支。

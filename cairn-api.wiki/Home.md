# Cairn API 接口文档

Cairn（cairn.cn）后端接口文档，供前端开发使用。

## Base URL

- 本地开发：`http://localhost:8080`
- 生产环境：`https://cairn.cn`
- 前端开发环境经 Vite 代理 `/api`，页面内一律使用相对路径 `/api/...`

## 目录

| 文档 | 内容 |
|---|---|
| [api/schema.md](api/schema.md) | 通用约定：HOST、请求格式、响应格式、鉴权 |
| [api/auth/](api/auth/) | 认证：注册 / 登录 / 登出 / 当前用户 |
| [api/blog/post/](api/blog/post/) | 博客·文章：创建 / 列表 / 详情 / 删除 |
| [api/blog/comment/](api/blog/comment/) | 博客·评论：创建 / 列表 / 删除 |
| [api/tool/](api/tool/) | 工具：JSON 格式化、教务系统登录与抢课 |
| [api/public/](api/public/) | 公共端点：健康检查、robots.txt、sitemap.xml |

## 登录

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| POST | /api/v1/auth/register | 公开 | 注册 |
| POST | /api/v1/auth/login | 公开 | 登录 |
| POST | /api/v1/auth/logout | 公开 | 登出 |
| GET | /api/v1/auth/me | 登录 | 当前用户信息 |

## 博客
| POST | /api/v1/posts | admin | 创建文章 |
| GET | /api/v1/posts | 公开 | 文章列表 |
| GET | /api/v1/posts/{slug} | 公开 | 文章详情 |
| DELETE | /api/v1/posts/{slug} | admin | 删除文章 |
| POST | /api/v1/posts/{slug}/comments | 登录 | 发表评论 |
| GET | /api/v1/posts/{slug}/comments | 公开 | 评论列表 |
| DELETE | /api/v1/comments/{id} | 登录 | 删除评论 |

> 各接口的请求参数、成功响应与预期失败详见对应文档。

## 教务系统与抢课

| 文档 | 内容 |
|---|---|
| [教务系统登录](api/tool/course_registration_login.md) | 登录 BJTU、保存凭据及登录次数 |
| [本方案课程列表](api/tool/course_registration_courses.md) | 查询完整课程名称、教师、余量和已选状态 |
| [新增抢课课程](api/tool/course_registration_grab.md) | 同步判断课程状态，必要时加入后台队列 |
| [抢课任务](api/tool/course_registration_tasks.md) | 查询任务以及按课程暂停、恢复 |
| [公共 MIS 绑定](api/tool/mis_account_bind.md) | 验证并保存当前用户的 MIS 账号，不触发抢课 |
| [课程作业](api/tool/course_homework.md) | 当前学期作业快照、手动同步、正文及附件名称 |

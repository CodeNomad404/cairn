# 抢课

## 接口

```
POST /api/v1/tools/course-registration/grab
```

需要先登录 Cairn，再调用教务系统登录接口建立会话。本接口只处理本方案课程。课程必须使用课程列表接口返回的完整 `name` 精确匹配；已选中、抢课成功或未找到时直接返回，只有课程存在但暂无余量时才写入后台抢课队列。

## 请求示例

```json
{
  "courses": [
    "M302006B:计算机体系结构 01 计算机学院"
  ]
}
```

`courses` 必须包含 1–50 个完整课程名称。匹配时只规范连续空格，不按课程号或课程名模糊匹配。

| 参数 | 类型 | 必填 | 约束 |
|---|---|---|---|
| `courses` | `string[]` | 是 | 1–50 项；每项不超过 256 个字符 |

## 成功响应

```json
{
  "code": 0,
  "data": {
    "results": [
      {
        "target": "M302006B:计算机体系结构 01 计算机学院",
        "status": "succeeded",
        "course": {
          "id": "<upstream-course-id>",
          "name": "M302006B:计算机体系结构 01 计算机学院",
          "teacher": "田艺",
          "remaining": 3,
          "selected": false
        },
        "selected": true,
        "message": "选课成功"
      }
    ]
  }
}
```

无余量时，对应结果包含数据库任务的 `task_id`，状态为 `grabbing`。同一学生的全部 `grabbing` 课程由一个 Worker 共用教务会话和课程列表处理。

无余量并创建任务的响应项示例：

```json
{
  "target": "M302006B:计算机体系结构 04 计算机学院",
  "task_id": 18,
  "status": "grabbing",
  "course": {
    "id": "<upstream-course-id>",
    "name": "M302006B:计算机体系结构 04 计算机学院",
    "teacher": "李夏青",
    "remaining": 0,
    "selected": false
  },
  "selected": false,
  "message": "该课程暂无余量，已加入抢课任务"
}
```

响应项状态：

| 状态 | 是否创建任务 | 说明 |
|---|---|---|
| `succeeded` | 否 | 当前请求已经抢到课程 |
| `already_selected` | 否 | 查询时课程已经选中 |
| `not_found` | 否 | 没有精确匹配到完整课程名称 |
| `ambiguous` | 否 | 完整名称意外匹配到多条课程，后端不会选择第一条 |
| `grabbing` | 是 | 课程存在但暂无余量，已新增或复用数据库任务 |
| `paused` | 复用已有任务 | 同一课程已有暂停任务，新增接口不会自动恢复 |
| `unconfirmed` | 否 | 已提交选课，但重新查询后仍无法确认最终状态 |

HTTP `200` 表示所有目标均已完成同步判断，不代表每门课程都已抢到。应分别读取每个结果的 `status` 和 `selected`。

## 相关接口

```
GET /api/v1/tools/course-registration/grab-tasks
```

返回当前用户的学生级 Worker 状态、登录次数和全部课程任务。

## 暂停或恢复一门课程

```
PATCH /api/v1/tools/course-registration/grab-tasks/{taskID}
```

请求体为 `{"status":"paused"}` 或 `{"status":"grabbing"}`。控制粒度是单门课程，只允许任务所属的 Cairn 用户执行。

详细字段和响应示例见 [course_registration_tasks.md](course_registration_tasks.md)。

## 失败响应

请求中没有课程：

```json
{
  "code": 400,
  "message": "courses is required"
}
```

未建立或已经失去 BJTU 会话时返回 `401`。验证码识别失败时返回 `422`。

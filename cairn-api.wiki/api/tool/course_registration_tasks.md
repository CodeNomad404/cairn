# 抢课任务

抢课任务按课程分别保存和控制；同一学生的所有运行中课程由一个后台 Worker 共用教务会话和课程列表处理。后台循环间隔当前为 2 秒。

## 查询任务列表

```
GET /api/v1/tools/course-registration/grab-tasks
```

只返回当前 Cairn 用户自己的任务。

### 成功响应

```json
{
  "code": 0,
  "data": {
    "worker_status": "running",
    "login_count": 2,
    "last_login_at": "2026-09-07T12:00:00+08:00",
    "items": [
      {
        "id": 18,
        "course_name": "M302006B:计算机体系结构 04 计算机学院",
        "upstream_course_id": "<upstream-course-id>",
        "teacher": "李夏青",
        "remaining": 0,
        "status": "grabbing",
        "message": "该课程暂无余量，正在抢课",
        "attempt_count": 35,
        "last_checked_at": "2026-09-07T12:30:00+08:00",
        "created_at": "2026-09-07T12:28:50+08:00",
        "updated_at": "2026-09-07T12:30:00+08:00"
      }
    ]
  }
}
```

学生级 Worker 状态：

| 状态 | 说明 |
|---|---|
| `idle` | 没有正在运行的课程任务，不访问教务系统 |
| `running` | 至少有一门课程正在抢 |
| `auth_failed` | 会话失效后的自动登录失败，需要用户重新登录 |

课程任务状态：

| 状态 | 说明 |
|---|---|
| `grabbing` | 正在检查余量并抢课 |
| `paused` | 用户暂停了这门课程 |
| `succeeded` | 已抢到或确认已经选中 |
| `not_found` | 后台检查时不再能找到该完整课程名称 |

`login_count` 是该学生实际发起 BJTU 登录的累计次数，包括用户主动登录和 Worker 在会话失效后的自动登录。

## 暂停或恢复一门课程

```
PATCH /api/v1/tools/course-registration/grab-tasks/{taskID}
```

暂停请求：

```json
{
  "status": "paused"
}
```

恢复请求：

```json
{
  "status": "grabbing"
}
```

只能修改当前用户自己的 `grabbing` 或 `paused` 任务。`succeeded` 和 `not_found` 是终态，不能暂停或恢复。

### 成功响应

```json
{
  "code": 0,
  "data": {
    "id": 18,
    "course_name": "M302006B:计算机体系结构 04 计算机学院",
    "teacher": "李夏青",
    "remaining": 0,
    "status": "paused",
    "message": "抢课任务已暂停",
    "attempt_count": 35,
    "created_at": "2026-09-07T12:28:50+08:00",
    "updated_at": "2026-09-07T12:31:00+08:00"
  }
}
```

### 失败响应

任务 ID 无效：

```json
{
  "code": 400,
  "message": "抢课任务 ID 无效"
}
```

任务不存在、不属于当前用户或已经处于终态：

```json
{
  "code": 404,
  "message": "未找到可修改的抢课任务"
}
```

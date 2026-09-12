# 本方案课程列表

## 接口

```
GET /api/v1/tools/course-registration/courses
```

需要先登录 Cairn，并通过教务系统登录接口建立 BJTU 会话。当前版本只返回本方案课程，不接收 `type` 或 `course_type` 参数。

## 成功响应

HTTP `200 OK`

```json
{
  "code": 0,
  "data": {
    "selection_open": true,
    "courses": [
      {
        "name": "M302006B:计算机体系结构 01 计算机学院",
        "teacher": "田艺",
        "remaining": 0,
        "selected": true
      },
      {
        "id": "<upstream-course-id>",
        "name": "M302006B:计算机体系结构 04 计算机学院",
        "teacher": "李夏青",
        "remaining": 2,
        "selected": false
      }
    ]
  }
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `selection_open` | `boolean` | 当前网上选课阶段是否开放；关闭时 `courses` 为空数组 |
| `id` | `string` | 教务系统内部课程 ID；已选课程或不可提交的课程可能没有该字段 |
| `name` | `string` | 完整课程名称，是新增抢课课程时使用的精确标识 |
| `teacher` | `string` | 任课教师 |
| `remaining` | `integer` | 当前余量 |
| `selected` | `boolean` | 当前学生是否已经选中 |

前端应让用户从该列表中选择课程，并将完整 `name` 原样提交给抢课新增接口。不要只提交课程号或课程名称。

当前选课阶段关闭时仍返回 `200 OK`，用于区分“尚未开放”和“登录失败”：

```json
{
  "code": 0,
  "data": {
    "selection_open": false,
    "courses": []
  }
}
```

## 失败响应

未建立 BJTU 会话：

```json
{
  "code": 401,
  "message": "BJTU login session not found"
}
```

教务会话已经失效：

```json
{
  "code": 401,
  "message": "BJTU academic system session expired"
}
```

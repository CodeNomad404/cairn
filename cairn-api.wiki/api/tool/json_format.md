# JSON 格式化

## 接口

```
POST /api/v1/tools/json/format
```

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 默认值 | 约束 | 示例 |
|---|---|---|---|---|---|---|
| `input` | Body | `string` | 是 | — | 合法的 JSON 字符串，大小不超过 1 MB | `{"name":"cairn","n":1}` |

## 请求示例

```json
{
  "input": "{\"name\":\"cairn\",\"n\":1}"
}
```

## 成功响应

HTTP `200 OK`

```json
{
  "code": 0,
  "data": {
    "formatted": "{\n  \"n\": 1,\n  \"name\": \"cairn\"\n}"
  }
}
```

`formatted` 为缩进两个空格的格式化 JSON 字符串，键按字典序排列，可直接在前端以等宽字体展示。

## 失败响应

`input` 不是合法 JSON：

HTTP `400 Bad Request`

```json
{
  "code": 400,
  "message": "invalid JSON: invalid character 'b' looking for beginning of object key string"
}
```

`message` 中的具体报错随输入变化，前端可直接展示。

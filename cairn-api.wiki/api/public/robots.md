# 爬虫协议

## 接口

```
GET /robots.txt
```

## 请求参数

无。

## 成功响应

```text
User-agent: *
Disallow: /upload
Disallow: /healthz

Sitemap: https://cairn.cn/sitemap.xml
```

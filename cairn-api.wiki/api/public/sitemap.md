# 站点地图

## 接口

```
GET /sitemap.xml
```

## 请求参数

无。

## 成功响应

HTTP `200 OK`，`Content-Type: application/xml`。除固定页面外，每篇已发布的文章各占一个 `<url>`，`<lastmod>` 为文章发布时间。

```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://cairn.cn/</loc>
  </url>
  <url>
    <loc>https://cairn.cn/posts</loc>
  </url>
  <url>
    <loc>https://cairn.cn/tools</loc>
  </url>
  <url>
    <loc>https://cairn.cn/tools/json</loc>
  </url>
  <url>
    <loc>https://cairn.cn/tools/diff</loc>
  </url>
  <url>
    <loc>https://cairn.cn/posts/blog-project-diff-2026-05-31</loc>
    <lastmod>2026-06-08T15:11:16Z</lastmod>
  </url>
  <url>
    <loc>https://cairn.cn/posts/blog-project-diff-2026-05-27</loc>
    <lastmod>2026-06-08T15:10:51Z</lastmod>
  </url>
  <url>
    <loc>https://cairn.cn/posts/day-one-server-ready</loc>
    <lastmod>2026-06-06T14:23:33Z</lastmod>
  </url>
</urlset>
```

软删除的文章不会出现在站点地图中。

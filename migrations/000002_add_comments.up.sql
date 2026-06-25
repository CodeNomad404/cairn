COMMENT ON COLUMN posts.slug         IS '网址标识，文章的唯一英文别名，如 my-first-post';
COMMENT ON COLUMN posts.title        IS '文章标题';
COMMENT ON COLUMN posts.content_md   IS 'Markdown 原文';
COMMENT ON COLUMN posts.content_html IS 'Markdown 转换后的 HTML';
COMMENT ON COLUMN posts.status       IS '状态：draft（草稿）或 published（已发布）';
COMMENT ON COLUMN posts.created_at   IS '创建时间（UTC）';
COMMENT ON COLUMN posts.updated_at   IS '最近修改时间（UTC）';
COMMENT ON COLUMN posts.published_at IS '发布时间（UTC），草稿为空';

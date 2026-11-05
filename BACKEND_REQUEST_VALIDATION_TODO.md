# Cairn JSON 请求标签校验待办

## 目标

继续使用 chi。在 `internal/httpx.ParseBody` 集中执行结构体标签校验。以后新写 JSON 接口时，必填、数值范围和长度等常规规则可以直接写在请求 DTO 的 `validate` 标签里。已有接口逐步迁移。

## 实施清单

- [ ] 引入 `github.com/go-playground/validator/v10`，在 `httpx` 复用一个校验器实例。默认使用 `validate:"..."` 标签。
- [ ] 将 `ParseBody[T Validatable]` 改为 `ParseBody[T any]`，约定 `T` 是请求 DTO 结构体。严格解码成功后统一执行标签校验，再通过对 `&req` 的接口断言按需调用 `Validate() error`，兼容现有的值接收者和指针接收者。标签校验通过后才执行自定义校验。
- [ ] 保留现有请求体大小上限、未知字段检查、单个 JSON 值检查和鉴权顺序。校验失败沿用 `AppError` 和现有的 HTTP 400 响应信封。对外错误使用 JSON 字段名和可读文案，迁移接口时保留现有公开文案。
- [ ] 先迁移 `POST /api/v1/tool/accounting-expense/create`。用标签声明正数的 `book_id`、正数的 `amount_cents`、合法的 `category` 和必填的 `spent_at`。新增支出 DTO 移除手写的 `Validate()`。备注去首尾空白后的长度校验复用现有 Logic，账本归属仍由现有业务和数据访问流程校验。
- [ ] 后续按接口迁移。只有标签难以表达的请求规则才保留 DTO 的 `Validate()`。同一条输入规则在标签和自定义方法之间选一个位置。

## 验收清单

- [ ] 只声明标签、没有 `Validate()` 的 DTO 可以成功解析，有效请求能进入 Handler 的业务流程，无效字段返回 HTTP 400。
- [ ] 旧 DTO 的自定义校验仍会运行，值接收者和指针接收者均有覆盖。已有接口保持原有响应形状和错误文案。
- [ ] 回归验证新增支出的非法金额、分类、缺失时间和备注超长，以及请求过大、未知字段、尾随 JSON 和未登录场景。
- [ ] 从仓库根目录运行 `bash deploy/ci.sh backend`。

参考 [validator 文档](https://pkg.go.dev/github.com/go-playground/validator/v10)和 [Gin 的绑定与校验文档](https://gin-gonic.com/en/docs/binding/binding-and-validation/)。

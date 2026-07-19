// 统一请求工具，自动走vite代理，不用写localhost:8080
export function request(url, options = {}) {
  // 基础配置：默认POST带json请求头
  const baseOpt = {
    headers: {
      'Content-Type': 'application/json'
    },
    ...options
  }
  return fetch(url, baseOpt).then(res => {
    // 401未登录自动跳登录页
    if (res.status === 401) {
      location.href = '/login'
    }
    return res.json()
  })
}
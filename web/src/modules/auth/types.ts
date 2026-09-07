export type LoginRequest = {
  username: string
  password: string
}

export type RegisterRequest = {
  username: string
  password: string
}

export type UserResult = {
  id: number
  username: string
  role: 'admin' | 'user'
  created_at: string
}

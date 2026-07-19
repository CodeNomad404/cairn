export type Credentials = {
  username: string
  password: string
}

export type User = {
  id: number
  username: string
  role: 'admin' | 'user'
  created_at: string
}

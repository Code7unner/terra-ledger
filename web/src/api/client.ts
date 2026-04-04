const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:3000'
const API_KEY = import.meta.env.VITE_API_KEY || ''

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  if (API_KEY) {
    headers['X-API-Key'] = API_KEY
  }

  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })

  const data = await res.json()

  if (!res.ok) {
    throw new Error(data.error || `Request failed: ${res.status}`)
  }

  return data as T
}

export function get<T>(path: string): Promise<T> {
  return request<T>('GET', path)
}

export function post<T>(path: string, body: unknown): Promise<T> {
  return request<T>('POST', path, body)
}

export function put<T>(path: string, body: unknown): Promise<T> {
  return request<T>('PUT', path, body)
}

export function del<T>(path: string): Promise<T> {
  return request<T>('DELETE', path)
}

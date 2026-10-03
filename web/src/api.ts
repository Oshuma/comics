export interface User {
  id: number
  email: string
  admin: boolean
  createdAt: string
  groupCount: number
  comicCount: number
}

export interface Group {
  id: number
  name: string
  comicCount: number
  coverPageId: number | null
  read: boolean
  reading: boolean
  diskSize: number
}

export interface Comic {
  id: number
  filename: string
  name: string
  groupId: number
  pageCount: number
  coverPageId: number | null
  read: boolean
  reading: boolean
}

export interface Page {
  id: number
  number: number
  read: boolean
}

export interface ComicDetail extends Comic {
  groupName: string
  pages: Page[]
}

export interface PageView extends Page {
  comicId: number
  comicName: string
  groupId: number
  groupName: string
  pageCount: number
  prevPageId: number | null
  nextPageId: number | null
}

export interface History {
  id: number
  groupName: string
  comicName: string
  createdAt: string
  groupId: number | null
  comicId: number | null
}

export interface Paginated<T> {
  items: T[]
  total: number
  page: number
  perPage: number
}

export interface Status {
  setupRequired: boolean
  user: User | null
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

// Called when the API reports the session is gone.
let onUnauthorized: () => void = () => {}
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn
}

// The server rejects state-changing requests without this header (CSRF guard).
export const csrfHeaders = { 'X-Requested-With': 'XMLHttpRequest' }

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`/api${path}`, {
    method,
    credentials: 'same-origin',
    headers: {
      ...csrfHeaders,
      ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    if (res.status === 401 && !path.startsWith('/session')) onUnauthorized()
    throw new ApiError(res.status, data.error || res.statusText)
  }
  return data as T
}

const get = <T>(path: string) => request<T>('GET', path)
const post = <T>(path: string, body?: unknown) => request<T>('POST', path, body ?? {})
const put = <T>(path: string, body?: unknown) => request<T>('PUT', path, body ?? {})
const patch = <T>(path: string, body?: unknown) => request<T>('PATCH', path, body ?? {})
const del = <T>(path: string) => request<T>('DELETE', path)

type OK = { ok: true }

export interface NewUser {
  email: string
  password: string
  passwordConfirmation: string
  admin?: boolean
}

export const api = {
  status: () => get<Status>('/status'),
  setup: (user: NewUser) => post<User>('/setup', user),
  login: (email: string, password: string, remember: boolean) =>
    post<User>('/session', { email, password, remember }),
  logout: () => del<OK>('/session'),

  groups: () => get<Group[]>('/groups'),
  group: (id: number) => get<{ group: Group; comics: Comic[] }>(`/groups/${id}`),
  createGroup: (name: string) => post<Group>('/groups', { name }),
  renameGroup: (id: number, name: string) => patch<OK>(`/groups/${id}`, { name }),
  deleteGroup: (id: number) => del<OK>(`/groups/${id}`),
  deleteGroupComics: (id: number) => del<OK>(`/groups/${id}/comics`),
  deleteGroupReadComics: (id: number) => del<OK>(`/groups/${id}/comics/read`),

  comic: (id: number) => get<ComicDetail>(`/comics/${id}`),
  deleteComic: (id: number) => del<OK>(`/comics/${id}`),
  deleteReadComics: () => del<OK>('/comics/read'),
  moveComic: (id: number, groupId: number) => put<OK>(`/comics/${id}/group`, { groupId }),
  finishComic: (id: number) =>
    put<{ groupId: number; nextComicId: number | null }>(`/comics/${id}/finish`),
  resumeComic: (id: number) => get<{ pageId: number | null }>(`/comics/${id}/resume`),
  viewPage: (comicId: number, pageId: number) =>
    put<PageView>(`/comics/${comicId}/pages/${pageId}/current`),

  history: (page: number) => get<Paginated<History>>(`/history?page=${page}`),
  stats: () => get<{ totalSize: number; groups: Group[] }>('/stats'),

  users: (page: number) => get<Paginated<User>>(`/admin/users?page=${page}`),
  createUser: (user: NewUser) => post<User>('/admin/users', user),
  setAdmin: (id: number, admin: boolean) => put<OK>(`/admin/users/${id}/admin`, { admin }),
  deleteUser: (id: number) => del<OK>(`/admin/users/${id}`),
}

export const pageImageUrl = (pageId: number) => `/api/pages/${pageId}/image`
export const pageThumbUrl = (pageId: number) => `/api/pages/${pageId}/thumb`

// Where "read this comic" should go: first unread page, or the page list.
export async function resumePath(comicId: number): Promise<string> {
  const { pageId } = await api.resumeComic(comicId)
  return pageId ? `/comics/${comicId}/pages/${pageId}` : `/comics/${comicId}`
}

export function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

// Mirrors Rails' number_to_human_size.
export function humanSize(bytes: number): string {
  const units = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let n = bytes
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024
    i++
  }
  if (i === 0) return `${n} ${n === 1 ? 'Byte' : 'Bytes'}`
  return `${parseFloat(n.toPrecision(3))} ${units[i]}`
}

export function pluralize(count: number, word: string): string {
  return `${count} ${count === 1 ? word : word + 's'}`
}

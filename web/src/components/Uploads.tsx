import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import { csrfHeaders } from '../api'
import { useAuth } from '../context'
import { useImports } from './Imports'

const PARALLEL_UPLOADS = 3

// Uploaded files leave this list and are tracked as import jobs.
export type UploadStatus = 'added' | 'queued' | 'uploading' | 'error'

export interface Upload {
  key: number
  file: File
  status: UploadStatus
  loaded: number
  error?: string
  // Set when started: "id:<n>" for an existing group or "name:<s>" to
  // find or create one by name, and the group's name for display.
  target?: string
  groupName?: string
}

export const isUploading = (u: Upload) => u.status === 'queued' || u.status === 'uploading'

interface UploadsState {
  uploads: Upload[]
  // Increases each time an upload is handed off to the server.
  uploadedCount: number
  addFiles: (files: FileList | File[]) => void
  start: (target: string, groupName: string, keys?: number[]) => void
  cancel: (key: number) => void
  cancelAll: () => void
}

const UploadsContext = createContext<UploadsState | null>(null)

let nextKey = 1

function groupField(target: string): [string, string] {
  return target.startsWith('id:') ? ['group_id', target.slice(3)] : ['group_name', target.slice(5)]
}

// Runs browser uploads app-wide, so they continue while navigating.
export function UploadsProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const { refresh: refreshImports } = useImports()
  const [uploads, setUploads] = useState<Upload[]>([])
  const [uploadedCount, setUploadedCount] = useState(0)
  const xhrs = useRef(new Map<number, XMLHttpRequest>())

  const update = (key: number, patch: Partial<Upload>) =>
    setUploads((list) => list.map((u) => (u.key === key ? { ...u, ...patch } : u)))

  const send = (upload: Upload) => {
    const xhr = new XMLHttpRequest()
    xhrs.current.set(upload.key, xhr)

    const form = new FormData()
    const [field, value] = groupField(upload.target!)
    form.append(field, value)
    form.append('comic', upload.file)

    xhr.upload.onprogress = (e) => update(upload.key, { loaded: e.loaded })
    xhr.onload = () => {
      xhrs.current.delete(upload.key)
      let data: { error?: string } = {}
      try {
        data = JSON.parse(xhr.responseText)
      } catch {
        // Non-JSON error response.
      }
      if (xhr.status >= 200 && xhr.status < 300) {
        setUploads((list) => list.filter((u) => u.key !== upload.key))
        setUploadedCount((n) => n + 1)
        refreshImports()
      } else {
        update(upload.key, { status: 'error', error: data.error || `Upload failed (${xhr.status})` })
      }
    }
    xhr.onerror = () => {
      xhrs.current.delete(upload.key)
      update(upload.key, { status: 'error', error: 'Network error' })
    }

    xhr.open('POST', '/api/comics')
    for (const [k, v] of Object.entries(csrfHeaders)) xhr.setRequestHeader(k, v)
    xhr.send(form)
    update(upload.key, { status: 'uploading', loaded: 0, error: undefined })
  }

  // Start queued uploads while there are free slots.
  useEffect(() => {
    const active = uploads.filter((u) => u.status === 'uploading').length
    const queued = uploads.filter((u) => u.status === 'queued')
    queued.slice(0, Math.max(0, PARALLEL_UPLOADS - active)).forEach(send)
  }, [uploads])

  const cancelAll = useCallback(() => {
    xhrs.current.forEach((x) => x.abort())
    xhrs.current.clear()
    setUploads([])
  }, [])

  // Uploads belong to the signed-in user.
  useEffect(() => {
    if (!user) cancelAll()
  }, [user, cancelAll])

  // Leaving the app would abort uploads in progress, so ask first.
  const active = uploads.some(isUploading)
  useEffect(() => {
    if (!active) return
    const warn = (e: BeforeUnloadEvent) => {
      e.preventDefault()
      e.returnValue = '' // Required by some browsers.
    }
    window.addEventListener('beforeunload', warn)
    return () => window.removeEventListener('beforeunload', warn)
  }, [active])

  const value: UploadsState = {
    uploads,
    uploadedCount,
    addFiles: (files) => {
      const added = Array.from(files).map((file) => ({ key: nextKey++, file, status: 'added' as const, loaded: 0 }))
      setUploads((list) => [...list, ...added])
    },
    start: (target, groupName, keys) =>
      setUploads((list) =>
        list.map((u) =>
          (u.status === 'added' || u.status === 'error') && (!keys || keys.includes(u.key))
            ? { ...u, status: 'queued', error: undefined, target, groupName }
            : u,
        ),
      ),
    cancel: (key) => {
      xhrs.current.get(key)?.abort()
      xhrs.current.delete(key)
      setUploads((list) => list.filter((u) => u.key !== key))
    },
    cancelAll,
  }

  return <UploadsContext.Provider value={value}>{children}</UploadsContext.Provider>
}

export function useUploads() {
  const ctx = useContext(UploadsContext)
  if (!ctx) throw new Error('useUploads must be used inside UploadsProvider')
  return ctx
}

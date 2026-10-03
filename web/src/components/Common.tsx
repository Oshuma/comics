import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { Link } from 'react-router'
import { Pagination as BsPagination } from 'react-bootstrap'
import { errorMessage, pageThumbUrl } from '../api'

export function Spinner() {
  return (
    <div className="d-flex justify-content-center p-5">
      <div className="spinner-border text-secondary" role="status">
        <span className="visually-hidden">Loading...</span>
      </div>
    </div>
  )
}

export function Icon({ name, className = '' }: { name: string; className?: string }) {
  return <i className={`bi bi-${name} ${className}`} aria-hidden="true" />
}

// Loads data on mount and whenever deps change.
export function useLoad<T>(fn: () => Promise<T>, deps: unknown[]) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const load = useCallback(fn, deps)

  const reload = useCallback(async () => {
    try {
      setData(await load())
      setError(null)
    } catch (err) {
      setError(errorMessage(err))
    }
  }, [load])

  useEffect(() => {
    let cancelled = false
    load().then(
      (d) => {
        if (!cancelled) {
          setData(d)
          setError(null)
        }
      },
      (err) => !cancelled && setError(errorMessage(err)),
    )
    return () => {
      cancelled = true
    }
  }, [load])

  return { data, error, reload, setData }
}

export function LoadError({ error }: { error: string }) {
  return (
    <div className="alert alert-danger mx-auto mt-4" style={{ maxWidth: 600 }}>
      {error}
    </div>
  )
}

export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="row justify-content-center">
      <div className="col-md-6">
        <div className="card card-body bg-body-tertiary text-center py-5">
          <h1 className="text-body-secondary">{title}</h1>
          {children}
        </div>
      </div>
    </div>
  )
}

export function UploadButton({ groupId }: { groupId?: number }) {
  return (
    <Link to={groupId ? `/upload?group_id=${groupId}` : '/upload'} className="btn btn-outline-secondary btn-lg">
      <Icon name="cloud-upload" /> Upload Some!
    </Link>
  )
}

// Cover/page thumbnail, with a placeholder when there's no page.
export function Thumb({ pageId, empty }: { pageId: number | null; empty?: boolean }) {
  const [failed, setFailed] = useState(false)
  if (!pageId || failed) {
    return (
      <div className="thumb-img thumb-placeholder">
        <Icon name={empty ? 'collection' : 'book'} />
        {empty && <small>No Comics</small>}
      </div>
    )
  }
  return (
    <img className="thumb-img" src={pageThumbUrl(pageId)} alt="" loading="lazy" onError={() => setFailed(true)} />
  )
}

export function ReadState({ read, reading }: { read: boolean; reading: boolean }) {
  if (read) return <Icon name="check-lg" className="fs-4 text-success" />
  if (reading) return <Icon name="eye" className="fs-4" />
  return null
}

export function Pagination({
  page,
  total,
  perPage,
  onChange,
}: {
  page: number
  total: number
  perPage: number
  onChange: (page: number) => void
}) {
  const pages = Math.ceil(total / perPage)
  if (pages <= 1) return null

  const window = 4
  const from = Math.max(1, page - window)
  const to = Math.min(pages, page + window)
  const items = []
  for (let p = from; p <= to; p++) {
    items.push(
      <BsPagination.Item key={p} active={p === page} onClick={() => onChange(p)}>
        {p}
      </BsPagination.Item>,
    )
  }

  return (
    <BsPagination>
      <BsPagination.First disabled={page === 1} onClick={() => onChange(1)} />
      <BsPagination.Prev disabled={page === 1} onClick={() => onChange(page - 1)} />
      {from > 1 && <BsPagination.Ellipsis disabled />}
      {items}
      {to < pages && <BsPagination.Ellipsis disabled />}
      <BsPagination.Next disabled={page === pages} onClick={() => onChange(page + 1)} />
      <BsPagination.Last disabled={page === pages} onClick={() => onChange(pages)} />
    </BsPagination>
  )
}

// Mirrors Kaminari's page_entries_info.
export function EntriesInfo({ page, total, perPage }: { page: number; total: number; perPage: number }) {
  let text: string
  if (total === 0) text = 'No entries found'
  else if (total <= perPage) text = `Displaying ${total === 1 ? '1 entry' : `all ${total} entries`}`
  else {
    const first = (page - 1) * perPage + 1
    const last = Math.min(page * perPage, total)
    text = `Displaying entries ${first} - ${last} of ${total} in total`
  }
  return <p className="text-body-secondary">{text}</p>
}

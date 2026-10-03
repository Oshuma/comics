import { useCallback, useEffect, useRef, useState, type TouchEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { api, errorMessage, pageImageUrl, resumePath, type PageView } from '../api'
import { useFlash } from '../context'
import { Icon, LoadError, Spinner } from '../components/Common'

const SWIPE_THRESHOLD = 60

export default function Reader() {
  const params = useParams()
  const comicId = Number(params.id)
  const pageId = Number(params.pageId)
  const navigate = useNavigate()
  const flash = useFlash()
  const [view, setView] = useState<PageView | null>(null)
  const [error, setError] = useState<string | null>(null)
  const busy = useRef(false)
  const touchStart = useRef<{ x: number; y: number } | null>(null)

  useEffect(() => {
    let cancelled = false
    api.viewPage(comicId, pageId).then(
      (v) => {
        if (cancelled) return
        setView(v)
        document.title = `${v.comicName} (${v.number}/${v.pageCount}) · Comics`
        window.scrollTo(0, 0)
        // Preload the next page so it shows instantly.
        if (v.nextPageId) new Image().src = pageImageUrl(v.nextPageId)
      },
      (err) => !cancelled && setError(errorMessage(err)),
    )
    return () => {
      cancelled = true
    }
  }, [comicId, pageId])

  const finish = useCallback(
    async (goToNextComic: boolean) => {
      if (busy.current) return
      busy.current = true
      try {
        const { groupId, nextComicId } = await api.finishComic(comicId)
        if (goToNextComic && nextComicId) navigate(await resumePath(nextComicId))
        else navigate(`/groups/${groupId}`)
      } catch (err) {
        flash('danger', errorMessage(err))
      } finally {
        busy.current = false
      }
    },
    [comicId, navigate, flash],
  )

  const next = useCallback(() => {
    if (!view) return
    if (view.nextPageId) navigate(`/comics/${comicId}/pages/${view.nextPageId}`)
    else finish(true)
  }, [view, comicId, navigate, finish])

  const previous = useCallback(() => {
    if (!view) return
    if (view.prevPageId) navigate(`/comics/${comicId}/pages/${view.prevPageId}`)
    else navigate(`/groups/${view.groupId}`)
  }, [view, comicId, navigate])

  const confirmFinish = () => {
    if (window.confirm('Finished reading?')) finish(false)
  }

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.altKey || e.ctrlKey || e.metaKey) return
      if (e.key === 'ArrowRight') next()
      else if (e.key === 'ArrowLeft') previous()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [next, previous])

  const onTouchStart = (e: TouchEvent) => {
    if (e.touches.length !== 1) {
      touchStart.current = null
      return
    }
    touchStart.current = { x: e.touches[0].clientX, y: e.touches[0].clientY }
  }

  const onTouchEnd = (e: TouchEvent) => {
    const start = touchStart.current
    touchStart.current = null
    // Ignore pinch-zoomed views so panning doesn't turn pages.
    if (!start || (window.visualViewport && window.visualViewport.scale > 1.01)) return
    const dx = e.changedTouches[0].clientX - start.x
    const dy = e.changedTouches[0].clientY - start.y
    if (Math.abs(dx) < SWIPE_THRESHOLD || Math.abs(dx) < Math.abs(dy) * 1.5) return
    if (dx < 0) next()
    else previous()
  }

  if (error) return <LoadError error={error} />
  if (!view) return <Spinner />

  return (
    <div className="reader" data-bs-theme="dark">
      <nav className="reader-bar navbar fixed-top bg-body-tertiary border-bottom">
        <div className="container-fluid flex-nowrap gap-1">
          <div className="d-flex align-items-center gap-1 min-w-0">
            <button className="btn btn-link nav-link px-2" onClick={previous} title={view.prevPageId ? 'Previous Page' : 'View Group'}>
              <Icon name="chevron-left" className="fs-5" />
            </button>
            <Link to="/" className="nav-link px-2" title="Home">
              <Icon name="book-half" className="fs-5" />
            </Link>
            <Link to={`/groups/${view.groupId}`} className="nav-link px-2 text-truncate d-none d-sm-block">
              {view.groupName}
            </Link>
            <Link to={`/comics/${view.comicId}`} className="nav-link px-2 text-truncate">
              {view.comicName}
            </Link>
          </div>
          <div className="d-flex align-items-center gap-1 flex-shrink-0">
            <span className="navbar-text px-2 text-nowrap">
              {view.number} / {view.pageCount}
            </span>
            <button className="btn btn-link nav-link px-2" onClick={confirmFinish} title="Finish Comic">
              <Icon name="skip-end" className="fs-5" />
            </button>
            <button className="btn btn-link nav-link px-2" onClick={next} title="Next Page">
              <Icon name="chevron-right" className="fs-5" />
            </button>
          </div>
        </div>
      </nav>

      <img
        key={view.id}
        className="viewer"
        src={pageImageUrl(view.id)}
        alt={`Page ${view.number}`}
        onTouchStart={onTouchStart}
        onTouchEnd={onTouchEnd}
      />
    </div>
  )
}

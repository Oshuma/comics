import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { Badge, Button, Card, ProgressBar } from 'react-bootstrap'
import { Link } from 'react-router'
import { api, errorMessage, type ImportJob } from '../api'
import { useAuth, useFlash } from '../context'
import { Icon } from './Common'

// Poll quickly while imports run (for progress), and occasionally otherwise
// to notice imports started in another tab.
const ACTIVE_POLL_MS = 1000
const IDLE_POLL_MS = 30000

export const isActive = (j: ImportJob) => j.status === 'queued' || j.status === 'processing'

interface ImportsState {
  jobs: ImportJob[]
  refresh: () => Promise<void>
}

const ImportsContext = createContext<ImportsState>({ jobs: [], refresh: async () => {} })

// Shares the signed-in user's import jobs across the app.
export function ImportsProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const [jobs, setJobs] = useState<ImportJob[]>([])

  const refresh = useCallback(async () => {
    try {
      setJobs(await api.imports())
    } catch {
      // Keep the last known state; the next poll may succeed.
    }
  }, [])

  const signedIn = !!user
  const active = jobs.some(isActive)
  useEffect(() => {
    if (!signedIn) {
      setJobs([])
      return
    }
    refresh()
    const timer = setInterval(refresh, active ? ACTIVE_POLL_MS : IDLE_POLL_MS)
    return () => clearInterval(timer)
  }, [signedIn, active, refresh])

  return <ImportsContext.Provider value={{ jobs, refresh }}>{children}</ImportsContext.Provider>
}

export const useImports = () => useContext(ImportsContext)

const phaseLabels = { downloading: 'Downloading', extracting: 'Extracting', saving: 'Saving' }

// Progress bar for an active job: indeterminate while queued, saving, or
// when the size isn't known.
export function ImportProgressBar({ job }: { job: ImportJob }) {
  const p = job.progress
  let label = 'Queued'
  let percent: number | null = null
  if (job.status === 'processing') {
    label = p ? phaseLabels[p.phase] : 'Starting'
    if (p && p.total > 0 && p.phase !== 'saving') percent = Math.min(100, Math.round((p.done / p.total) * 100))
  }
  return (
    <>
      <div className="d-flex justify-content-between small text-body-secondary">
        <span>{label}</span>
        {percent !== null && <span>{percent}%</span>}
      </div>
      <ProgressBar
        className="import-progress"
        now={percent ?? 100}
        striped={percent === null}
        animated={job.status === 'processing' && percent === null}
        variant={job.status === 'queued' ? 'secondary' : 'info'}
        aria-label={`${job.name}: ${label}${percent !== null ? ` ${percent}%` : ''}`}
      />
    </>
  )
}

function StatusBadge({ job }: { job: ImportJob }) {
  switch (job.status) {
    case 'done':
      return <Badge bg="success">Done</Badge>
    case 'failed':
      return <Badge bg="danger">Failed</Badge>
    default:
      return null
  }
}

// Recent imports with their status, and buttons to cancel or dismiss them.
export function ImportList() {
  const { jobs, refresh } = useImports()
  const flash = useFlash()
  if (jobs.length === 0) return null

  const remove = async (ids: number[]) => {
    try {
      await Promise.all(ids.map((id) => api.deleteImport(id)))
    } catch (err) {
      flash('danger', errorMessage(err))
    }
    refresh()
  }
  const finished = jobs.filter((j) => !isActive(j)).map((j) => j.id)

  return (
    <>
      <div className="d-flex align-items-center justify-content-between mt-4 mb-2">
        <h6 className="mb-0">Recent Imports</h6>
        {finished.length > 0 && (
          <Button size="sm" variant="link" className="p-0" onClick={() => remove(finished)}>
            Clear finished
          </Button>
        )}
      </div>
      {jobs.map((job) => (
        <Card key={job.id} className="mb-2 import-job">
          <Card.Body className="py-2">
            <div className="d-flex align-items-center gap-2">
              <div className="flex-grow-1 min-w-0">
                <div className="text-truncate" title={job.filename}>
                  {job.status === 'done' && job.comicId ? <Link to={`/comics/${job.comicId}`}>{job.name}</Link> : job.name}
                </div>
                <small className="text-body-secondary">
                  {job.groupName}
                  {job.source === 'archive.org' && ' · from archive.org'}
                </small>
                {job.error && <div className="text-danger small fw-bold">{job.error}</div>}
              </div>
              <StatusBadge job={job} />
              <Button
                size="sm"
                variant="outline-secondary"
                onClick={() => remove([job.id])}
                title={isActive(job) ? 'Cancel import' : 'Dismiss'}
                aria-label={isActive(job) ? 'Cancel import' : 'Dismiss'}
              >
                <Icon name={isActive(job) ? 'slash-circle' : 'x-lg'} />
              </Button>
            </div>
            {isActive(job) && (
              <div className="mt-2">
                <ImportProgressBar job={job} />
              </div>
            )}
          </Card.Body>
        </Card>
      ))}
    </>
  )
}

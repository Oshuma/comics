import type { ReactNode } from 'react'
import { Badge, NavDropdown, ProgressBar } from 'react-bootstrap'
import { Link } from 'react-router'
import { Icon } from './Common'
import { ImportProgressBar, isActive, useImports } from './Imports'
import { isUploading, useUploads, type Upload } from './Uploads'

function UploadProgressBar({ upload }: { upload: Upload }) {
  const percent = upload.status === 'uploading' ? Math.min(100, Math.round((upload.loaded / upload.file.size) * 100)) : null
  const label = upload.status === 'queued' ? 'Waiting to upload' : 'Uploading'
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
        variant={percent === null ? 'secondary' : 'success'}
        aria-label={`${upload.file.name}: ${label}${percent !== null ? ` ${percent}%` : ''}`}
      />
    </>
  )
}

function Entry({ name, title, detail, children }: { name: string; title: string; detail: string; children: ReactNode }) {
  return (
    <div className="px-3 py-2">
      <div className="text-truncate fw-medium" title={title}>
        {name}
      </div>
      <div className="small text-body-secondary text-truncate mb-1">{detail}</div>
      {children}
    </div>
  )
}

// Navbar menu listing uploads and imports in progress; hidden when there
// are none.
export function ImportsMenu() {
  const { jobs } = useImports()
  const { uploads } = useUploads()
  const activeJobs = jobs.filter(isActive)
  const activeUploads = uploads.filter(isUploading)
  const count = activeJobs.length + activeUploads.length
  if (count === 0) return null

  return (
    <NavDropdown
      align="end"
      className="imports-menu"
      title={
        <>
          <span className="spinner-border spinner-border-sm" aria-hidden="true" />
          <span className="ms-2">Importing</span>
          <Badge bg="info" pill className="ms-1">
            {count}
          </Badge>
        </>
      }
    >
      <NavDropdown.Header>
        Importing {count} {count === 1 ? 'comic' : 'comics'}
      </NavDropdown.Header>
      {activeUploads.map((u) => (
        <Entry key={`u${u.key}`} name={u.file.name.replace(/\.(cbz|cbr|zip|rar)$/i, '')} title={u.file.name} detail={u.groupName ?? ''}>
          <UploadProgressBar upload={u} />
        </Entry>
      ))}
      {activeJobs.map((job) => (
        <Entry
          key={`j${job.id}`}
          name={job.name}
          title={job.filename}
          detail={job.groupName + (job.source === 'archive.org' ? ' · from archive.org' : '')}
        >
          <ImportProgressBar job={job} />
        </Entry>
      ))}
      <NavDropdown.Divider />
      <NavDropdown.Item as={Link} to="/upload">
        <Icon name="list-ul" className="me-2" />
        All Imports
      </NavDropdown.Item>
    </NavDropdown>
  )
}

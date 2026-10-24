import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { Button, Form, InputGroup, Table } from 'react-bootstrap'
import { Link, useSearchParams } from 'react-router'
import {
  api,
  errorMessage,
  humanSize,
  type ArchiveFile,
  type ArchiveResults,
  type ArchiveSort,
  type ImportJob,
} from '../api'
import { useFlash } from '../context'
import Layout from '../components/Layout'
import { EmptyState, Icon, LoadError, NameModal, Pagination, Spinner, useLoad } from '../components/Common'
import { useImports } from '../components/Imports'

// A row's import: a request being sent or rejected, or the latest job for the file.
type ImportState =
  | { status: 'requesting' }
  | { status: 'queued' | 'processing' }
  | { status: 'done'; comicId: number | null }
  | { status: 'error'; error: string }

function jobState(job: ImportJob): ImportState {
  switch (job.status) {
    case 'done':
      return { status: 'done', comicId: job.comicId }
    case 'failed':
      return { status: 'error', error: job.error || 'Import failed.' }
    default:
      return { status: job.status }
  }
}

const fileKey = (f: ArchiveFile) => `${f.identifier}/${f.name}`

// File name without its folder or extension.
function displayName(name: string) {
  return (name.split('/').pop() ?? name).replace(/\.(cbz|cbr)$/i, '')
}

function ImportButton({
  file,
  state,
  groupId,
  maxBytes,
  onImport,
}: {
  file: ArchiveFile
  state?: ImportState
  groupId: string
  maxBytes: number
  onImport: () => void
}) {
  if (state?.status === 'requesting' || state?.status === 'queued' || state?.status === 'processing') {
    return (
      <Button size="sm" variant="outline-secondary" className="text-nowrap" disabled>
        {state.status === 'queued' ? (
          <Icon name="hourglass-split" className="me-1" />
        ) : (
          <span className="spinner-border spinner-border-sm me-1" aria-hidden="true" />
        )}
        {state.status === 'queued' ? 'Queued' : 'Importing...'}
      </Button>
    )
  }
  if (state?.status === 'done') {
    return state.comicId ? (
      <Link to={`/comics/${state.comicId}`} className="btn btn-sm btn-outline-success text-nowrap">
        <Icon name="check-lg" /> View
      </Link>
    ) : (
      <span className="text-success text-nowrap small">
        <Icon name="check-lg" /> Imported
      </span>
    )
  }
  const tooLarge = file.size > maxBytes
  const reason = tooLarge ? 'Too large to import' : !groupId ? 'Choose a group first' : 'Import into the chosen group'
  return (
    <Button
      size="sm"
      variant={state ? 'outline-danger' : 'primary'}
      className="text-nowrap"
      disabled={tooLarge || !groupId}
      onClick={onImport}
      title={reason}
    >
      <Icon name={state ? 'arrow-clockwise' : 'download'} /> {state ? 'Retry' : 'Import'}
    </Button>
  )
}

const sortLabels: Record<ArchiveSort, string> = {
  relevance: 'Relevance',
  downloads: 'Most downloaded',
  newest: 'Newest',
  title: 'Title',
  size: 'Largest',
}

const isSort = (s: string | null): s is ArchiveSort => !!s && s in sortLabels

function Results({
  results,
  imports,
  groupId,
  sort,
  onImport,
  onPage,
  onSort,
}: {
  results: ArchiveResults
  imports: Record<string, ImportState>
  groupId: string
  sort: ArchiveSort
  onImport: (f: ArchiveFile) => void
  onPage: (page: number) => void
  onSort: (sort: ArchiveSort) => void
}) {
  if (results.files.length === 0) {
    return <EmptyState title="No Comics Found" />
  }
  const pages = Math.ceil(results.totalItems / results.perPage)
  return (
    <>
      <div className="d-flex flex-wrap align-items-center justify-content-between gap-2 mb-3">
        <span className="text-body-secondary small">
          {results.totalItems} matching {results.totalItems === 1 ? 'item' : 'items'} on archive.org
          {pages > 1 && ` · page ${results.page} of ${pages}`}
        </span>
        <InputGroup size="sm" className="w-auto">
          <InputGroup.Text>Sort</InputGroup.Text>
          <Form.Select value={sort} onChange={(e) => onSort(e.target.value as ArchiveSort)} aria-label="Sort results">
            {Object.entries(sortLabels).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </Form.Select>
        </InputGroup>
      </div>
      <Table hover className="align-middle">
        <thead>
          <tr>
            <th>Name</th>
            <th className="d-none d-md-table-cell">Item</th>
            <th className="text-end text-nowrap">Size</th>
            <th aria-label="Import" />
          </tr>
        </thead>
        <tbody>
          {results.files.map((f) => {
            const state = imports[fileKey(f)]
            return (
              <tr key={fileKey(f)}>
                <td className="text-break">
                  <span className="fw-medium" title={f.name}>
                    {displayName(f.name)}
                  </span>
                  <a
                    href={`https://archive.org/details/${encodeURIComponent(f.identifier)}`}
                    target="_blank"
                    rel="noreferrer"
                    className="d-md-none d-block small text-body-secondary"
                  >
                    {f.itemTitle}
                  </a>
                  {state?.status === 'error' && <div className="small text-danger">{state.error}</div>}
                </td>
                <td className="d-none d-md-table-cell text-break">
                  <a
                    href={`https://archive.org/details/${encodeURIComponent(f.identifier)}`}
                    target="_blank"
                    rel="noreferrer"
                    className="link-secondary"
                  >
                    {f.itemTitle} <Icon name="box-arrow-up-right" className="small" />
                  </a>
                </td>
                <td className="text-end text-nowrap">{f.size ? humanSize(f.size) : '?'}</td>
                <td className="text-end">
                  <ImportButton
                    file={f}
                    state={state}
                    groupId={groupId}
                    maxBytes={results.maxBytes}
                    onImport={() => onImport(f)}
                  />
                </td>
              </tr>
            )
          })}
        </tbody>
      </Table>
      <Pagination page={results.page} total={results.totalItems} perPage={results.perPage} onChange={onPage} />
    </>
  )
}

export default function Search() {
  const flash = useFlash()
  const [params, setParams] = useSearchParams()
  const q = params.get('q') ?? ''
  const page = Number(params.get('page')) || 1
  const sortParam = params.get('sort')
  const sort: ArchiveSort = isSort(sortParam) ? sortParam : 'relevance'
  const [input, setInput] = useState(q)
  const [groupId, setGroupId] = useState(params.get('group_id') ?? '')
  const [creating, setCreating] = useState(false)
  // Requests in flight, or rejected before a job was created.
  const [requests, setRequests] = useState<Record<string, ImportState>>({})
  const { jobs, refresh: refreshImports } = useImports()
  const { data: groups, reload: reloadGroups } = useLoad(api.groups, [])
  // Tagged with the search it's for, so stale results aren't shown while a new search loads.
  const searchKey = `${page}:${sort}:${q}`
  const { data, error } = useLoad(
    async () => ({ key: searchKey, results: q ? await api.archiveSearch(q, page, sort) : null }),
    [searchKey],
  )
  const results = data?.key === searchKey ? data.results : null

  // Updates the URL; the first page and relevance sort are left out.
  const go = (query: string, p: number, s: ArchiveSort) =>
    setParams({ q: query, ...(p > 1 && { page: String(p) }), ...(s !== 'relevance' && { sort: s }) })

  // Start fresh when the search is cleared, e.g. from the navbar icon, keeping
  // a group passed in the URL.
  useEffect(() => {
    if (!q) setGroupId(params.get('group_id') ?? '')
  }, [q])

  // Ignore a group that has since been deleted.
  const selectedGroup = groups?.some((g) => String(g.id) === groupId) ? groupId : ''

  const search = (e: FormEvent) => {
    e.preventDefault()
    const query = input.trim()
    if (query) go(query, 1, sort)
  }

  // Each file's import state: a pending request wins, otherwise its latest job.
  const imports = useMemo(() => {
    const states: Record<string, ImportState> = {}
    for (const job of jobs) {
      const key = `${job.archiveIdentifier}/${job.archiveName}`
      if (job.source === 'archive.org' && !states[key]) states[key] = jobState(job) // jobs are newest first
    }
    return { ...states, ...requests }
  }, [jobs, requests])

  const importFile = async (f: ArchiveFile) => {
    const key = fileKey(f)
    const clear = () =>
      setRequests((s) => {
        const { [key]: _, ...rest } = s
        return rest
      })
    setRequests((s) => ({ ...s, [key]: { status: 'requesting' } }))
    try {
      await api.archiveImport(f, Number(selectedGroup))
      await refreshImports()
      clear()
    } catch (err) {
      setRequests((s) => ({ ...s, [key]: { status: 'error', error: errorMessage(err) } }))
    }
  }

  const createGroup = async (name: string) => {
    try {
      const group = await api.createGroup(name)
      flash('success', 'Group added.')
      setCreating(false)
      await reloadGroups()
      setGroupId(String(group.id))
    } catch (err) {
      flash('danger', `There was a problem adding that group: ${errorMessage(err)}`)
    }
  }

  let content
  if (!q) {
    content = (
      <EmptyState title="Search archive.org">
        <p className="text-body-secondary">Find CBZ and CBR comics on archive.org and import them into a group.</p>
      </EmptyState>
    )
  } else if (error) content = <LoadError error={error} />
  else if (!results) content = <Spinner />
  else {
    content = (
      <Results
        results={results}
        imports={imports}
        groupId={selectedGroup}
        onImport={importFile}
        sort={sort}
        onPage={(p) => go(q, p, sort)}
        onSort={(s) => go(q, 1, s)}
      />
    )
  }

  return (
    <Layout title={q ? `Search: ${q}` : 'Search archive.org'}>
      <div className="row g-2 mb-3">
        <div className="col-lg-7">
          <Form onSubmit={search} role="search">
            <InputGroup>
              <Form.Control
                type="search"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder="Search archive.org..."
                aria-label="Search archive.org"
                autoFocus={!q}
              />
              <Button type="submit" disabled={!input.trim()}>
                <Icon name="search" /> Search
              </Button>
            </InputGroup>
          </Form>
        </div>
        <div className="col-lg-5">
          <InputGroup>
            <InputGroup.Text>Import into</InputGroup.Text>
            <Form.Select value={selectedGroup} onChange={(e) => setGroupId(e.target.value)} aria-label="Group">
              <option value="">Group...</option>
              {groups?.map((g) => (
                <option key={g.id} value={g.id}>
                  {g.name}
                </option>
              ))}
            </Form.Select>
            <Button variant="outline-secondary" onClick={() => setCreating(true)} title="New Group">
              <Icon name="plus-lg" />
            </Button>
          </InputGroup>
        </div>
      </div>

      {content}

      <NameModal
        show={creating}
        title="New Group"
        placeholder="Group name..."
        onHide={() => setCreating(false)}
        onSave={createGroup}
      />
    </Layout>
  )
}

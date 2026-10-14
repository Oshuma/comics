import { useState, type FormEvent } from 'react'
import { Button, Form, InputGroup, Nav, Table } from 'react-bootstrap'
import { Link, NavLink, useSearchParams } from 'react-router'
import { api, errorMessage, humanSize, type ArchiveFile, type ArchiveResults, type Comic } from '../api'
import { useFlash } from '../context'
import Layout from '../components/Layout'
import { EmptyState, Icon, LoadError, NameModal, Pagination, Spinner, useLoad } from '../components/Common'

type ImportState = { status: 'importing' } | { status: 'done'; comic: Comic } | { status: 'error'; error: string }

const groupKey = 'search-group'

function loadGroup(): string {
  try {
    return localStorage.getItem(groupKey) ?? ''
  } catch {
    return ''
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
  if (state?.status === 'importing') {
    return (
      <Button size="sm" variant="outline-secondary" className="text-nowrap" disabled>
        <span className="spinner-border spinner-border-sm me-1" aria-hidden="true" />
        Importing...
      </Button>
    )
  }
  if (state?.status === 'done') {
    return (
      <Link to={`/comics/${state.comic.id}`} className="btn btn-sm btn-outline-success text-nowrap">
        <Icon name="check-lg" /> View
      </Link>
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

function Results({
  results,
  imports,
  groupId,
  onImport,
  onPage,
}: {
  results: ArchiveResults
  imports: Record<string, ImportState>
  groupId: string
  onImport: (f: ArchiveFile) => void
  onPage: (page: number) => void
}) {
  if (results.files.length === 0) {
    return <EmptyState title="No Comics Found" />
  }
  const pages = Math.ceil(results.totalItems / results.perPage)
  return (
    <>
      <p className="text-body-secondary small">
        {results.totalItems} matching {results.totalItems === 1 ? 'item' : 'items'} on archive.org
        {pages > 1 && ` · page ${results.page} of ${pages}`}
      </p>
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
  const [input, setInput] = useState(q)
  const [groupId, setGroupId] = useState(loadGroup)
  const [creating, setCreating] = useState(false)
  const [imports, setImports] = useState<Record<string, ImportState>>({})
  const { data: groups, reload: reloadGroups } = useLoad(api.groups, [])
  // Tagged with the search it's for, so stale results aren't shown while a new search loads.
  const searchKey = `${page}:${q}`
  const { data, error } = useLoad(
    async () => ({ key: searchKey, results: q ? await api.archiveSearch(q, page) : null }),
    [searchKey],
  )
  const results = data?.key === searchKey ? data.results : null

  const chooseGroup = (id: string) => {
    setGroupId(id)
    try {
      localStorage.setItem(groupKey, id)
    } catch {
      // Not persisted; the choice still applies for this visit.
    }
  }

  // Ignore a remembered group that has since been deleted.
  const selectedGroup = groups?.some((g) => String(g.id) === groupId) ? groupId : ''

  const search = (e: FormEvent) => {
    e.preventDefault()
    const query = input.trim()
    if (query) setParams({ q: query })
  }

  const importFile = async (f: ArchiveFile) => {
    const key = fileKey(f)
    setImports((s) => ({ ...s, [key]: { status: 'importing' } }))
    try {
      const comic = await api.archiveImport(f, Number(selectedGroup))
      setImports((s) => ({ ...s, [key]: { status: 'done', comic } }))
    } catch (err) {
      setImports((s) => ({ ...s, [key]: { status: 'error', error: errorMessage(err) } }))
    }
  }

  const createGroup = async (name: string) => {
    try {
      const group = await api.createGroup(name)
      flash('success', 'Group added.')
      setCreating(false)
      await reloadGroups()
      chooseGroup(String(group.id))
    } catch (err) {
      flash('danger', `There was a problem adding that group: ${errorMessage(err)}`)
    }
  }

  const nav = (
    <Nav.Link as={NavLink} to="/search">
      <Icon name="search" /> Search archive.org
    </Nav.Link>
  )

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
        onPage={(p) => setParams({ q, page: String(p) })}
      />
    )
  }

  return (
    <Layout nav={nav} title={q ? `Search: ${q}` : 'Search archive.org'}>
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
            <Form.Select value={selectedGroup} onChange={(e) => chooseGroup(e.target.value)} aria-label="Group">
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

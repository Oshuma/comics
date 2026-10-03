import { useEffect, useRef, useState, type DragEvent } from 'react'
import { Button, Card, Form, InputGroup, Nav, ProgressBar } from 'react-bootstrap'
import { Link, NavLink, useSearchParams } from 'react-router'
import { api, csrfHeaders, errorMessage, humanSize, type Comic } from '../api'
import { useFlash } from '../context'
import Layout from '../components/Layout'
import { EmptyState, Icon, LoadError, Spinner, useLoad } from '../components/Common'
import { GroupNameModal } from '../components/GroupMenu'

const PARALLEL_UPLOADS = 3

type Status = 'added' | 'queued' | 'uploading' | 'done' | 'error'

interface Item {
  key: number
  file: File
  status: Status
  loaded: number
  error?: string
  comic?: Comic
}

let nextKey = 1

// Target group: "id:<n>" for an existing group or "name:<s>" to find/create by name.
function groupFields(selection: string): [string, string] {
  return selection.startsWith('id:') ? ['group_id', selection.slice(3)] : ['group_name', selection.slice(5)]
}

export default function Upload() {
  const flash = useFlash()
  const [params] = useSearchParams()
  const { data: groups, error, reload } = useLoad(api.groups, [])
  const [selection, setSelection] = useState(() => {
    if (params.get('group_id')) return `id:${params.get('group_id')}`
    if (params.get('group_name')) return `name:${params.get('group_name')}`
    return ''
  })
  const [items, setItems] = useState<Item[]>([])
  const [dragging, setDragging] = useState(false)
  const [creating, setCreating] = useState(false)
  const fileInput = useRef<HTMLInputElement>(null)
  const xhrs = useRef(new Map<number, XMLHttpRequest>())

  const update = (key: number, patch: Partial<Item>) =>
    setItems((list) => list.map((i) => (i.key === key ? { ...i, ...patch } : i)))

  const addFiles = (files: FileList | null) => {
    if (!files) return
    const added = Array.from(files).map((file) => ({ key: nextKey++, file, status: 'added' as Status, loaded: 0 }))
    setItems((list) => [...list, ...added])
  }

  const startUpload = (item: Item) => {
    const xhr = new XMLHttpRequest()
    xhrs.current.set(item.key, xhr)

    const form = new FormData()
    const [field, value] = groupFields(selection)
    form.append(field, value)
    form.append('comic', item.file)

    xhr.upload.onprogress = (e) => update(item.key, { loaded: e.loaded })
    xhr.onload = () => {
      xhrs.current.delete(item.key)
      let data: { error?: string } & Partial<Comic> = {}
      try {
        data = JSON.parse(xhr.responseText)
      } catch {
        // Non-JSON error response.
      }
      if (xhr.status >= 200 && xhr.status < 300) {
        update(item.key, { status: 'done', loaded: item.file.size, comic: data as Comic })
      } else {
        update(item.key, { status: 'error', error: data.error || `Upload failed (${xhr.status})` })
      }
    }
    xhr.onerror = () => {
      xhrs.current.delete(item.key)
      update(item.key, { status: 'error', error: 'Network error' })
    }

    xhr.open('POST', '/api/comics')
    for (const [k, v] of Object.entries(csrfHeaders)) xhr.setRequestHeader(k, v)
    xhr.send(form)
    update(item.key, { status: 'uploading', loaded: 0, error: undefined })
  }

  // Start queued uploads while there are free slots.
  useEffect(() => {
    const active = items.filter((i) => i.status === 'uploading').length
    const queued = items.filter((i) => i.status === 'queued')
    queued.slice(0, Math.max(0, PARALLEL_UPLOADS - active)).forEach(startUpload)
    // startUpload reads the latest selection on each render.
  }, [items])

  // A group created by name during upload now exists; refresh the list.
  const anyDone = items.some((i) => i.status === 'done')
  useEffect(() => {
    if (anyDone) reload()
  }, [anyDone, reload])

  useEffect(() => () => xhrs.current.forEach((x) => x.abort()), [])

  const enqueue = (keys?: number[]) =>
    setItems((list) =>
      list.map((i) =>
        (i.status === 'added' || i.status === 'error') && (!keys || keys.includes(i.key))
          ? { ...i, status: 'queued', error: undefined }
          : i,
      ),
    )

  const cancel = (key: number) => {
    xhrs.current.get(key)?.abort()
    xhrs.current.delete(key)
    setItems((list) => list.filter((i) => i.key !== key))
  }

  const cancelAll = () => {
    xhrs.current.forEach((x) => x.abort())
    xhrs.current.clear()
    setItems((list) => list.filter((i) => i.status === 'done'))
  }

  const deleteUploaded = async (item: Item) => {
    if (!item.comic || !window.confirm('Delete comic?')) return
    try {
      await api.deleteComic(item.comic.id)
      setItems((list) => list.filter((i) => i.key !== item.key))
    } catch (err) {
      flash('danger', errorMessage(err))
    }
  }

  const createGroup = async (name: string) => {
    try {
      const group = await api.createGroup(name)
      flash('success', 'Group added.')
      setCreating(false)
      await reload()
      setSelection(`id:${group.id}`)
    } catch (err) {
      flash('danger', `There was a problem adding that group: ${errorMessage(err)}`)
    }
  }

  const onDrop = (e: DragEvent) => {
    e.preventDefault()
    setDragging(false)
    if (selection) addFiles(e.dataTransfer.files)
  }

  const nav = (
    <>
      <Nav.Link as={NavLink} to="/upload">
        <Icon name="cloud-upload" /> Upload Comics
      </Nav.Link>
      <Nav.Link onClick={() => setCreating(true)}>
        <Icon name="plus-lg" /> New Group
      </Nav.Link>
    </>
  )

  if (error) return <Layout nav={nav}><LoadError error={error} /></Layout>
  if (!groups) return <Layout nav={nav}><Spinner /></Layout>

  const uploading = items.filter((i) => i.status === 'uploading' || i.status === 'queued')
  const totalBytes = uploading.reduce((n, i) => n + i.file.size, 0)
  const loadedBytes = uploading.reduce((n, i) => n + i.loaded, 0)
  const pending = items.some((i) => i.status === 'added' || i.status === 'error')
  const selectedName = selection.startsWith('name:') ? selection.slice(5) : null
  const nameIsExisting = selectedName !== null && groups.some((g) => g.name === selectedName)

  return (
    <Layout nav={nav} title="Upload Comics">
      <div className="row justify-content-center">
        <div className="col-md-8 col-lg-6">
          {groups.length === 0 && !selectedName && (
            <EmptyState title="No Groups">
              <p>
                Create a group with the <Icon name="plus-lg" /> button, then add some comics.
              </p>
            </EmptyState>
          )}

          <InputGroup className="my-3">
            <Form.Select value={selection} onChange={(e) => setSelection(e.target.value)} aria-label="Group">
              <option value="">Group...</option>
              {selectedName && !nameIsExisting && <option value={selection}>{selectedName} (new)</option>}
              {groups.map((g) => (
                <option key={g.id} value={`id:${g.id}`}>
                  {g.name}
                </option>
              ))}
            </Form.Select>
            <Button variant="outline-secondary" onClick={() => setCreating(true)} title="New Group">
              <Icon name="plus-lg" />
            </Button>
          </InputGroup>

          <Card
            className={`drop-zone mb-3 ${dragging ? 'is-dragging' : ''} ${selection ? '' : 'is-disabled'}`}
            onDragOver={(e) => {
              e.preventDefault()
              setDragging(true)
            }}
            onDragLeave={() => setDragging(false)}
            onDrop={onDrop}
            onClick={() => selection && fileInput.current?.click()}
          >
            <Card.Body className="text-center py-4">
              <Icon name="file-earmark-zip" className="fs-1 text-body-secondary" />
              <p className="mb-0 mt-2">
                {selection ? 'Drop CBZ / CBR files here, or click to choose' : 'Select a group first'}
              </p>
            </Card.Body>
          </Card>
          <input
            ref={fileInput}
            type="file"
            multiple
            accept=".cbz,.cbr,.zip,.rar"
            className="d-none"
            onChange={(e) => {
              addFiles(e.target.files)
              e.target.value = ''
            }}
          />

          <div className="d-flex gap-2 mb-3">
            <Button variant="success" disabled={!selection} onClick={() => fileInput.current?.click()}>
              <Icon name="plus-lg" /> Add Comics
            </Button>
            <Button disabled={!selection || !pending} onClick={() => enqueue()}>
              <Icon name="cloud-upload" /> Start All
            </Button>
            <Button variant="outline-secondary" disabled={items.length === 0} onClick={cancelAll}>
              <Icon name="slash-circle" /> Cancel All
            </Button>
          </div>

          {uploading.length > 0 && (
            <ProgressBar
              striped
              animated
              variant="success"
              className="mb-3"
              now={totalBytes ? (loadedBytes / totalBytes) * 100 : 0}
            />
          )}

          {items.map((item) => (
            <Card key={item.key} className="mb-2">
              <Card.Body className="py-2">
                <div className="d-flex align-items-center gap-2">
                  <div className="flex-grow-1 min-w-0">
                    <div className="text-truncate">
                      {item.status === 'done' && item.comic ? (
                        <Link to={`/comics/${item.comic.id}`}>{item.file.name}</Link>
                      ) : (
                        item.file.name
                      )}
                    </div>
                    <small className="text-body-secondary">
                      {humanSize(item.file.size)}
                      {item.status === 'done' && item.comic && ` · ${item.comic.pageCount} pages`}
                    </small>
                    {item.error && <div className="text-danger small fw-bold">{item.error}</div>}
                  </div>
                  {(item.status === 'added' || item.status === 'error') && (
                    <Button size="sm" disabled={!selection} onClick={() => enqueue([item.key])}>
                      <Icon name="cloud-upload" /> {item.status === 'error' ? 'Retry' : 'Start'}
                    </Button>
                  )}
                  {item.status === 'done' ? (
                    <Button size="sm" variant="danger" onClick={() => deleteUploaded(item)}>
                      <Icon name="trash" /> Delete
                    </Button>
                  ) : (
                    <Button size="sm" variant="outline-secondary" onClick={() => cancel(item.key)}>
                      <Icon name="slash-circle" /> Cancel
                    </Button>
                  )}
                </div>
                {item.status === 'uploading' && (
                  <ProgressBar
                    striped
                    animated
                    variant={item.loaded >= item.file.size ? 'info' : 'success'}
                    className="mt-2"
                    now={(item.loaded / item.file.size) * 100}
                    label={item.loaded >= item.file.size ? 'Processing...' : undefined}
                  />
                )}
                {item.status === 'queued' && <small className="text-body-secondary">Queued...</small>}
              </Card.Body>
            </Card>
          ))}
        </div>
      </div>

      <GroupNameModal show={creating} title="New Group" onHide={() => setCreating(false)} onSave={createGroup} />
    </Layout>
  )
}

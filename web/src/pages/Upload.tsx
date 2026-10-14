import { useEffect, useRef, useState, type DragEvent } from 'react'
import { Alert, Button, Card, Form, InputGroup, Nav, ProgressBar } from 'react-bootstrap'
import { NavLink, useSearchParams } from 'react-router'
import { api, errorMessage, humanSize } from '../api'
import { useFlash } from '../context'
import Layout from '../components/Layout'
import { EmptyState, Icon, LoadError, NameModal, Spinner, useLoad } from '../components/Common'
import { ImportList } from '../components/Imports'
import { useUploads } from '../components/Uploads'

export default function Upload() {
  const flash = useFlash()
  const [params] = useSearchParams()
  const { data: groups, error, reload } = useLoad(api.groups, [])
  const [selection, setSelection] = useState(() => {
    if (params.get('group_id')) return `id:${params.get('group_id')}`
    if (params.get('group_name')) return `name:${params.get('group_name')}`
    return ''
  })
  const { uploads: items, uploadedCount, addFiles: add, start, cancel, cancelAll } = useUploads()
  const [dragging, setDragging] = useState(false)
  const [creating, setCreating] = useState(false)
  const fileInput = useRef<HTMLInputElement>(null)

  // An upload may have created its group by name.
  useEffect(() => {
    if (uploadedCount > 0) reload()
  }, [uploadedCount, reload])

  const addFiles = (files: FileList | null) => {
    if (files) add(files)
  }

  const enqueue = (keys?: number[]) => {
    const name = selection.startsWith('name:')
      ? selection.slice(5)
      : groups?.find((g) => `id:${g.id}` === selection)?.name ?? ''
    start(selection, name, keys)
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
    addFiles(e.dataTransfer.files)
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
  // Files were added but can't be uploaded until a group is chosen.
  const needsGroup = !selection && items.some((i) => i.status === 'added' || i.status === 'error')
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
            <Form.Select
              value={selection}
              onChange={(e) => setSelection(e.target.value)}
              aria-label="Group"
              isInvalid={needsGroup}
            >
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
          {needsGroup && (
            <Alert variant="warning" className="py-2 d-flex align-items-center gap-2" role="alert">
              <Icon name="exclamation-triangle" />
              Select a group for these comics before uploading them.
            </Alert>
          )}

          <Card
            className={`drop-zone mb-3 ${dragging ? 'is-dragging' : ''}`}
            onDragOver={(e) => {
              e.preventDefault()
              setDragging(true)
            }}
            onDragLeave={() => setDragging(false)}
            onDrop={onDrop}
            onClick={() => fileInput.current?.click()}
          >
            <Card.Body className="text-center py-4">
              <Icon name="file-earmark-zip" className="fs-1 text-body-secondary" />
              <p className="mb-0 mt-2">
                Drop CBZ / CBR files here, or click to choose
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
            <Button variant="success" onClick={() => fileInput.current?.click()}>
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
                    <div className="text-truncate">{item.file.name}</div>
                    <small className="text-body-secondary">
                      {humanSize(item.file.size)}
                      {item.groupName && ` · ${item.groupName}`}
                    </small>
                    {item.error && <div className="text-danger small fw-bold">{item.error}</div>}
                  </div>
                  {(item.status === 'added' || item.status === 'error') && (
                    <Button size="sm" disabled={!selection} onClick={() => enqueue([item.key])}>
                      <Icon name="cloud-upload" /> {item.status === 'error' ? 'Retry' : 'Start'}
                    </Button>
                  )}
                  <Button size="sm" variant="outline-secondary" onClick={() => cancel(item.key)}>
                    <Icon name="slash-circle" /> Cancel
                  </Button>
                </div>
                {item.status === 'uploading' && (
                  <ProgressBar
                    striped
                    animated
                    variant={item.loaded >= item.file.size ? 'info' : 'success'}
                    className="mt-2"
                    now={(item.loaded / item.file.size) * 100}
                    label={item.loaded >= item.file.size ? 'Saving...' : undefined}
                  />
                )}
                {item.status === 'queued' && <small className="text-body-secondary">Queued...</small>}
              </Card.Body>
            </Card>
          ))}

          <ImportList />
        </div>
      </div>

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

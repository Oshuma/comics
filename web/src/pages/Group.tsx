import { useState, type MouseEvent } from 'react'
import { Button, Card, Collapse, Dropdown, Form, Nav, NavDropdown, Table } from 'react-bootstrap'
import { Link, NavLink, useNavigate, useParams } from 'react-router'
import { api, errorMessage, resumePath, type Comic, type Group } from '../api'
import { useFlash } from '../context'
import Layout from '../components/Layout'
import {
  EmptyState,
  Icon,
  LoadError,
  ReadState,
  Spinner,
  Thumb,
  UploadButton,
  useLoad,
  useViewMode,
  ViewToggle,
} from '../components/Common'
import { useGroupMenu } from '../components/GroupMenu'

interface ComicProps {
  comic: Comic
  groups: Group[]
  onRemoved: () => void
}

function useComicActions({ comic, onRemoved }: ComicProps) {
  const flash = useFlash()
  const navigate = useNavigate()

  const read = async (e: MouseEvent) => {
    e.preventDefault()
    try {
      navigate(await resumePath(comic.id))
    } catch (err) {
      flash('danger', errorMessage(err))
    }
  }

  const move = async (groupId: number) => {
    try {
      await api.moveComic(comic.id, groupId)
      flash('success', 'Comic moved.')
      onRemoved()
    } catch (err) {
      flash('danger', errorMessage(err))
    }
  }

  const remove = async () => {
    if (!window.confirm('Delete comic?')) return
    try {
      await api.deleteComic(comic.id)
      flash('success', 'Comic deleted.')
      onRemoved()
    } catch (err) {
      flash('danger', errorMessage(err))
    }
  }

  return { read, move, remove }
}

function ComicCard(props: ComicProps) {
  const { comic, groups } = props
  const { read, move, remove } = useComicActions(props)
  const [open, setOpen] = useState(false)
  const [target, setTarget] = useState('')

  return (
    <div className="col-6 col-sm-4 col-md-3 col-xl-2 mb-4">
      <Card className="h-100">
        <a href={`/comics/${comic.id}`} onClick={read} title="Read">
          <Thumb pageId={comic.coverPageId} />
        </a>
        <Card.Body className="p-2">
          <h6 className="text-truncate mb-2" title={comic.filename}>
            {comic.name}
          </h6>
          <div className="d-flex justify-content-between align-items-center">
            <div className="btn-group">
              <Link to={`/comics/${comic.id}`} className="btn btn-sm btn-outline-secondary" title="View Pages">
                <Icon name="grid-3x3-gap" />
              </Link>
              <Button
                size="sm"
                variant="outline-secondary"
                onClick={() => setOpen((o) => !o)}
                aria-expanded={open}
                title="Options"
              >
                <Icon name="gear" /> <Icon name={open ? 'caret-up-fill' : 'caret-down-fill'} className="small" />
              </Button>
            </div>
            <ReadState read={comic.read} reading={comic.reading} />
          </div>

          <Collapse in={open}>
            <div>
              <div className="pt-2">
                <Form.Select size="sm" className="mb-2" value={target} onChange={(e) => setTarget(e.target.value)}>
                  <option value="">Move comic...</option>
                  {groups
                    .filter((g) => g.id !== comic.groupId)
                    .map((g) => (
                      <option key={g.id} value={g.id}>
                        {g.name}
                      </option>
                    ))}
                </Form.Select>
                <div className="d-flex gap-2">
                  <Button size="sm" disabled={!target} onClick={() => move(Number(target))}>
                    Move
                  </Button>
                  <Button size="sm" variant="danger" onClick={remove}>
                    Delete
                  </Button>
                </div>
              </div>
            </div>
          </Collapse>
        </Card.Body>
      </Card>
    </div>
  )
}

function ComicRow(props: ComicProps) {
  const { comic, groups } = props
  const { read, move, remove } = useComicActions(props)
  const others = groups.filter((g) => g.id !== comic.groupId)

  return (
    <tr>
      <td className="table-thumb">
        <a href={`/comics/${comic.id}`} onClick={read} title="Read">
          <Thumb pageId={comic.coverPageId} />
        </a>
      </td>
      <td className="text-break">
        <a href={`/comics/${comic.id}`} onClick={read} title={comic.filename} className="table-link">
          {comic.name}
        </a>
      </td>
      <td className="text-end text-nowrap">{comic.pageCount}</td>
      <td className="text-center">
        <ReadState read={comic.read} reading={comic.reading} />
      </td>
      <td className="text-end">
        <Dropdown align="end">
          <Dropdown.Toggle variant="outline-secondary" size="sm" aria-label="Manage comic">
            <Icon name="gear" />
          </Dropdown.Toggle>
          <Dropdown.Menu className="dropdown-menu-scroll">
            <Dropdown.Item as={Link} to={`/comics/${comic.id}`}>
              <Icon name="grid-3x3-gap" className="me-2" />
              View Pages
            </Dropdown.Item>
            {others.length > 0 && (
              <>
                <Dropdown.Divider />
                <Dropdown.Header>Move to</Dropdown.Header>
                {others.map((g) => (
                  <Dropdown.Item key={g.id} onClick={() => move(g.id)} title={g.name}>
                    <Icon name="folder" className="me-2" />
                    {g.name}
                  </Dropdown.Item>
                ))}
              </>
            )}
            <Dropdown.Divider />
            <Dropdown.Item className="text-danger" onClick={remove}>
              <Icon name="trash" className="me-2" />
              Delete Comic
            </Dropdown.Item>
          </Dropdown.Menu>
        </Dropdown>
      </td>
    </tr>
  )
}

function GroupView({ group, comics, groups, reload }: { group: Group; comics: Comic[]; groups: Group[]; reload: () => void }) {
  const navigate = useNavigate()
  const { items, modal } = useGroupMenu({ group, onChanged: reload, onDeleted: () => navigate('/') })
  const [view, setView] = useViewMode('comics-view')

  const nav = (
    <>
      <Nav.Link as={NavLink} to={`/groups/${group.id}`} className="text-truncate nav-title">
        {group.name}
      </Nav.Link>
      <NavDropdown
        title={
          <>
            <Icon name="gear" /> Manage
          </>
        }
      >
        {items}
      </NavDropdown>
    </>
  )

  return (
    <Layout nav={nav} title={group.name}>
      {comics.length > 0 ? (
        <>
          <ViewToggle view={view} onChange={setView} />
          {view === 'tiles' ? (
            <div className="row">
              {comics.map((c) => (
                <ComicCard key={c.id} comic={c} groups={groups} onRemoved={reload} />
              ))}
            </div>
          ) : (
            <Table hover className="align-middle">
              <thead>
                <tr>
                  <th aria-label="Cover" />
                  <th>Name</th>
                  <th className="text-end">Pages</th>
                  <th className="text-center">Status</th>
                  <th aria-label="Manage" />
                </tr>
              </thead>
              <tbody>
                {comics.map((c) => (
                  <ComicRow key={c.id} comic={c} groups={groups} onRemoved={reload} />
                ))}
              </tbody>
            </Table>
          )}
        </>
      ) : (
        <EmptyState title="No Comics">
          <p>
            <UploadButton groupId={group.id} />
          </p>
        </EmptyState>
      )}
      {modal}
    </Layout>
  )
}

export default function GroupPage() {
  const id = Number(useParams().id)
  const { data, error, reload } = useLoad(() => Promise.all([api.group(id), api.groups()]), [id])

  if (error) return <Layout><LoadError error={error} /></Layout>
  if (!data) return <Layout><Spinner /></Layout>

  const [{ group, comics }, groups] = data
  return <GroupView group={group} comics={comics} groups={groups} reload={reload} />
}

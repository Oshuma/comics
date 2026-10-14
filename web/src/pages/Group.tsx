import { type MouseEvent } from 'react'
import { Card, Dropdown, Nav, Table } from 'react-bootstrap'
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
import { useRenameComic } from '../components/RenameComic'

interface ComicProps {
  comic: Comic
  groups: Group[]
  onChanged: () => void
}

function useComicActions({ comic, onChanged }: ComicProps) {
  const flash = useFlash()
  const navigate = useNavigate()
  const rename = useRenameComic(comic, onChanged)

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
      onChanged()
    } catch (err) {
      flash('danger', errorMessage(err))
    }
  }

  const remove = async () => {
    if (!window.confirm('Delete comic?')) return
    try {
      await api.deleteComic(comic.id)
      flash('success', 'Comic deleted.')
      onChanged()
    } catch (err) {
      flash('danger', errorMessage(err))
    }
  }

  return { read, move, remove, rename }
}

// Options menu for a comic, shared by the tile and table views.
function ComicMenu({
  comic,
  groups,
  actions,
  align = 'end',
}: {
  comic: Comic
  groups: Group[]
  actions: ReturnType<typeof useComicActions>
  // Which edge of the gear button the menu lines up with.
  align?: 'start' | 'end'
}) {
  const { move, remove, rename } = actions
  const others = groups.filter((g) => g.id !== comic.groupId)
  return (
    <>
      <Dropdown align={align}>
        <Dropdown.Toggle variant="outline-secondary" size="sm" aria-label="Manage comic">
          <Icon name="gear" />
        </Dropdown.Toggle>
        <Dropdown.Menu className="dropdown-menu-scroll">
          <Dropdown.Item as={Link} to={`/comics/${comic.id}`}>
            <Icon name="grid-3x3-gap" className="me-2" />
            View Pages
          </Dropdown.Item>
          <Dropdown.Item onClick={rename.open}>
            <Icon name="pencil" className="me-2" />
            Rename Comic
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
      {rename.modal}
    </>
  )
}

function ComicCard(props: ComicProps) {
  const { comic, groups } = props
  const actions = useComicActions(props)

  return (
    <div className="col-6 col-sm-4 col-md-3 col-xl-2 mb-4">
      <Card className="h-100">
        <a href={`/comics/${comic.id}`} onClick={actions.read} title="Read">
          <Thumb pageId={comic.coverPageId} />
        </a>
        <Card.Body className="p-2">
          <h6 className="text-truncate mb-2" title={comic.filename}>
            {comic.name}
          </h6>
          <div className="d-flex justify-content-between align-items-center">
            <ComicMenu comic={comic} groups={groups} actions={actions} align="start" />
            <ReadState read={comic.read} reading={comic.reading} />
          </div>
        </Card.Body>
      </Card>
    </div>
  )
}

function ComicRow(props: ComicProps) {
  const { comic, groups } = props
  const actions = useComicActions(props)

  return (
    <tr>
      <td className="table-thumb">
        <a href={`/comics/${comic.id}`} onClick={actions.read} title="Read">
          <Thumb pageId={comic.coverPageId} />
        </a>
      </td>
      <td className="text-break">
        <a href={`/comics/${comic.id}`} onClick={actions.read} title={comic.filename} className="table-link">
          {comic.name}
        </a>
      </td>
      <td className="text-end text-nowrap">{comic.pageCount}</td>
      <td className="text-center">
        <ReadState read={comic.read} reading={comic.reading} />
      </td>
      <td className="text-end">
        <ComicMenu comic={comic} groups={groups} actions={actions} />
      </td>
    </tr>
  )
}

function GroupView({ group, comics, groups, reload }: { group: Group; comics: Comic[]; groups: Group[]; reload: () => void }) {
  const navigate = useNavigate()
  const { items, modal } = useGroupMenu({
    group,
    onChanged: reload,
    onDeleted: () => navigate('/'),
    uploadLabel: 'Upload',
  })
  const [view, setView] = useViewMode('comics-view')

  const nav = (
    <Nav.Link as={NavLink} to={`/groups/${group.id}`} className="text-truncate nav-title">
      {group.name}
    </Nav.Link>
  )

  return (
    <Layout
      nav={nav}
      quickLinks={false}
      manage={items}
      actions={comics.length > 0 && <ViewToggle view={view} onChange={setView} />}
      title={group.name}
    >
      {comics.length > 0 ? (
        <>
          {view === 'tiles' ? (
            <div className="row">
              {comics.map((c) => (
                <ComicCard key={c.id} comic={c} groups={groups} onChanged={reload} />
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
                  <ComicRow key={c.id} comic={c} groups={groups} onChanged={reload} />
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

  if (error)
    return (
      <Layout quickLinks={false}>
        <LoadError error={error} />
      </Layout>
    )
  if (!data)
    return (
      <Layout quickLinks={false}>
        <Spinner />
      </Layout>
    )

  const [{ group, comics }, groups] = data
  return <GroupView group={group} comics={comics} groups={groups} reload={reload} />
}

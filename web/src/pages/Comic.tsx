import { Nav, NavDropdown } from 'react-bootstrap'
import { Link, NavLink, useNavigate, useParams } from 'react-router'
import { api, errorMessage, type ComicDetail } from '../api'
import { useFlash } from '../context'
import Layout from '../components/Layout'
import { Icon, LoadError, Spinner, Thumb, useLoad } from '../components/Common'
import { useRenameComic } from '../components/RenameComic'

function ComicView({ comic, reload }: { comic: ComicDetail; reload: () => void }) {
  const flash = useFlash()
  const navigate = useNavigate()
  const rename = useRenameComic(comic, reload)

  const setRead = async (read: boolean) => {
    try {
      await api.setComicRead(comic.id, read)
      flash('success', read ? 'Comic marked read.' : 'Comic marked unread.')
      reload()
    } catch (err) {
      flash('danger', errorMessage(err))
    }
  }

  const remove = async () => {
    if (!window.confirm('Delete comic?')) return
    try {
      await api.deleteComic(comic.id)
      flash('success', 'Comic deleted.')
      navigate(`/groups/${comic.groupId}`)
    } catch (err) {
      flash('danger', errorMessage(err))
    }
  }

  const nav = (
    <>
      <Nav.Link as={Link} to={`/groups/${comic.groupId}`} className="text-truncate nav-title">
        {comic.groupName}
      </Nav.Link>
      <Nav.Link as={NavLink} to={`/comics/${comic.id}`} className="text-truncate nav-title" title={comic.filename}>
        {comic.name}
      </Nav.Link>
    </>
  )

  const manage = (
    <>
      {!comic.read && (
        <NavDropdown.Item onClick={() => setRead(true)}>
          <Icon name="check-circle" className="me-2" />
          Mark Read
        </NavDropdown.Item>
      )}
      {(comic.read || comic.reading) && (
        <NavDropdown.Item onClick={() => setRead(false)}>
          <Icon name="circle" className="me-2" />
          Mark Unread
        </NavDropdown.Item>
      )}
      <NavDropdown.Divider />
      <NavDropdown.Item onClick={rename.open}>
        <Icon name="pencil" className="me-2" />
        Rename Comic
      </NavDropdown.Item>
      <NavDropdown.Item onClick={remove} className="text-danger">
        <Icon name="trash" className="me-2" />
        Delete Comic
      </NavDropdown.Item>
    </>
  )

  return (
    <Layout nav={nav} quickLinks={false} manage={manage} title={comic.name}>
      <div className="row">
        {comic.pages.map((page) => (
          <div key={page.id} className="col-4 col-sm-3 col-md-2 mb-4">
            <Link to={`/comics/${comic.id}/pages/${page.id}`} className={`page-thumb ${page.read ? 'is-read' : ''}`}>
              <Thumb pageId={page.id} />
              <span className="page-number">{page.number}</span>
            </Link>
          </div>
        ))}
      </div>
      {rename.modal}
    </Layout>
  )
}

export default function ComicPage() {
  const id = Number(useParams().id)
  const { data: comic, error, reload } = useLoad(() => api.comic(id), [id])

  if (error)
    return (
      <Layout quickLinks={false}>
        <LoadError error={error} />
      </Layout>
    )
  if (!comic)
    return (
      <Layout quickLinks={false}>
        <Spinner />
      </Layout>
    )
  return <ComicView comic={comic} reload={reload} />
}

import { Nav, NavDropdown } from 'react-bootstrap'
import { Link, NavLink, useNavigate, useParams } from 'react-router'
import { api, errorMessage } from '../api'
import { useFlash } from '../context'
import Layout from '../components/Layout'
import { Icon, LoadError, Spinner, Thumb, useLoad } from '../components/Common'

export default function ComicPage() {
  const id = Number(useParams().id)
  const flash = useFlash()
  const navigate = useNavigate()
  const { data: comic, error } = useLoad(() => api.comic(id), [id])

  if (error) return <Layout><LoadError error={error} /></Layout>
  if (!comic) return <Layout><Spinner /></Layout>

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
      <NavDropdown
        title={
          <>
            <Icon name="gear" /> Manage
          </>
        }
      >
        <NavDropdown.Item onClick={remove} className="text-danger">
          <Icon name="trash" className="me-2" />
          Delete Comic
        </NavDropdown.Item>
      </NavDropdown>
    </>
  )

  return (
    <Layout nav={nav} title={comic.name}>
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
    </Layout>
  )
}

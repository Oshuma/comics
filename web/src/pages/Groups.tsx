import { useState } from 'react'
import { Card, Dropdown, NavDropdown, Table } from 'react-bootstrap'
import { Link } from 'react-router'
import { api, errorMessage, humanSize, type Group } from '../api'
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

function GroupCard({ group, reload }: { group: Group; reload: () => void }) {
  const { items, modal } = useGroupMenu({ group, onChanged: reload, onDeleted: reload })
  const empty = group.comicCount === 0

  return (
    <div className="col-6 col-sm-4 col-md-3 col-xl-2 mb-4">
      <Card className="h-100">
        <Link to={`/groups/${group.id}`}>
          <Thumb pageId={group.coverPageId} empty={empty} />
        </Link>
        <Card.Body className="p-2">
          <h6 className="text-truncate mb-2" title={group.name}>
            {group.name}
          </h6>
          <div className="d-flex justify-content-between align-items-center">
            <Dropdown>
              <Dropdown.Toggle variant="outline-secondary" size="sm" aria-label="Manage group">
                <Icon name="gear" />
              </Dropdown.Toggle>
              <Dropdown.Menu>{items}</Dropdown.Menu>
            </Dropdown>
            {!empty && <ReadState read={group.read} reading={group.reading} />}
          </div>
        </Card.Body>
      </Card>
      {modal}
    </div>
  )
}

function GroupRow({ group, reload }: { group: Group; reload: () => void }) {
  const { items, modal } = useGroupMenu({ group, onChanged: reload, onDeleted: reload })
  const empty = group.comicCount === 0

  return (
    <tr>
      <td className="table-thumb">
        <Link to={`/groups/${group.id}`}>
          <Thumb pageId={group.coverPageId} empty={empty} />
        </Link>
      </td>
      <td className="text-break">
        <Link to={`/groups/${group.id}`} className="table-link">
          {group.name}
        </Link>
      </td>
      <td className="text-end text-nowrap">{group.comicCount}</td>
      <td className="text-end text-nowrap d-none d-sm-table-cell">{humanSize(group.diskSize)}</td>
      <td className="text-center">{!empty && <ReadState read={group.read} reading={group.reading} />}</td>
      <td className="text-end">
        <Dropdown align="end">
          <Dropdown.Toggle variant="outline-secondary" size="sm" aria-label="Manage group">
            <Icon name="gear" />
          </Dropdown.Toggle>
          <Dropdown.Menu>{items}</Dropdown.Menu>
        </Dropdown>
        {modal}
      </td>
    </tr>
  )
}

export default function Groups() {
  const flash = useFlash()
  const { data: groups, error, reload } = useLoad(api.groups, [])
  const [showEmpty, setShowEmpty] = useState(false)
  const [view, setView] = useViewMode('groups-view')

  const removeRead = async () => {
    if (!window.confirm('Remove all read comics?')) return
    try {
      await api.deleteReadComics()
      flash('success', 'Read comics removed.')
      reload()
    } catch (err) {
      flash('danger', errorMessage(err))
    }
  }

  const nav = (
    <NavDropdown
      title={
        <>
          <Icon name="gear" /> Manage
        </>
      }
    >
      <NavDropdown.Item as={Link} to="/upload">
        <Icon name="cloud-upload" className="me-2" />
        Upload Comics
      </NavDropdown.Item>
      <NavDropdown.Divider />
      <NavDropdown.Item onClick={() => setShowEmpty((v) => !v)}>
        <Icon name={showEmpty ? 'toggle-on' : 'toggle-off'} className="me-2" />
        {showEmpty ? 'Hide' : 'Show'} Empty Groups
      </NavDropdown.Item>
      <NavDropdown.Divider />
      <NavDropdown.Item onClick={removeRead}>
        <Icon name="eye-slash" className="me-2" />
        Remove Read Comics
      </NavDropdown.Item>
    </NavDropdown>
  )

  let content
  if (error) content = <LoadError error={error} />
  else if (!groups) content = <Spinner />
  else if (groups.length === 0)
    content = (
      <EmptyState title="No Comics">
        <p>
          <UploadButton />
        </p>
      </EmptyState>
    )
  else {
    const visible = groups.filter((g) => showEmpty || g.comicCount > 0)
    content = (
      <>
        {visible.length > 0 && <ViewToggle view={view} onChange={setView} />}
        {visible.length > 0 &&
          (view === 'tiles' ? (
            <div className="row">
              {visible.map((g) => (
                <GroupCard key={g.id} group={g} reload={reload} />
              ))}
            </div>
          ) : (
            <Table hover className="align-middle">
              <thead>
                <tr>
                  <th aria-label="Cover" />
                  <th>Name</th>
                  <th className="text-end">Comics</th>
                  <th className="text-end d-none d-sm-table-cell">Size</th>
                  <th className="text-center">Status</th>
                  <th aria-label="Manage" />
                </tr>
              </thead>
              <tbody>
                {visible.map((g) => (
                  <GroupRow key={g.id} group={g} reload={reload} />
                ))}
              </tbody>
            </Table>
          ))}
        {visible.length === 0 && (
          <EmptyState title="No Comics">
            <p>
              <UploadButton />
            </p>
          </EmptyState>
        )}
      </>
    )
  }

  return <Layout nav={nav}>{content}</Layout>
}

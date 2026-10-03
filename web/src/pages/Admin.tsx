import { useState, type FormEvent } from 'react'
import { Badge, Button, Dropdown, Form, Modal, Nav, Table } from 'react-bootstrap'
import { NavLink, useSearchParams } from 'react-router'
import { api, errorMessage, type User } from '../api'
import { useAuth, useFlash } from '../context'
import Layout from '../components/Layout'
import { EntriesInfo, Icon, LoadError, Pagination, Spinner, useLoad } from '../components/Common'

function NewUserModal({ show, onHide, onCreated }: { show: boolean; onHide: () => void; onCreated: () => void }) {
  const flash = useFlash()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [passwordConfirmation, setPasswordConfirmation] = useState('')
  const [admin, setAdmin] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const reset = () => {
    setEmail('')
    setPassword('')
    setPasswordConfirmation('')
    setAdmin(false)
    setError(null)
  }

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    try {
      await api.createUser({ email, password, passwordConfirmation, admin })
      flash('success', 'User added.')
      reset()
      onCreated()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <Modal show={show} onHide={onHide} size="sm" onExited={reset}>
      <Form onSubmit={submit}>
        <Modal.Header closeButton>
          <Modal.Title as="h5">New User</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          {error && <div className="alert alert-danger py-2">{error}</div>}
          <Form.Control
            className="mb-3"
            type="email"
            placeholder="user@example.com"
            required
            autoFocus
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          <Form.Control
            className="mb-3"
            type="password"
            placeholder="Password"
            autoComplete="new-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <Form.Control
            className="mb-3"
            type="password"
            placeholder="Confirm Password"
            autoComplete="new-password"
            required
            value={passwordConfirmation}
            onChange={(e) => setPasswordConfirmation(e.target.value)}
          />
          <Form.Check id="new-user-admin" label="Admin" checked={admin} onChange={(e) => setAdmin(e.target.checked)} />
        </Modal.Body>
        <Modal.Footer>
          <Button type="submit">Add User</Button>
          <Button variant="secondary" onClick={onHide}>
            Cancel
          </Button>
        </Modal.Footer>
      </Form>
    </Modal>
  )
}

function UserActions({ user, isSelf, reload }: { user: User; isSelf: boolean; reload: () => void }) {
  const flash = useFlash()

  const run = async (confirmText: string, action: () => Promise<unknown>, done: string) => {
    if (!window.confirm(confirmText)) return
    try {
      await action()
      flash('success', done)
      reload()
    } catch (err) {
      flash('danger', errorMessage(err))
    }
  }

  return (
    <Dropdown>
      <Dropdown.Toggle variant="outline-secondary" size="sm" aria-label="User actions">
        <Icon name="gear" />
      </Dropdown.Toggle>
      <Dropdown.Menu>
        {user.admin ? (
          <Dropdown.Item
            disabled={isSelf}
            title={isSelf ? "Can't remove yourself as admin." : undefined}
            onClick={() => run('Disable admin?', () => api.setAdmin(user.id, false), 'User is no longer an admin.')}
          >
            <Icon name="person-dash" className="me-2" />
            Disable Admin
          </Dropdown.Item>
        ) : (
          <Dropdown.Item
            onClick={() => run('Enable admin?', () => api.setAdmin(user.id, true), 'User is now an admin.')}
          >
            <Icon name="person-plus" className="me-2" />
            Enable Admin
          </Dropdown.Item>
        )}
        <Dropdown.Divider />
        <Dropdown.Item
          disabled={isSelf}
          className={isSelf ? '' : 'text-danger'}
          title={isSelf ? "Can't delete your own account." : undefined}
          onClick={() =>
            run('Remove this user and their comics?', () => api.deleteUser(user.id), 'User and their comics were removed.')
          }
        >
          <Icon name="trash" className="me-2" />
          Delete User
        </Dropdown.Item>
      </Dropdown.Menu>
    </Dropdown>
  )
}

export default function Admin() {
  const { user: me } = useAuth()
  const [params, setParams] = useSearchParams()
  const page = Math.max(1, Number(params.get('page')) || 1)
  const { data, error, reload } = useLoad(() => api.users(page), [page])
  const [adding, setAdding] = useState(false)

  const nav = (
    <>
      <Nav.Link as={NavLink} to="/admin">
        <Icon name="gear-wide-connected" /> Admin
      </Nav.Link>
      <Nav.Link onClick={() => setAdding(true)}>
        <Icon name="person-plus" /> Add User
      </Nav.Link>
    </>
  )

  return (
    <Layout nav={nav} title="Admin">
      <NewUserModal
        show={adding}
        onHide={() => setAdding(false)}
        onCreated={() => {
          setAdding(false)
          reload()
        }}
      />
      {error ? (
        <LoadError error={error} />
      ) : !data ? (
        <Spinner />
      ) : (
        <>
          <EntriesInfo page={data.page} total={data.total} perPage={data.perPage} />
          <Table hover responsive>
            <thead>
              <tr>
                <th style={{ width: '1%' }}></th>
                <th>Email</th>
                <th>Groups</th>
                <th>Comics</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map((u) => (
                <tr key={u.id}>
                  <td>
                    <UserActions user={u} isSelf={u.id === me?.id} reload={reload} />
                  </td>
                  <td>
                    {u.email}{' '}
                    {u.id === me?.id && <Badge bg="primary">Your Account</Badge>}{' '}
                    {u.admin && <Badge bg="info">Admin</Badge>}
                  </td>
                  <td>{u.groupCount}</td>
                  <td>{u.comicCount}</td>
                </tr>
              ))}
            </tbody>
          </Table>
          <Pagination
            page={data.page}
            total={data.total}
            perPage={data.perPage}
            onChange={(p) => setParams({ page: String(p) })}
          />
        </>
      )}
    </Layout>
  )
}

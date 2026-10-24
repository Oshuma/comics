import { useState, type FormEvent } from 'react'
import { Button, Card, Form, Nav } from 'react-bootstrap'
import { NavLink } from 'react-router'
import { api, errorMessage } from '../api'
import { useAuth, useFlash } from '../context'
import Layout from '../components/Layout'
import { Icon } from '../components/Common'

function ChangePassword() {
  const { user } = useAuth()
  const flash = useFlash()
  const [currentPassword, setCurrentPassword] = useState('')
  const [password, setPassword] = useState('')
  const [passwordConfirmation, setPasswordConfirmation] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await api.changePassword({ currentPassword, password, passwordConfirmation })
      flash('success', 'Password changed.')
      setCurrentPassword('')
      setPassword('')
      setPasswordConfirmation('')
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Card>
      <Card.Header as="h5">Change Password</Card.Header>
      <Card.Body>
        <p className="text-body-secondary">Changing your password signs you out everywhere else.</p>
        {error && <div className="alert alert-danger py-2">{error}</div>}
        <Form onSubmit={submit}>
          {/* Lets password managers associate the new password with this account. */}
          <input type="email" autoComplete="username" value={user?.email ?? ''} readOnly hidden />
          <Form.Group className="mb-3" controlId="currentPassword">
            <Form.Label>Current Password</Form.Label>
            <Form.Control
              type="password"
              autoComplete="current-password"
              required
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
            />
          </Form.Group>
          <Form.Group className="mb-3" controlId="password">
            <Form.Label>New Password</Form.Label>
            <Form.Control
              type="password"
              autoComplete="new-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </Form.Group>
          <Form.Group className="mb-3" controlId="passwordConfirmation">
            <Form.Label>Confirm New Password</Form.Label>
            <Form.Control
              type="password"
              autoComplete="new-password"
              required
              value={passwordConfirmation}
              onChange={(e) => setPasswordConfirmation(e.target.value)}
            />
          </Form.Group>
          <Button type="submit" disabled={busy}>
            Change Password
          </Button>
        </Form>
      </Card.Body>
    </Card>
  )
}

export default function Settings() {
  const nav = (
    <Nav.Link as={NavLink} to="/settings">
      <Icon name="person-gear" /> Settings
    </Nav.Link>
  )

  return (
    <Layout nav={nav} title="Settings">
      <div className="row justify-content-center">
        <div className="col-md-8 col-lg-6">
          <ChangePassword />
        </div>
      </div>
    </Layout>
  )
}

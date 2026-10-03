import { useState, type FormEvent } from 'react'
import { Button, Card, Form } from 'react-bootstrap'
import { Navigate, useNavigate } from 'react-router'
import { api, errorMessage } from '../api'
import { useAuth, useFlash } from '../context'
import { BareLayout } from '../components/Layout'

export default function Setup() {
  const { status, signedIn } = useAuth()
  const flash = useFlash()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [passwordConfirmation, setPasswordConfirmation] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  if (status && !status.setupRequired) return <Navigate to="/" replace />

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      signedIn(await api.setup({ email, password, passwordConfirmation }))
      flash('success', 'Admin user created.')
      navigate('/', { replace: true })
    } catch (err) {
      setError(errorMessage(err))
      setBusy(false)
    }
  }

  return (
    <BareLayout label="Setup">
      <div className="row justify-content-center">
        <div className="col-md-8 col-lg-6">
          <Card>
            <Card.Header as="h5">Create Admin User</Card.Header>
            <Card.Body>
              <p className="text-body-secondary">
                Create an initial admin user. This user can upload and manage comics like any normal user.
              </p>
              {error && <div className="alert alert-danger py-2">{error}</div>}
              <Form onSubmit={submit}>
                <Form.Group className="mb-3" controlId="email">
                  <Form.Label>Email</Form.Label>
                  <Form.Control
                    type="email"
                    placeholder="user@example.com"
                    autoComplete="username"
                    required
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                  />
                </Form.Group>
                <Form.Group className="mb-3" controlId="password">
                  <Form.Label>Password</Form.Label>
                  <Form.Control
                    type="password"
                    placeholder="Password"
                    autoComplete="new-password"
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                  />
                </Form.Group>
                <Form.Group className="mb-3" controlId="passwordConfirmation">
                  <Form.Label>Confirm Password</Form.Label>
                  <Form.Control
                    type="password"
                    placeholder="Confirm Password"
                    autoComplete="new-password"
                    required
                    value={passwordConfirmation}
                    onChange={(e) => setPasswordConfirmation(e.target.value)}
                  />
                </Form.Group>
                <Button type="submit" disabled={busy}>
                  Create
                </Button>
              </Form>
            </Card.Body>
          </Card>
        </div>
      </div>
    </BareLayout>
  )
}

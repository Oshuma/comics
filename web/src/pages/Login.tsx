import { useState, type FormEvent } from 'react'
import { Button, Card, Form } from 'react-bootstrap'
import { Navigate, useLocation, useNavigate } from 'react-router'
import { api, errorMessage } from '../api'
import { useAuth } from '../context'
import { BareLayout } from '../components/Layout'

export default function Login() {
  const { user, signedIn } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  if (user) return <Navigate to="/" replace />

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      signedIn(await api.login(email, password, remember))
      navigate((location.state as { from?: string } | null)?.from || '/', { replace: true })
    } catch (err) {
      setError(errorMessage(err))
      setBusy(false)
    }
  }

  return (
    <BareLayout>
      <div className="row justify-content-center">
        <div className="col-sm-8 col-md-5 col-lg-4">
          <Card>
            <Card.Body>
              <Card.Title className="mb-3">Sign In</Card.Title>
              {error && <div className="alert alert-danger py-2">{error}</div>}
              <Form onSubmit={submit}>
                <Form.Group className="mb-3" controlId="email">
                  <Form.Label>Email</Form.Label>
                  <Form.Control
                    type="email"
                    autoComplete="username"
                    autoFocus
                    required
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                  />
                </Form.Group>
                <Form.Group className="mb-3" controlId="password">
                  <Form.Label>Password</Form.Label>
                  <Form.Control
                    type="password"
                    autoComplete="current-password"
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                  />
                </Form.Group>
                <Form.Check
                  className="mb-3"
                  id="remember"
                  label="Remember Me"
                  checked={remember}
                  onChange={(e) => setRemember(e.target.checked)}
                />
                <Button type="submit" disabled={busy} className="w-100">
                  Login
                </Button>
              </Form>
            </Card.Body>
          </Card>
        </div>
      </div>
    </BareLayout>
  )
}

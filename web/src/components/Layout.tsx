import { useEffect, type ReactNode } from 'react'
import { Container, Nav, Navbar, NavDropdown } from 'react-bootstrap'
import { Link, NavLink, useNavigate } from 'react-router'
import { useAuth } from '../context'
import { Icon } from './Common'

interface Props {
  // Page-specific navbar items.
  nav?: ReactNode
  title?: string
  children: ReactNode
}

export default function Layout({ nav, title, children }: Props) {
  const { user, signOut } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    document.title = title ? `${title} · Comics` : 'Comics'
  }, [title])

  const handleSignOut = async () => {
    await signOut()
    navigate('/login')
  }

  return (
    <>
      <Navbar expand="sm" className="bg-body-tertiary mb-3 border-bottom" collapseOnSelect>
        <Container fluid>
          <Navbar.Brand as={Link} to="/">
            <Icon name="book-half" /> Comics
          </Navbar.Brand>
          <Navbar.Toggle aria-controls="app-nav" />
          <Navbar.Collapse id="app-nav">
            <Nav className="me-auto">{nav}</Nav>
            {user && (
              <Nav>
                <Nav.Link as={NavLink} to="/upload" title="Upload Comics" end>
                  <Icon name="cloud-upload" className="fs-5" />
                  <span className="d-sm-none ms-2">Upload Comics</span>
                </Nav.Link>
                <NavDropdown
                  align="end"
                  title={
                    <>
                      <Icon name="person-circle" className="fs-5" />
                      <span className="d-sm-none ms-2">Account</span>
                    </>
                  }
                >
                  <NavDropdown.Header>{user.email}</NavDropdown.Header>
                  <NavDropdown.Item as={Link} to="/history">
                    <Icon name="clock-history" className="me-2" />
                    History
                  </NavDropdown.Item>
                  <NavDropdown.Item as={Link} to="/stats">
                    <Icon name="bar-chart" className="me-2" />
                    Stats
                  </NavDropdown.Item>
                  {user.admin && (
                    <>
                      <NavDropdown.Divider />
                      <NavDropdown.Item as={Link} to="/admin">
                        <Icon name="gear-wide-connected" className="me-2" />
                        Admin
                      </NavDropdown.Item>
                    </>
                  )}
                  <NavDropdown.Divider />
                  <NavDropdown.Item onClick={handleSignOut}>
                    <Icon name="box-arrow-right" className="me-2" />
                    Sign Out
                  </NavDropdown.Item>
                </NavDropdown>
              </Nav>
            )}
          </Navbar.Collapse>
        </Container>
      </Navbar>
      <Container fluid className="pb-4">
        {children}
      </Container>
    </>
  )
}

// Simple page layout used before sign-in (login/setup).
export function BareLayout({ label, children }: { label?: string; children: ReactNode }) {
  return (
    <>
      <Navbar className="bg-body-tertiary mb-4 border-bottom">
        <Container fluid>
          <Navbar.Brand as={Link} to="/">
            <Icon name="book-half" /> Comics
          </Navbar.Brand>
          {label && (
            <Navbar.Text className="me-auto">
              <Icon name="gear" /> {label}
            </Navbar.Text>
          )}
        </Container>
      </Navbar>
      <Container>{children}</Container>
    </>
  )
}

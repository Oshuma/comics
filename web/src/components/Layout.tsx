import { useEffect, type ReactNode } from 'react'
import { Container, Nav, Navbar, NavDropdown } from 'react-bootstrap'
import { Link, NavLink, useNavigate } from 'react-router'
import { useAuth } from '../context'
import { useTheme } from '../theme'
import { Icon } from './Common'
import { ImportsMenu } from './ImportsMenu'

function ThemeToggle() {
  const { theme, toggle } = useTheme()
  const label = theme === 'dark' ? 'Light Mode' : 'Dark Mode'
  return (
    <button type="button" className="btn btn-link nav-link text-start" onClick={toggle} title={label} aria-label={label}>
      <Icon name={theme === 'dark' ? 'sun' : 'moon-stars'} className="fs-5" />
      <span className="d-sm-none ms-2">{label}</span>
    </button>
  )
}

interface Props {
  // Page-specific navbar items.
  nav?: ReactNode
  title?: string
  children: ReactNode
}

export default function Layout({ nav, title, children }: Props) {
  const { status, user, signOut } = useAuth()
  const navigate = useNavigate()
  const { theme, toggle: toggleTheme } = useTheme()

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
            <Nav>
              {user && (
                <>
                  <ImportsMenu />
                  <Nav.Link as={NavLink} to="/search" title="Search archive.org">
                    <Icon name="search" className="fs-5" />
                    <span className="d-sm-none ms-2">Search archive.org</span>
                  </Nav.Link>
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
                    <NavDropdown.Item as="button" onClick={toggleTheme}>
                      <Icon name={theme === 'dark' ? 'sun' : 'moon-stars'} className="me-2" />
                      {theme === 'dark' ? 'Light Mode' : 'Dark Mode'}
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
                    {status?.version && (
                      <>
                        <NavDropdown.Divider />
                        <NavDropdown.ItemText className="small text-body-secondary text-nowrap">
                          v{status.version.split('+')[0]}
                        </NavDropdown.ItemText>
                      </>
                    )}
                  </NavDropdown>
                </>
              )}
            </Nav>
          </Navbar.Collapse>
        </Container>
      </Navbar>
      <Container fluid className="pb-4">
        {children}
      </Container>
    </>
  )
}

// Simple page layout used before sign-in (login/setup), which has no user menu
// for the theme setting.
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
          <Nav className="ms-auto">
            <ThemeToggle />
          </Nav>
        </Container>
      </Navbar>
      <Container>{children}</Container>
    </>
  )
}

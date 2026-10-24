import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, Navigate, Outlet, RouterProvider, useLocation } from 'react-router'
import 'bootstrap/dist/css/bootstrap.min.css'
import 'bootstrap-icons/font/bootstrap-icons.css'
import './styles.css'

import { AuthProvider, FlashProvider, useAuth } from './context'
import { ImportsProvider } from './components/Imports'
import { UploadsProvider } from './components/Uploads'
import { Spinner } from './components/Common'
import Login from './pages/Login'
import Setup from './pages/Setup'
import Groups from './pages/Groups'
import GroupPage from './pages/Group'
import ComicPage from './pages/Comic'
import Reader from './pages/Reader'
import Upload from './pages/Upload'
import Search from './pages/Search'
import HistoryPage from './pages/History'
import Stats from './pages/Stats'
import Settings from './pages/Settings'
import Admin from './pages/Admin'

// Redirects to setup or login as needed before rendering app routes.
function Gate({ requireUser, requireAdmin }: { requireUser?: boolean; requireAdmin?: boolean }) {
  const { status, user } = useAuth()
  const location = useLocation()

  if (!status) return <Spinner />
  if (status.setupRequired && location.pathname !== '/setup') return <Navigate to="/setup" replace />
  if (requireUser && !user) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  if (requireAdmin && !user?.admin) return <Navigate to="/" replace />
  return <Outlet />
}

const router = createBrowserRouter([
  {
    element: <Gate />,
    children: [
      { path: '/setup', element: <Setup /> },
      { path: '/login', element: <Login /> },
    ],
  },
  {
    element: <Gate requireUser />,
    children: [
      { path: '/', element: <Groups /> },
      { path: '/groups/:id', element: <GroupPage /> },
      { path: '/comics/:id', element: <ComicPage /> },
      { path: '/comics/:id/pages/:pageId', element: <Reader /> },
      { path: '/upload', element: <Upload /> },
      { path: '/search', element: <Search /> },
      { path: '/history', element: <HistoryPage /> },
      { path: '/stats', element: <Stats /> },
      { path: '/settings', element: <Settings /> },
    ],
  },
  {
    element: <Gate requireUser requireAdmin />,
    children: [{ path: '/admin', element: <Admin /> }],
  },
  { path: '*', element: <Navigate to="/" replace /> },
])

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AuthProvider>
      <FlashProvider>
        <ImportsProvider>
          <UploadsProvider>
            <RouterProvider router={router} />
          </UploadsProvider>
        </ImportsProvider>
      </FlashProvider>
    </AuthProvider>
  </StrictMode>,
)

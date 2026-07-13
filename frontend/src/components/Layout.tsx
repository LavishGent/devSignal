import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'

interface NavEntry {
  to: string
  label: string
}

const SECTIONS: { title: string; items: NavEntry[] }[] = [
  {
    title: 'Overview',
    items: [
      { to: '/', label: 'Dashboard' },
      { to: '/search', label: 'Search' },
      { to: '/reports', label: 'Reports' },
    ],
  },
  {
    title: 'Job search',
    items: [
      { to: '/applications', label: 'Applications' },
      { to: '/interviews', label: 'Interviews' },
      { to: '/contacts', label: 'Contacts' },
    ],
  },
  {
    title: 'Development',
    items: [
      { to: '/skills', label: 'Skills' },
      { to: '/study-sessions', label: 'Study sessions' },
      { to: '/certifications', label: 'Certifications' },
      { to: '/projects', label: 'Projects' },
    ],
  },
  {
    title: 'Planning',
    items: [{ to: '/goals', label: 'Goals' }],
  },
]

// Layout renders the sidebar shell around every authenticated page.
export default function Layout() {
  const { user, logout } = useAuth()

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="sidebar-brand">
          <span className="logo">DS</span>
          DevSignal
        </div>
        <nav className="sidebar-nav">
          {SECTIONS.map((section) => (
            <div key={section.title}>
              <div className="nav-section">{section.title}</div>
              {section.items.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.to === '/'}
                  className={({ isActive }) => 'nav-item' + (isActive ? ' active' : '')}
                >
                  {item.label}
                </NavLink>
              ))}
            </div>
          ))}
        </nav>
        <div className="sidebar-footer">
          <div className="sidebar-user">
            <div className="name">{user?.displayName}</div>
            <div className="email">{user?.email}</div>
          </div>
          <button className="btn btn-ghost btn-sm" onClick={() => void logout()}>
            Log out
          </button>
        </div>
      </aside>
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}

import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useAuth } from '../App'

const I = {
  home: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M3 10.5 12 3l9 7.5M5 9.5V21h14V9.5" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  chat: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M21 11.5c0 4.1-4 7.5-9 7.5-1 0-2-.1-2.9-.4L4 20l1.2-3.2C3.8 15.4 3 13.5 3 11.5 3 7.4 7 4 12 4s9 3.4 9 7.5z" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  target: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><circle cx="12" cy="12" r="8"/><circle cx="12" cy="12" r="4"/><circle cx="12" cy="12" r="0.5"/></svg>,
  route: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M4 18h4a3 3 0 0 0 3-3V9a3 3 0 0 1 3-3h6m0 0-3-3m3 3-3 3M4 18l3-3m-3 3 3 3" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  widget: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><rect x="3.5" y="3.5" width="17" height="17" rx="4"/><path d="M8 14.5c1 1 2.4 1.5 4 1.5s3-.5 4-1.5" strokeLinecap="round"/></svg>,
  code: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="m8 8-4.5 4L8 16m8-8 4.5 4L16 16m-2.5-11-3 14" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  gear: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><circle cx="12" cy="12" r="3.2"/><path d="M12 3v2.2m0 13.6V21m9-9h-2.2M5.2 12H3m14.8-6.8-1.6 1.6M7.8 16.2l-1.6 1.6m0-11.6 1.6 1.6m8.4 8.4 1.6 1.6" strokeLinecap="round"/></svg>,
  brand: <svg viewBox="0 0 256 256" fill="currentColor"><path d="M138,128a10,10,0,1,1-10-10A10,10,0,0,1,138,128ZM84,118a10,10,0,1,0,10,10A10,10,0,0,0,84,118Zm88,0a10,10,0,1,0,10,10A10,10,0,0,0,172,118Zm58,10A102,102,0,0,1,79.31,217.65L44.44,229.27a14,14,0,0,1-17.71-17.71l11.62-34.87A102,102,0,1,1,230,128Zm-12,0A90,90,0,1,0,50.08,173.06a6,6,0,0,1,.5,4.91L38.12,215.35a2,2,0,0,0,2.53,2.53L78,205.42a6.2,6.2,0,0,1,1.9-.31,6.09,6.09,0,0,1,3,.81A90,90,0,0,0,218,128Z"/></svg>,
}

export default function Layout() {
  const { account, setAccount } = useAuth()
  const nav = useNavigate()

  async function logout() {
    await api.post('/api/auth/logout')
    setAccount(null)
    nav('/login')
  }

  return (
    <div className="app">
      <a className="skip-link" href="#dashboard-main">Skip to main content</a>
      <aside className="sidebar">
        <div className="brand"><span className="brand-dot">{I.brand}</span><span>Lead Qualifier</span></div>
        <nav className="sidebar-nav" aria-label="Dashboard navigation">
          <NavLink to="/" end className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}>{I.home}<span>Overview</span></NavLink>
          <NavLink to="/conversations" className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}>{I.chat}<span>Conversations</span></NavLink>
          <div className="nav-section">Configure</div>
          <NavLink to="/settings/icp" className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}>{I.target}<span>Ideal customer</span></NavLink>
          <NavLink to="/settings/routing" className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}>{I.route}<span>Lead routing</span></NavLink>
          <NavLink to="/settings/widget" className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}>{I.widget}<span>Widget design</span></NavLink>
          <NavLink to="/settings/install" className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}>{I.code}<span>Install &amp; API</span></NavLink>
          <NavLink to="/settings/account" className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}>{I.gear}<span>Plan &amp; account</span></NavLink>
        </nav>
        <div className="sidebar-foot">
          <b>{account?.name}</b>
          {account?.email}
          <br />
          <button className="logout-btn" onClick={logout}>Sign out</button>
        </div>
      </aside>
      <main className="main" id="dashboard-main" tabIndex={-1}>
        <Outlet />
      </main>
    </div>
  )
}

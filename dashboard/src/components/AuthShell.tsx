import type { ReactNode } from 'react'

const ChatIcon = (
  <svg viewBox="0 0 256 256" fill="currentColor" aria-hidden="true">
    <path d="M138,128a10,10,0,1,1-10-10A10,10,0,0,1,138,128ZM84,118a10,10,0,1,0,10,10A10,10,0,0,0,84,118Zm88,0a10,10,0,1,0,10,10A10,10,0,0,0,172,118Zm58,10A102,102,0,0,1,79.31,217.65L44.44,229.27a14,14,0,0,1-17.71-17.71l11.62-34.87A102,102,0,1,1,230,128Zm-12,0A90,90,0,1,0,50.08,173.06a6,6,0,0,1,.5,4.91L38.12,215.35a2,2,0,0,0,2.53,2.53L78,205.42a6.2,6.2,0,0,1,1.9-.31,6.09,6.09,0,0,1,3,.81A90,90,0,0,0,218,128Z" />
  </svg>
)

const Check = (
  <svg viewBox="0 0 256 256" fill="currentColor" aria-hidden="true">
    <path d="M228.24,76.24l-128,128a6,6,0,0,1-8.48,0l-56-56a6,6,0,0,1,8.48-8.48L96,191.51,219.76,67.76a6,6,0,0,1,8.48,8.48Z" />
  </svg>
)

const POINTS = [
  ['Qualifies while you sleep', 'Budget, authority, need and timeline — asked naturally, scored 0–100.'],
  ['Only real leads reach you', 'Email, Slack or your CRM. Full transcript attached, every time.'],
  ['You stay in control', 'See every chat, override any verdict — the AI learns from you.'],
]

export default function AuthShell({ children }: { children: ReactNode }) {
  return (
    <div className="auth-wrap">
      <div className="auth-grid fade-up">
        <aside className="auth-side">
          <div className="auth-side-inner">
            <div className="auth-brand">
              <span className="auth-brand-dot">{ChatIcon}</span>
              Lead Qualifier
            </div>
            <h2>Turn website visitors into qualified leads.</h2>
            <ul className="auth-points">
              {POINTS.map(([title, sub]) => (
                <li key={title}>
                  <span className="auth-check">{Check}</span>
                  <span>
                    <b>{title}</b>
                    <small>{sub}</small>
                  </span>
                </li>
              ))}
            </ul>
            <div className="auth-side-foot">Live in ~14 minutes · built for small B2B teams in Canada &amp; the US</div>
          </div>
        </aside>
        <div className="shell auth-card">
          <div className="card auth-card-inner">
            <div className="auth-brand mobile-only">
              <span className="auth-brand-dot">{ChatIcon}</span>
              Lead Qualifier
            </div>
            {children}
          </div>
        </div>
      </div>
    </div>
  )
}

import { Link } from 'react-router'
import { ROUTES } from '@/constants/routes'
import './404.css'

export function NotFound() {
  return (
    <main className="page not-found">
      <h1 className="page__title">404</h1>
      <p className="state">this page does not exist.</p>
      <p>
        <Link to={ROUTES.DASHBOARD}>back to dashboard</Link>
      </p>
    </main>
  )
}

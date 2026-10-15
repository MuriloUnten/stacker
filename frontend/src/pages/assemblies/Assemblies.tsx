import { useEffect, useState } from 'react'
import { Link } from 'react-router'
import type { Assembly } from '@/models/items'
import { ROUTES } from '@/constants/routes'
import { listAssemblies } from '@/services/items'
import { formatApiError } from '@/utils/errors'
import { ItemCard } from '@/components/item/ItemCard'
import './Assemblies.css'

export function Assemblies() {
  const [assemblies, setAssemblies] = useState<Assembly[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    const loadAssemblies = async () => {
      const result = await listAssemblies()
      if (cancelled) {
        return
      }
      if (result.success) {
        setAssemblies(result.data)
        setError(null)
      } else {
        setError(formatApiError(result.error))
      }
      setLoading(false)
    }

    void loadAssemblies()
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <main className="page">
      <header className="page__header">
        <h1 className="page__title">assemblies</h1>
        <div className="page__actions">
          <Link className="assemblies__link" to={ROUTES.DASHBOARD}>
            back to dashboard
          </Link>
        </div>
      </header>

      {error ? <p className="error">{error}</p> : null}

      {loading ? (
        <p className="state">loading assemblies…</p>
      ) : assemblies.length === 0 ? (
        <p className="state">no assemblies yet.</p>
      ) : (
        <div className="assemblies__grid">
          {assemblies.map((assembly) => (
            <ItemCard key={assembly.id} item={assembly} />
          ))}
        </div>
      )}
    </main>
  )
}

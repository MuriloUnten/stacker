import { useEffect, useState } from 'react'
import { Link } from 'react-router'
import type { Item } from '@/models/items'
import { ROUTES } from '@/constants/routes'
import { listItems } from '@/services/items'
import { formatApiError } from '@/utils/errors'
import { ItemCard } from '@/components/item/ItemCard'
import { ComponentForm } from '@/components/forms/ComponentForm'
import { AssemblyForm } from '@/components/forms/AssemblyForm'
import './Dashboard.css'

export function Dashboard() {
  const [items, setItems] = useState<Item[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [showComponentForm, setShowComponentForm] = useState(false)
  const [showAssemblyForm, setShowAssemblyForm] = useState(false)

  useEffect(() => {
    let cancelled = false
    const loadItems = async () => {
      setLoading(true)
      const result = await listItems()
      if (cancelled) {
        return
      }
      if (result.success) {
        setItems(result.data)
        setError(null)
      } else {
        setError(formatApiError(result.error))
      }
      setLoading(false)
    }

    void loadItems()
    return () => {
      cancelled = true
    }
  }, [reloadKey])

  const refresh = () => {
    setShowComponentForm(false)
    setShowAssemblyForm(false)
    setReloadKey((key) => key + 1)
  }

  return (
    <main className="page">
      <header className="page__header">
        <h1 className="page__title">dashboard</h1>
        <div className="page__actions">
          <Link className="dashboard__link" to={ROUTES.ASSEMBLIES}>
            all assemblies
          </Link>
          <button
            type="button"
            onClick={() => {
              setShowAssemblyForm(false)
              setShowComponentForm(true)
            }}
          >
            new component
          </button>
          <button
            type="button"
            onClick={() => {
              setShowComponentForm(false)
              setShowAssemblyForm(true)
            }}
          >
            new assembly
          </button>
        </div>
      </header>

      {showComponentForm ? (
        <ComponentForm onCreated={refresh} onCancel={() => setShowComponentForm(false)} />
      ) : null}

      {showAssemblyForm ? (
        <AssemblyForm onCreated={refresh} onCancel={() => setShowAssemblyForm(false)} />
      ) : null}

      {error ? <p className="error">{error}</p> : null}

      {loading ? (
        <p className="state">loading items…</p>
      ) : items.length === 0 ? (
        <p className="state">no items yet.</p>
      ) : (
        <div className="dashboard__grid">
          {items.map((item) => (
            <ItemCard key={item.id} item={item} />
          ))}
        </div>
      )}
    </main>
  )
}

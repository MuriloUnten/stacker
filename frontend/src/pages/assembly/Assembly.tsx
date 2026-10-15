import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router'
import type { Assembly, AssemblyNode, BaseItemVersion } from '@/models/items'
import { ROUTES } from '@/constants/routes'
import {
  deprecateVersion,
  getAssembly,
  getBillOfMaterials,
  listAssemblyVersions,
  publishVersion,
} from '@/services/items'
import { formatApiError } from '@/utils/errors'
import { itemThumbnailUrl } from '@/utils/thumbnail'
import { BomTree } from '@/components/bom/BomTree'
import { VersionForm } from '@/components/forms/VersionForm'
import './Assembly.css'

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString()
}

export function Assembly() {
  const { assemblyId } = useParams()
  const id = Number(assemblyId)
  const validId = Number.isInteger(id) && id > 0

  const [assembly, setAssembly] = useState<Assembly | null>(null)
  const [versions, setVersions] = useState<BaseItemVersion[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [showVersionForm, setShowVersionForm] = useState(false)
  const [bomVersionId, setBomVersionId] = useState<number | null>(null)
  const [bom, setBom] = useState<AssemblyNode | null>(null)
  const [bomError, setBomError] = useState<string | null>(null)

  useEffect(() => {
    if (!validId) {
      return
    }

    let cancelled = false
    const load = async () => {
      setLoading(true)
      setBom(null)
      setBomVersionId(null)
      setBomError(null)

      const [assemblyResult, versionsResult] = await Promise.all([
        getAssembly(id),
        listAssemblyVersions(id),
      ])
      if (cancelled) {
        return
      }

      if (assemblyResult.success) {
        setAssembly(assemblyResult.data)
        setError(null)
      } else {
        setError(formatApiError(assemblyResult.error))
      }

      if (versionsResult.success) {
        setVersions(versionsResult.data)
      } else {
        setError(formatApiError(versionsResult.error))
      }
      setLoading(false)
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [id, validId, reloadKey])

  const toggleBom = async (versionId: number) => {
    if (bomVersionId === versionId) {
      setBomVersionId(null)
      setBom(null)
      setBomError(null)
      return
    }

    setBomVersionId(versionId)
    setBom(null)
    setBomError(null)

    const result = await getBillOfMaterials(versionId)
    if (result.success) {
      setBom(result.data)
    } else {
      setBomError(formatApiError(result.error))
    }
  }

  const handlePublish = async (versionId: number) => {
    const result = await publishVersion(versionId)
    if (!result.success) {
      setError(formatApiError(result.error))
      return
    }
    setReloadKey((key) => key + 1)
  }

  const handleDeprecate = async (versionId: number) => {
    const result = await deprecateVersion(versionId)
    if (!result.success) {
      setError(formatApiError(result.error))
      return
    }
    setReloadKey((key) => key + 1)
  }

  return (
    <main className="page">
      <header className="page__header">
        <h1 className="page__title">{assembly ? assembly.name : 'assembly'}</h1>
        <div className="page__actions">
          <Link className="assembly__link" to={ROUTES.ASSEMBLIES}>
            all assemblies
          </Link>
          <Link className="assembly__link" to={ROUTES.DASHBOARD}>
            dashboard
          </Link>
        </div>
      </header>

      {!validId ? <p className="error">invalid assembly id</p> : null}

      {validId && error ? <p className="error">{error}</p> : null}

      {validId && loading ? <p className="state">loading assembly…</p> : null}

      {validId && !loading && assembly ? (
        <>
          <section className="assembly__summary panel">
            <div className="assembly__thumb">
              {assembly.thumbnail_id === null ? (
                <span className="item-card__thumb-empty">no image</span>
              ) : (
                <img src={itemThumbnailUrl(assembly.id)} alt={assembly.name} />
              )}
            </div>
            <div className="assembly__details">
              <p className="assembly__sku">{assembly.sku}</p>
              {assembly.description ? (
                <p className="assembly__description">{assembly.description}</p>
              ) : null}
              <p className="assembly__inventory">
                stock {assembly.inventory.available} {assembly.inventory.unit} · reserved{' '}
                {assembly.inventory.reserved}
              </p>
              <p className="assembly__current">
                current version:{' '}
                {assembly.current_version
                  ? `${assembly.current_version.version_code} (${assembly.current_version.status})`
                  : '—'}
              </p>
            </div>
          </section>

          <section className="assembly__versions">
            <div className="assembly__versions-head">
              <h2 className="assembly__section-title">versions</h2>
              <button
                type="button"
                onClick={() => setShowVersionForm(true)}
                disabled={showVersionForm}
              >
                new version
              </button>
            </div>

            {showVersionForm ? (
              <VersionForm
                assemblyId={id}
                onCreated={() => {
                  setShowVersionForm(false)
                  setReloadKey((key) => key + 1)
                }}
                onCancel={() => setShowVersionForm(false)}
              />
            ) : null}

            {versions.length === 0 ? (
              <p className="state">no versions yet.</p>
            ) : (
              <ul className="assembly__version-list">
                {versions.map((version) => (
                  <li className="assembly__version" key={version.id}>
                    <div className="assembly__version-main">
                      <span className="assembly__version-code">
                        {version.version_code}
                      </span>
                      <span className={`status status--${version.status}`}>
                        {version.status}
                      </span>
                      <span className="assembly__version-notes">{version.notes}</span>
                    </div>
                    <div className="assembly__version-meta">
                      <span>created {formatDateTime(version.created_at)}</span>
                      <span>
                        {version.published_at
                          ? `published ${formatDateTime(version.published_at)}`
                          : 'not published'}
                      </span>
                    </div>
                    <div className="assembly__version-actions">
                      <button type="button" onClick={() => toggleBom(version.id)}>
                        {bomVersionId === version.id ? 'hide bom' : 'view bom'}
                      </button>
                      {version.status === 'draft' ? (
                        <button type="button" onClick={() => handlePublish(version.id)}>
                          publish
                        </button>
                      ) : null}
                      {version.status === 'published' ? (
                        <button
                          type="button"
                          onClick={() => handleDeprecate(version.id)}
                        >
                          deprecate
                        </button>
                      ) : null}
                    </div>

                    {bomVersionId === version.id ? (
                      <div className="assembly__bom">
                        {bomError ? (
                          <p className="error">{bomError}</p>
                        ) : bom === null ? (
                          <p className="state">loading bom…</p>
                        ) : (
                          <BomTree nodes={bom.children ?? []} />
                        )}
                      </div>
                    ) : null}
                  </li>
                ))}
              </ul>
            )}
          </section>
        </>
      ) : null}
    </main>
  )
}

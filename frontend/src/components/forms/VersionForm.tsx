import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import type { Item } from '@/models/items'
import { createItemVersionParamsSchema } from '@/models/items'
import { createAssemblyVersion, listItems } from '@/services/items'
import { formatApiError, formatZodError } from '@/utils/errors'
import { emptyDraft, toAssemblyChildren } from '@/utils/bom'
import type { BomChildDraft } from '@/utils/bom'
import { BomLinesEditor } from '@/components/bom/BomLinesEditor'
import './VersionForm.css'

interface VersionFormProps {
  assemblyId: number
  onCreated: () => void
  onCancel: () => void
}

export function VersionForm({ assemblyId, onCreated, onCancel }: VersionFormProps) {
  const [versionCode, setVersionCode] = useState('')
  const [versionNotes, setVersionNotes] = useState('')
  const [children, setChildren] = useState<BomChildDraft[]>([emptyDraft()])
  const [items, setItems] = useState<Item[]>([])
  const [itemsError, setItemsError] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    let cancelled = false
    const loadItems = async () => {
      const result = await listItems()
      if (cancelled) {
        return
      }
      if (result.success) {
        setItems(result.data)
      } else {
        setItemsError(formatApiError(result.error))
      }
    }

    void loadItems()
    return () => {
      cancelled = true
    }
  }, [])

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError(null)

    const parsed = createItemVersionParamsSchema.safeParse({
      version_code: versionCode,
      version_notes: versionNotes,
      children: toAssemblyChildren(children, items),
    })
    if (!parsed.success) {
      setError(formatZodError(parsed.error))
      return
    }

    setSubmitting(true)
    const result = await createAssemblyVersion(assemblyId, parsed.data)
    if (!result.success) {
      setError(formatApiError(result.error))
      setSubmitting(false)
      return
    }

    onCreated()
  }

  return (
    <form className="version-form panel" onSubmit={handleSubmit}>
      <h2 className="panel__title">new version</h2>

      <div className="version-form__row">
        <label className="field">
          <span className="field__label">version code</span>
          <input
            value={versionCode}
            onChange={(event) => setVersionCode(event.target.value)}
            disabled={submitting}
          />
        </label>

        <label className="field">
          <span className="field__label">version notes</span>
          <input
            value={versionNotes}
            onChange={(event) => setVersionNotes(event.target.value)}
            disabled={submitting}
          />
        </label>
      </div>

      {itemsError ? <p className="form-error">{itemsError}</p> : null}

      <BomLinesEditor
        items={items}
        value={children}
        disabled={submitting || itemsError !== null}
        onChange={setChildren}
      />

      {error ? <p className="form-error">{error}</p> : null}

      <div className="form-actions">
        <button type="submit" disabled={submitting || itemsError !== null}>
          {submitting ? 'creating…' : 'create version'}
        </button>
        <button type="button" onClick={onCancel} disabled={submitting}>
          cancel
        </button>
      </div>
    </form>
  )
}

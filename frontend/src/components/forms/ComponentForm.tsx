import { useState } from 'react'
import type { FormEvent } from 'react'
import { createComponentParamsSchema } from '@/models/items'
import { createComponent } from '@/services/items'
import { formatApiError, formatZodError } from '@/utils/errors'
import { BaseItemFields } from './BaseItemFields'
import type { BaseItemFieldsValue } from './BaseItemFields'
import './ComponentForm.css'

interface ComponentFormProps {
  onCreated: () => void
  onCancel: () => void
}

const initialState: BaseItemFieldsValue = {
  name: '',
  sku: '',
  description: '',
  unit: 'each',
  available: '0',
}

export function ComponentForm({ onCreated, onCancel }: ComponentFormProps) {
  const [form, setForm] = useState<BaseItemFieldsValue>(initialState)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError(null)

    const parsed = createComponentParamsSchema.safeParse(form)
    if (!parsed.success) {
      setError(formatZodError(parsed.error))
      return
    }

    setSubmitting(true)
    const result = await createComponent(parsed.data)
    if (!result.success) {
      setError(formatApiError(result.error))
      setSubmitting(false)
      return
    }

    onCreated()
  }

  return (
    <form className="component-form panel" onSubmit={handleSubmit}>
      <h2 className="panel__title">new component</h2>

      <BaseItemFields
        value={form}
        disabled={submitting}
        onChange={(patch) => setForm((current) => ({ ...current, ...patch }))}
      />

      {error ? <p className="form-error">{error}</p> : null}

      <div className="form-actions">
        <button type="submit" disabled={submitting}>
          {submitting ? 'creating…' : 'create'}
        </button>
        <button type="button" onClick={onCancel} disabled={submitting}>
          cancel
        </button>
      </div>
    </form>
  )
}

import { z } from 'zod'
import type { ApiError } from '@/models/error'

export function formatApiError(error: ApiError): string {
  return error.field ? `${error.msg} (${error.field})` : error.msg
}

export function formatZodError(error: z.ZodError): string {
  return z.prettifyError(error)
}

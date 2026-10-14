import { z } from 'zod';

export const errorKindSchema = z.enum([
    "front_fetch",
    "front_json_parse",
	"invalid",
	"io",
	"validation",
	"exist",
	"not_exist",
	"internal",
	"database",
	// TODO add this back whenever authentication is implemented
    // "unauthenticated",
    // "forbidden",
]);
export type ErrorKind = z.infer<typeof errorKindSchema>;

export const ApiErrorSchema = z.object({
    kind: errorKindSchema,
    msg: z.string(),
    field: z.string().optional(),
});
export type ApiError = z.infer<typeof ApiErrorSchema>;

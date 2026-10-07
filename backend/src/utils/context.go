package utils

// Context keys shared between middleware and controllers/handlers.
// Kept in utils (not middleware) to avoid an import cycle —
// middleware imports utils for HashToken, so utils must not import middleware.

type ctxKey string

const UserIDKey ctxKey = "user_id"
const SessionIDKey ctxKey = "session_id"

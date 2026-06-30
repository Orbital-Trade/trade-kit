package server

// Response helpers moved to api package to break import cycles.
// Server package uses api.WriteJSON/WriteError/WriteOK directly.

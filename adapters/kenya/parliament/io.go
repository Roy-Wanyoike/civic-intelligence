// Package parliament — internal helper alias for io.ReadAll so the adapter
// files don't need to import "io" everywhere. Production code can replace this
// with a proper streaming reader if memory pressure becomes an issue.
package parliament

import "io"

func io_ReadAll(r io.Reader) ([]byte, error) { return io.ReadAll(r) }

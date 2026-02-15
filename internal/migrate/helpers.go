package migrate

import "os"

// writeFile writes data to the named file, creating it with 0644 permissions.
func writeFile(name string, data []byte) error {
	return os.WriteFile(name, data, 0644)
}

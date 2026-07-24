// Package secret stores and retrieves sensitive values (e.g. AI provider API
// keys) in the OS secret store via the system keyring. On Linux this requires a
// running Secret Service provider (gnome-keyring, KWallet, …); when none is
// available the operations return an error and callers should fall back to
// "$ENV" references or literal config values.
package secret

import "github.com/zalando/go-keyring"

// service namespaces all seshat secrets in the OS keyring.
const service = "seshat"

// Set stores secret under name in the OS keyring.
func Set(name, secret string) error { return keyring.Set(service, name, secret) }

// Get retrieves the secret stored under name.
func Get(name string) (string, error) { return keyring.Get(service, name) }

// Delete removes the secret stored under name. A missing entry is not an error.
func Delete(name string) error {
	err := keyring.Delete(service, name)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}

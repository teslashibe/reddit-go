package redditmessenger

// Identity holds the authenticated user's basic info from /api/v1/me.
type Identity struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// Me returns the authenticated user's identity.
// Useful for verifying the token is valid.
func (m *Messenger) Me() (*Identity, error) {
	var raw struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	}
	if err := m.oauthGetJSON("/api/v1/me", &raw); err != nil {
		return nil, err
	}
	return &Identity{Name: raw.Name, ID: raw.ID}, nil
}

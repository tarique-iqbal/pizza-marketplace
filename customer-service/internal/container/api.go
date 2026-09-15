package container

type APIContainer struct {
	*Shared
}

func NewAPIContainer() (*APIContainer, error) {
	base, err := NewShared()
	if err != nil {
		return nil, err
	}

	return &APIContainer{
		Shared: base,
	}, nil
}

func (c *APIContainer) Close() {
	if c.DB != nil {
		db, err := c.DB.DB()
		if err == nil {
			_ = db.Close()
		}
	}
}

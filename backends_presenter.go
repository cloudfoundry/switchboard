package switchboard

import "encoding/json"

type backendsPresenter struct {
	backends Backends
}

func NewBackendsPresenter(backends Backends) backendsPresenter {
	return backendsPresenter{
		backends: backends,
	}
}

func (bp backendsPresenter) Present() ([]byte, error) {
	backendsResponse := []string{}
	for range bp.backends.All() {
		backendsResponse = append(backendsResponse, "")
	}

	return json.Marshal(backendsResponse)
}

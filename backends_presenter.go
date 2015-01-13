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
	backendsAsJSON := []BackendJSON{}
	for backend := range bp.backends.All() {
		backendsAsJSON = append(backendsAsJSON, backend.AsJSON())
	}

	return json.Marshal(backendsAsJSON)
}

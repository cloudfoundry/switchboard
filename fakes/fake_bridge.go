package fakes

type FakeBridge struct {
	WasClosed bool
}

func (fb *FakeBridge) Connect() {}
func (fb *FakeBridge) Close() {
	fb.WasClosed = true
}

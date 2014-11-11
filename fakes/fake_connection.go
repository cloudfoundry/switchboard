package fakes

type FakeConnection struct{}

func (fc *FakeConnection) Read(buf []byte) (n int, err error) {
	return 0, nil
}

func (fc *FakeConnection) Write(buf []byte) (n int, err error) {
	return 0, nil
}

func (fc *FakeConnection) Close() (err error) {
	return nil
}

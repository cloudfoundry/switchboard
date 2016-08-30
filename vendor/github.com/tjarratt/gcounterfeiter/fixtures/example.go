package fixtures

//go:generate counterfeiter . Example
type Example interface {
	Something()
	TakesAParameter(string)
	TakesAnInt(int)
	TakesAUint64(uint64)
	TakesThreeParameters(string, string, string)
}

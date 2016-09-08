package monitor

type DecisionCounters struct {
	counters     map[string]uint64
	conditionals map[string]func() bool
}

func NewDecisionCounters() *DecisionCounters {
	return &DecisionCounters{
		counters:     map[string]uint64{},
		conditionals: map[string]func() bool{},
	}
}

func (c *DecisionCounters) GetCount(name string) uint64 {
	return c.counters[name]
}

func (c *DecisionCounters) IncrementCount(name string) {
	c.counters[name] += 1
}

func (c *DecisionCounters) ResetCount(name string) {
	c.counters[name] = 0
}

func (c *DecisionCounters) AddCondition(taskName string, predicate func() bool) {
	c.conditionals[taskName] = predicate
}

func (c *DecisionCounters) Should(taskName string) bool {
	f, ok := c.conditionals[taskName]

	if !ok {
		return false
	}

	return f()
}

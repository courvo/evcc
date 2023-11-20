package core

type Progress struct {
	min, step float64
	current   *float64
}

func NewProgress(min, step float64) *Progress {
	return &Progress{
		min:  min,
		step: step,
		// current: min - 1,
	}
}

func (p *Progress) NextStep(value float64) bool {
	// test guard
	if p == nil || (p.current != nil && value == *p.current) || value < p.min {
		return false
	}

	lower := p.min
	for lower < value-p.step {
		lower += p.step
	}
	upper := lower + p.step

	defer func() {
		p.current = &value
	}()

	if p.current == nil {
		if value == lower || value == upper {
			return true
		}
		return false
	}

	if value < *p.current && value <= lower || value > *p.current && value >= upper {
		return true
	}

	// 	for p.current <= value {
	// 		p.current += p.step
	// 	}

	// 	return true
	// }

	// if value <= p.current-p.step {
	// 	for p.current >= value {
	// 		p.current -= p.step
	// 	}

	// 	return true
	// }

	return false
}

func (p *Progress) Reset() {
	// test guard
	if p != nil {
		p.current = nil
	}
}

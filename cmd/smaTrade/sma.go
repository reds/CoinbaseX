package main

// Simple moving average

type SMA struct {
	data []float64
	size int
	idx  int
	avg  float64
}

func newSMA(size int) *SMA {
	return &SMA{data: make([]float64, 0, size), size: size}
}

func (a *SMA) next(n float64) float64 {
	if a.idx < a.size {
		a.data = append(a.data, n)
		a.idx++
		return .0
	}
	if a.idx >= a.size {
		sum := .0
		for i := 0; i < a.size; i++ {
			sum += a.data[i]
		}
		a.avg = sum / float64(a.size)
		a.data[a.idx%a.size] = n
		a.idx++
		return a.avg
	}
	oldn := a.data[a.idx%a.size]
	a.data[a.idx%a.size] = n
	a.idx++
	a.avg = a.avg - oldn/float64(a.size) + n/float64(a.size)
	return a.avg
}

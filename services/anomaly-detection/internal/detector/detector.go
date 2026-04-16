package detector

import (
	"math"
)

type Stats struct {
	Mean   float64
	StdDev float64
}

type Anomaly struct {
	Index    int
	Actual   float64
	Expected float64
	ZScore   float64
	Severity string
}

func ComputeStats(values []float64) Stats {
	if len(values) == 0 {
		return Stats{}
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	return Stats{Mean: mean, StdDev: math.Sqrt(variance)}
}

// DetectAnomalies checks every point against the global mean/stddev
func DetectAnomalies(values []float64, threshold float64) []Anomaly {
	if len(values) < 10 {
		return nil
	}

	stats := ComputeStats(values)
	if stats.StdDev == 0 {
		return nil
	}

	var anomalies []Anomaly
	for i, v := range values {
		zscore := math.Abs((v - stats.Mean) / stats.StdDev)
		if zscore >= threshold {
			severity := "warning"
			if zscore >= threshold*1.5 {
				severity = "critical"
			}
			anomalies = append(anomalies, Anomaly{
				Index:    i,
				Actual:   v,
				Expected: stats.Mean,
				ZScore:   zscore,
				Severity: severity,
			})
		}
	}
	return anomalies
}

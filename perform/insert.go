package main

import (
	"math/rand/v2"
)

type LSLScenario struct {
	AppendScenario
	LSL float64
}

func (cfg LSLScenario) Value(remain int) float64 {
	if remain == cfg.DataTotal/2 {
		// Add a specific value for the middle record
		return cfg.LSL
	} else {
		// Generate a random value for the record
		return rand.Float64() * 10
	}
}

func (cfg *LSLScenario) Run() {
	cfg.run(cfg.Value)
}

type USLScenario struct {
	AppendScenario
	USL float64
}

func (cfg USLScenario) Value(remain int) float64 {
	if remain == cfg.DataTotal/2 {
		// Add a specific value for the middle record
		return cfg.USL
	} else {
		// Generate a random value for the record
		return rand.Float64() * 10
	}
}

func (cfg *USLScenario) Run() {
	cfg.run(cfg.Value)
}

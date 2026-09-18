package nexus

import (
	"testing"
)

func TestReleaseMissionWorkerCannotDeleteNewGeneration(t *testing.T) {
	n := &Nexus{workers: map[string]*missionWorker{}}
	old := &missionWorker{done: make(chan struct{})}
	newGeneration := &missionWorker{done: make(chan struct{})}
	n.workers["run-1"] = newGeneration

	n.releaseMissionWorker("run-1", old)
	n.workersMu.Lock()
	current := n.workers["run-1"]
	n.workersMu.Unlock()
	if current != newGeneration {
		t.Fatal("old worker removed or replaced the newer worker generation")
	}

	n.releaseMissionWorker("run-1", newGeneration)
	n.workersMu.Lock()
	_, exists := n.workers["run-1"]
	n.workersMu.Unlock()
	if exists {
		t.Fatal("current worker generation was not released")
	}
}

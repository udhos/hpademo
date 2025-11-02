package main

import "time"

type deployment struct {
	podList         []pod
	desiredReplicas int
	startupTime     time.Duration
	stopTime        time.Duration
}

type pod struct {
	status           podStatus
	lastStatusChange time.Time
}

type podStatus int

const (
	podStatusStarting podStatus = iota
	podStatusRunning
	podStatusTerminating
)

func (d *deployment) getReplicas() int {
	return len(d.podList)
}

func (d *deployment) scale(replicas int) {
	d.desiredReplicas = replicas
}

func (d *deployment) update() {
	var newPodList []pod

	var running int

	for _, p := range d.podList {
		switch p.status {
		case podStatusTerminating:
			if elap := time.Since(p.lastStatusChange); elap < d.stopTime {
				newPodList = append(newPodList, p) // preserve pod
			}
		case podStatusStarting:
			if elap := time.Since(p.lastStatusChange); elap > d.startupTime {
				// promote to running
				p.status = podStatusRunning
				p.lastStatusChange = time.Now()
			}
			newPodList = append(newPodList, p)
		case podStatusRunning:
			running++
		}
	}

	// remove running PODs
	removePods := running - d.desiredReplicas
	for i, p := range newPodList {
		if removePods < 1 {
			break
		}
		if p.status == podStatusRunning {
			// switch pod to terminating
			p.status = podStatusTerminating
			p.lastStatusChange = time.Now()
			newPodList[i] = p
			removePods--
		}
	}

	// create PODs
	needNewPods := d.desiredReplicas - len(newPodList)
	if needNewPods > 0 {
		for range needNewPods {
			newPodList = append(newPodList, pod{
				status:           podStatusStarting,
				lastStatusChange: time.Now(),
			})
		}
	}

	d.podList = newPodList
}

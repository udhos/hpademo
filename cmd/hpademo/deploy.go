package main

import (
	"time"
)

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

func (d *deployment) countStatus(status podStatus) int {
	var count int
	for _, p := range d.podList {
		if p.status == status {
			count++
		}
	}
	return count
}

func (d *deployment) getRunning() int {
	return d.countStatus(podStatusRunning)
}

func (d *deployment) getStarting() int {
	return d.countStatus(podStatusStarting)
}

func (d *deployment) getStopping() int {
	return d.countStatus(podStatusTerminating)
}

func (d *deployment) scale(replicas int) {
	d.desiredReplicas = replicas
}

/*
func (d *deployment) log(label string) {
	fmt.Printf("%s: pods:%d run:%d start:%d stop:%d\n",
		label,
		d.getReplicas(), d.getRunning(), d.getStarting(),
		d.getStopping())
}
*/

func (d *deployment) update() {
	var newPodList []pod

	var running int

	//d.log("before")

	for _, p := range d.podList {
		switch p.status {
		case podStatusTerminating:
			if elap := time.Since(p.lastStatusChange); elap < d.stopTime {
				newPodList = append(newPodList, p) // preserve pod
			}
			//fmt.Println("preserved stopping")
		case podStatusStarting:
			if elap := time.Since(p.lastStatusChange); elap > d.startupTime {
				// promote to running
				p.status = podStatusRunning
				p.lastStatusChange = time.Now()
				//fmt.Println("promoted to running")
			}
			newPodList = append(newPodList, p)
		case podStatusRunning:
			newPodList = append(newPodList, p)
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
			//fmt.Println("promoted to terminating")
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
			//fmt.Println("started")
		}
	}

	d.podList = newPodList

	//d.log("after")
}

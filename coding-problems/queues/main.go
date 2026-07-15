package main

// number of recent calls

type RecentCounter struct {
	queue []int
}

func Constructor() RecentCounter {
	return RecentCounter{queue: []int{}}
}

func (rc *RecentCounter) Ping(time int) int {
	rc.queue = append(rc.queue, time)
	for rc.queue[0] < time-3000 {
		rc.queue = rc.queue[1:]
	}
	return len(rc.queue)
}

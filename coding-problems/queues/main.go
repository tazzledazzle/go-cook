package main

// number of recent calls

type RecentCounter struct {
	queue []int
}

const MaxInt = int(^uint(0) >> 1)
const MinInt = -MaxInt - 1

type Queue struct {
	// enQueue increments rear then writes to array[rear]; enQueue writes to array[front] then
	// decrements front; len(array) is a power of two; unused slots are nil and not garbage
	array    []interface{}
	front    int
	rear     int
	capacity int
	size     int
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

// dota2 senate
func predictPartyVictory(senate string) string {
	n := len(senate)
	radiant, dire := []int{}, []int{}
	for i, ch := range senate {
		if ch == 'R' {
			radiant = append(radiant, i)
		} else {
			dire = append(dire, i)
		}
	}
	for len(radiant) > 0 && len(dire) > 0 {
		rad, dir := radiant[0], dire[0]
		radiant = radiant[1:]
		dire = dire[1:]
		if rad < dir {
			radiant = append(radiant, rad+n)
		} else {
			dire = append(dire, rad+n)
		}
	}
	if len(radiant) > 0 {
		return "Radiant"
	}
	return "Dire"
}

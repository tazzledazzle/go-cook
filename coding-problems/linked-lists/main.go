package main

// common defs
type ListNode struct {
	Val  int
	Next *ListNode
}

// delete middle
func deleteMiddle(head *ListNode) *ListNode {
	if head == nil || head.Next == nil {
		return nil
	}
	var prev *ListNode
	slow, fast := head, head
	for fast != nil && fast.Next != nil {
		prev = slow
		slow = slow.Next
		fast = fast.Next.Next
	}
	prev.Next = slow.Next
	return head
}

// odd even linked list
func oddEvenList(head *ListNode) *ListNode {
	if head == nil || head.Next == nil {
		return head
	}
	odd, even := head, head.Next
	evenHead := even
	for even != nil && even.Next != nil {
		odd.Next = even.Next
		odd = odd.Next
		even.Next = odd.Next
		even = even.Next
	}
	odd.Next = evenHead
	return head
}

// reverse linked list
func reverseList(head *ListNode) *ListNode {
	var previous *ListNode
	current := head
	for current != nil {
		next := current.Next
		current.Next = previous
		previous = current
		current = next
	}
	return previous
}

// maximum twin sum of a linked list
func pairSum(head *ListNode) int {
	slowPtr, fastPtr := head, head
	for fastPtr != nil && fastPtr.Next != nil {
		slowPtr = slowPtr.Next
		fastPtr = fastPtr.Next.Next
	}
	var previous *ListNode
	current := slowPtr
	for current != nil {
		next := current.Next
		current.Next = previous
		previous = current
		current = next
	}
	best := 0
	firstPtr, secondPtr := head, previous
	for secondPtr != nil {
		if firstPtr.Val+secondPtr.Val > best {
			best = firstPtr.Val + secondPtr.Val
		}
		firstPtr = firstPtr.Next
		secondPtr = secondPtr.Next
	}
	return best
}

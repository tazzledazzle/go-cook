package main

type Trie struct {
	letter   rune                   // Contains the current node character
	children []*Trie                // Pointers to the other tri nodes
	meta     map[string]interface{} // meta data for the given word
	isLeaf   bool                   // Indicates whether the string formed from root to current node is a string or not
}

// set a current node as a root node
// set the current letter as the first letter of the word
// if the current node has an already existing reference to the current letters,
// then set current node to that referenced node
// otherwise create a new node, set the letter equal to the current letter, initialize current node to new node
// repeat until the key is traversed

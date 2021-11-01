// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Generating random text: a Markov chain algorithm

Based on the program presented in the "Design and Implementation" chapter
of The Practice of Programming (Kernighan and Pike, Addison-Wesley 1999).
See also Computer Recreations, Scientific American 260, 122 - 125 (1989).

A Markov chain algorithm generates text by creating a statistical model of
potential textual suffixes for a given prefix. Consider this text:

	I am not a number! I am a free man!

Our Markov chain algorithm would arrange this text into this set of prefixes
and suffixes, or "chain": (This table assumes a prefix length of two words.)

	Prefix       Suffix

	"" ""        I
	"" I         am
	I am         a
	I am         not
	a free       man!
	am a         free
	am not       a
	a number!    I
	number! I    am
	not a        number!

To generate text using this table we select an initial prefix ("I am", for
example), choose one of the suffixes associated with that prefix at random
with probability determined by the input statistics ("a"),
and then create a new prefix by removing the first word from the prefix
and appending the suffix (making the new prefix is "am a"). Repeat this process
until we can't find any suffixes for the current prefix or we exceed the word
limit. (The word limit is necessary as the chain table may contain cycles.)

Our version of this program reads text from standard input, parsing it into a
Markov chain, and writes generated text to standard output.
The prefix and output lengths can be specified using the -prefix and -words
flags on the command-line.
*/
package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

// Prefix is a Markov chain prefix of one or more words.
type Prefix []string

// String returns the Prefix as a string (for use as a map key).
func (p Prefix) String() string {
	return strings.Join(p, " ")
}

// Shift removes the first word from the Prefix and appends the given word.
func (p Prefix) Shift(word string) {
	copy(p, p[1:])
	p[len(p)-1] = word
}

// Chain contains a map ("chain") of prefixes to a list of suffixes.
// A prefix is a string of prefixLen words joined with spaces.
// A suffix is a single word. A prefix can have multiple suffixes.
type Chain struct {
	chain     map[string][]string
	freqTable map[string]map[string]int
	prefixLen int
}

// NewChain returns a new Chain with prefixes of prefixLen words.
func NewChain(prefixLen int) *Chain {
	return &Chain{make(map[string][]string), make(map[string]map[string]int), prefixLen}
}

// RecordSuffixFrequency takes a prefix and a suffix and creates a
// frequency map of suffixes to the count
func (c *Chain) RecordSuffixFrequency(p, s string) map[string]map[string]int {

	if freqMap, ok := c.freqTable[p]; ok { // if prefix is in freqTable
		freqMap[s]++
	} else {
		freqMap := make(map[string]int)
		c.freqTable[p] = freqMap
		freqMap[s]++
	}

	return c.freqTable
}

// Build reads text from the provided Reader and
// parses it into prefixes and suffixes that are stored in Chain.
func (c *Chain) Build(inFile string) {

	file, err := os.Open(inFile)
	if err != nil {
		fmt.Println("Error:  opening the file.")
	}

	defer file.Close()

	br := bufio.NewReader(file)

	p := make(Prefix, c.prefixLen)
	for i := range p {
		p[i] = "\"\""
	}

	for {
		var s string
		if _, err := fmt.Fscan(br, &s); err != nil {
			break
		}
		prefixKey := p.String()

		c.chain[prefixKey] = append(c.chain[prefixKey], s)
		c.freqTable = c.RecordSuffixFrequency(prefixKey, s)

		// fmt.Println(prefixKey, c.chain[prefixKey]) // for testing

		p.Shift(s)
	}

}

// StoreFrequencyTable translates creates a file called filname
// and prints the chain frequency table (c.FreqTable, a map of prefixes
// to a map of suffixes to count) to the file
func (c *Chain) StoreFrequencyTable(filename string) {

	out, err := os.Create(filename)
	if err != nil {
		fmt.Println("Sorry: couldn't create the file!")
	}
	defer out.Close()

	fmt.Fprintln(out, c.prefixLen)

	for prefix, freqMap := range c.freqTable {
		fmt.Fprint(out, prefix)

		for s, count := range freqMap {
			fmt.Fprint(out, " ", s, " ", strconv.Itoa(count))
		}
		fmt.Fprintln(out)
	}
}

func ReadFrequencyTableFromFile(filename string) *Chain {

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error:  opening the file.")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	scanner.Scan()
	prefixLen, _ := strconv.Atoi(scanner.Text())

	c := NewChain(prefixLen)
	p := make(Prefix, prefixLen)

	//Scan through each line
	for scanner.Scan() {
		currentLine := scanner.Text()
		line := strings.Split(currentLine, " ")

		for i := 0; i < prefixLen; i++ {
			if line[i] == "\"\"" {
				line[i] = ""
			}
			p[i] = line[i]
		}
		prefixKey := p.String()
		for i := prefixLen; i < len(line); i++ {
			suffix := line[i]
			freq, err := strconv.Atoi(line[i+1])
			if err != nil {
				panic(err)
			}
			for j := 0; j < freq; j++ {
				c.chain[prefixKey] = append(c.chain[prefixKey], suffix)
			}
			i++
		}
	}

	return c
}

// Generate returns a string of at most n words generated from modelFile.
func Generate(filename string, n int) string {

	c := ReadFrequencyTableFromFile(filename)

	p := make(Prefix, c.prefixLen)
	var words []string
	for i := 0; i < n; i++ {
		choices := c.chain[p.String()]
		if len(choices) == 0 {
			break
		}
		next := choices[rand.Intn(len(choices))]
		words = append(words, next)
		p.Shift(next)
	}

	return strings.Join(words, " ")
}

func main() {

	command := os.Args[1]

	if command == "read" {
		prefixLen, _ := strconv.Atoi(os.Args[2]) // gives value and err
		outfile := os.Args[3]

		c := NewChain(prefixLen)
		for i := 4; i < len(os.Args); i++ {
			c.Build(os.Args[i])
		}
		c.StoreFrequencyTable(outfile)

	} else if command == "generate" {
		modelFile := os.Args[2]
		numWords, _ := strconv.Atoi(os.Args[3])

		text := Generate(modelFile, numWords) // Generate text.
		fmt.Println(text)                     // Write text to standard output.

	} else {
		panic("Invalid command")
	}
}

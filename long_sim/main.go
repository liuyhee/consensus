// example run: ./long_sim -lbp=100 -rounds=10 -miners=10 -trials=100
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var suite bool
var test bool

var uniqueID int

const bigOlNum = 100000

func printSingle(content string) {
	if !suite {
		fmt.Printf(content)
	}
}

func printNonTest(content string) {
	if !test {
		fmt.Printf(content)
	}
}

func getUniqueID() int {
	uniqueID += 1
	return uniqueID - 1
}

// Input a set of newly mined blocks, return a map grouping these blocks
// into tipsets that obey the tipset invariants.
func allTipsets(blks []*Block) map[string]*Tipset {
	tipsets := make(map[string]*Tipset)
	for i, blk1 := range blks {
		tipset := []*Block{blk1}
		for j, blk2 := range blks {
			if i != j {
				if blk1.Parents.Name == blk2.Parents.Name &&
					blk1.Height == blk2.Height {
					tipset = append(tipset, blk2)
				}
			}
		}
		key := stringifyBlocks(tipset)
		if _, seen := tipsets[key]; !seen {
			tipsets[key] = NewTipset(tipset)
		}
	}
	return tipsets
}

// forkTipsets returns the n subsets of a tipset of length n: for every ticket
// it returns a tipset containing the block containing that ticket and all blocks
// containing a ticket larger than it.  This is a rational miner trying to mine
// all possible non-slashable forks off of a tipset.
func forkTipsets(ts *Tipset) []*Tipset {
	var forks []*Tipset
	// works because blocks are kept ordered in Tipsets
	for i := range ts.Blocks {
		currentFork := []*Block{ts.Blocks[i]}
		for j := i + 1; j < len(ts.Blocks); j++ {
			currentFork = append(currentFork, ts.Blocks[j])
		}
		forks = append(forks, NewTipset(currentFork))
	}
	return forks
}

// Block
type Block struct {
	// Nonce is unique for each block
	Nonce   int
	Parents *Tipset
	Owner   int
	Height  int
	Null    bool
	Weight  int
	Seed    int64
}

// Tipset
type Tipset struct {
	// Blocks are sorted
	Blocks    []*Block
	Name      string
	MinTicket int64
}

// Tipset helper functions
func NewTipset(blocks []*Block) *Tipset {
	sort.Slice(blocks, func(i, j int) bool { return blocks[i].Seed < blocks[j].Seed })
	minTicket := int64(-1)
	for _, block := range blocks {
		if minTicket == int64(-1) || block.Seed < minTicket {
			minTicket = block.Seed
		}
	}
	return &Tipset{
		Blocks:    blocks,
		Name:      stringifyBlocks(blocks),
		MinTicket: minTicket,
	}
}

func getSortedBlockNames(blocks []*Block) []string {
	var blockNames []string
	for _, block := range blocks {
		blockNames = append(blockNames, strconv.Itoa(block.Nonce))
	}

	sort.Strings(blockNames)
	return blockNames
}

func stringifyBlocks(blocks []*Block) string {
	strBlocks := getSortedBlockNames(blocks)
	return strings.Join(strBlocks, "-")
}

func (ts *Tipset) getHeight() int {
	if len(ts.Blocks) == 0 {
		panic("Don't call height on no parents")
	}
	// Works because all blocks in a tipset have same height (see allTipsets)
	return ts.Blocks[0].Height
}

func (ts *Tipset) getWeight() int {
	if len(ts.Blocks) == 0 {
		panic("Don't call weight on no parents")
	}
	// Works because all blocks in a tipset have the same parent (see allTipsets)
	return len(ts.Blocks) + ts.Blocks[0].Weight - 1
}

func (ts *Tipset) getParents() *Tipset {
	if len(ts.Blocks) == 0 {
		panic("Don't call parents on nil blocks")
	}
	return ts.Blocks[0].Parents
}

// Chain tracker
type chainTracker struct {
	// index tipsets per height
	blocksByHeight map[int][]*Block
	blocks         map[int]*Block
	maxHeight      int
}

func NewChainTracker() *chainTracker {
	return &chainTracker{
		blocksByHeight: make(map[int][]*Block),
		blocks:         make(map[int]*Block),
		maxHeight:      -1,
	}
}

// Rational Miner
type RationalMiner struct {
	Power        float64
	PrivateForks map[string]*Tipset
	ID           int
	TotalMiners  int
}

// Rational Miner helper functions
func NewRationalMiner(id int, power float64, totalMiners int) *RationalMiner {
	return &RationalMiner{
		Power:        power,
		PrivateForks: make(map[string]*Tipset, 0),
		ID:           id,
		TotalMiners:  totalMiners,
	}
}

// Input the base tipset for mining lookbackTipset will return the ancestor
// tipset that should be used for sampling the leader election seed.
// On LBP == 1, returns itself (as in no farther than direct parents)
func lookbackTipset(tipset *Tipset, lbp int) *Tipset {
	// TODO: what happens when lbp > chain height? It seems to work, but why? gen block does not recurse infinitely
	for i := 0; i < lbp-1; i++ {
		tipset = tipset.getParents()
	}
	return tipset
}

// generateBlock makes a new block with the given parents
func (m *RationalMiner) generateBlock(parents *Tipset, lbp int) *Block {
	// Given parents and id we have a unique source for new ticket
	minTicket := lookbackTipset(parents, lbp).MinTicket

	t := m.generateTicket(minTicket)
	nextBlock := &Block{
		Nonce:   getUniqueID(),
		Parents: parents,
		Owner:   m.ID,
		Height:  parents.getHeight() + 1,
		Weight:  parents.getWeight(),
		Seed:    t,
	}

	if isWinningTicket(t, m.Power, m.TotalMiners) {
		nextBlock.Null = false
		nextBlock.Weight += 1
	} else {
		nextBlock.Null = true
	}

	return nextBlock
}

func isWinningTicket(ticket int64, power float64, totalMiners int) bool {
	// this is a simulation of ticket checking: the ticket is drawn uniformly from 0 to bigOlNum * totalMiners.
	// If it is smaller than that * the miner's power (between 0 and 1), it wins.
	return float64(ticket) < float64(bigOlNum)*float64(totalMiners)*power
}

// generateTicket
func (m *RationalMiner) generateTicket(minTicket int64) int64 {
	seed := minTicket + int64(m.ID)
	r := rand.New(rand.NewSource(seed))
	ticket := r.Int63n(int64(bigOlNum * m.TotalMiners))
	return ticket
}

func (m *RationalMiner) SourceAllForks(newBlocks []*Block) {
	// split the newblocks into all potential forkable tipsets
	allTipsets := allTipsets(newBlocks)
	// rational miner strategy look for all potential minblocks there
	for k := range allTipsets {
		forkTipsets := forkTipsets(allTipsets[k])
		for _, ts := range forkTipsets {
			m.PrivateForks[ts.Name] = ts
		}
	}
}

// Mine outputs the block that a miner mines in a round where the leaves of
// the block tree are given by newBlocks.  A miner will only ever mine one
// block in a round because if it mines two or more it gets slashed.  #Incentives #Blockchain
func (m *RationalMiner) Mine(newBlocks []*Block, lbp int) *Block {
	// Start by combining existing pforks and new blocks available to mine atop of
	m.SourceAllForks(newBlocks)

	var nullBlocks []*Block
	maxWeight := 0
	var bestBlock *Block
	printSingle(fmt.Sprintf("miner %d. number of priv forks: %d\n", m.ID, len(m.PrivateForks)))
	for k := range m.PrivateForks {
		// generateBlock takes in a block's parent tipset, as in current head of PrivateForks
		blk := m.generateBlock(m.PrivateForks[k], lbp)
		if !blk.Null && blk.Weight > maxWeight {
			bestBlock = blk
			maxWeight = blk.Weight
		} else if blk.Null && bestBlock == nil {
			// if blk is null and we haven't found a winning block yet
			// we will want to extend private forks with it
			// no need to do it if blk is not null since the pforks will get deleted anyways
			nullBlocks = append(nullBlocks, blk)
		}
	}

	// if bestBlock is not null
	if bestBlock != nil {
		// kill all pforks
		m.PrivateForks = make(map[string]*Tipset)
	} else {
		// extend null block chain
		for _, nblk := range nullBlocks {
			delete(m.PrivateForks, nblk.Parents.Name)
			// add the new null block to our private forks
			nullTipset := NewTipset([]*Block{nblk})
			m.PrivateForks[nullTipset.Name] = nullTipset
		}
	}
	return bestBlock
}

// makeGen makes the genesis block.  In the case the lbp is more than 1 it also
// makes lbp -1 genesis ancestors for sampling the first lbp - 1 blocks after genesis
func makeGen(lbp int, totalMiners int) *Block {
	var gen *Tipset
	for i := 0; i < lbp; i++ {
		gen = NewTipset([]*Block{&Block{
			Nonce:   getUniqueID(),
			Parents: gen,
			Owner:   -1,
			Height:  0,
			Null:    false,
			Weight:  0,
			Seed:    rand.Int63n(int64(bigOlNum * totalMiners)),
		}})
	}
	return gen.Blocks[0]
}

// drawChain output a dot graph of the entire blockchain generated by the simulation
func drawChain(ct *chainTracker) {
	fmt.Printf("Writing Graph\n")
	fil, err := os.Create("chain.dot")
	if err != nil {
		panic(err)
	}
	defer fil.Close()

	fmt.Fprintln(fil, "digraph G {")
	fmt.Fprintln(fil, "\t{\n\t\tnode [shape=plaintext];")

	// Write out height index alongside the block graph
	fmt.Fprintf(fil, "\t\t0")
	// Start at 1 because we already wrote out the 0 for the .dot file
	for cur := int(1); cur <= ct.maxHeight; cur++ {
		fmt.Fprintf(fil, " -> %d", cur)
	}
	fmt.Fprintln(fil, ";")
	fmt.Fprintln(fil, "\t}")

	// Write out the actual blocks
	fmt.Fprintln(fil, "\tnode [shape=box];")
	for cur := ct.maxHeight; cur >= 0; cur-- {
		// get blocks per height
		blocks, ok := ct.blocksByHeight[cur]
		// if no blocks at height, skip
		if !ok {
			continue
		}

		// for every block at this height
		fmt.Fprintf(fil, "\t{ rank = same; %d;", cur)

		for _, block := range blocks {
			// print block
			fmt.Fprintf(fil, " \"b%d (m%d)\";", block.Nonce, block.Owner)
		}
		fmt.Fprintln(fil, " }")

		// link to parents
		for _, block := range blocks {
			// genesis has no parents
			if block.Owner == -1 {
				continue
			}
			parents := block.Parents
			// Tipsets with null blocks only contain one block (since null blocks are mined privately)
			// walk back until we find a tipset with a live parent
			for parents.Blocks[0].Null {
				parents = parents.Blocks[0].Parents
			}
			for _, parent := range parents.Blocks {
				fmt.Fprintf(fil, "\t\"b%d (m%d)\" -> \"b%d (m%d)\";\n", block.Nonce, block.Owner, parent.Nonce, parent.Owner)
			}
		}
	}

	fmt.Fprintln(fil, "}\n")
}

func averageLiveForksPerRound(ct *chainTracker) float64 {
	var sum int
	for cur := ct.maxHeight; cur >= 0; cur-- {
		// get blocks per height
		blocks, ok := ct.blocksByHeight[cur]
		// if no blocks at height, skip
		if !ok {
			continue
		}
		sum += len(blocks)

	}
	return float64(sum) / float64(ct.maxHeight+1)
}

func analyzeSim(cts []*chainTracker) float64 {
	// run analysis on the chains here

	// 1. average num of live forks
	// TODO: math lib? parallel?
	var sum float64
	for n := 0; n < len(cts); n++ {
		// for each chain
		sum += averageLiveForksPerRound(cts[n])
	}

	return sum / float64(len(cts))
}

func runSingleSim(totalMiners int, roundNum int, lbp int, c chan *chainTracker) {
	uniqueID = 0
	rand.Seed(time.Now().UnixNano())
	chainTracker := NewChainTracker()
	miners := make([]*RationalMiner, totalMiners)
	gen := makeGen(lbp, totalMiners)
	for m := 0; m < totalMiners; m++ {
		miners[m] = NewRationalMiner(m, 1.0/float64(totalMiners), totalMiners)
	}
	blocks := []*Block{gen}
	var currentHeight int
	for round := 0; round < roundNum; round++ {

		// Cache blocks for future stats
		for _, blk := range blocks {
			chainTracker.blocks[blk.Nonce] = blk
		}

		// checking an assumption
		if len(blocks) > 0 {
			currentHeight = blocks[0].Height
		}
		for _, blk := range blocks {
			if currentHeight != blk.Height {
				// TODO: have seen this, can't reproduce. Fix.
				panic("Check your assumptions: all block heights from a round are not equal")
			}
		}
		chainTracker.blocksByHeight[currentHeight] = blocks

		printSingle(fmt.Sprintf("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%\n"))
		printSingle(fmt.Sprintf("Round %d -- %d new blocks\n", round, len(blocks)))
		for _, blk := range blocks {
			printSingle(fmt.Sprintf("b%d (m%d)\t", blk.Nonce, blk.Owner))
		}
		printSingle(fmt.Sprintf("\n"))
		printSingle(fmt.Sprintf("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%\n"))
		var newBlocks = []*Block{}
		for _, m := range miners {
			// Each miner mines
			blk := m.Mine(blocks, lbp)
			if blk != nil {
				newBlocks = append(newBlocks, blk)
			}
		}
		// NewBlocks added to network
		// use if condition as otherwise blocks with empty next heights are erased
		if len(newBlocks) > 0 {
			blocks = newBlocks
		}
	}
	// height is 0 indexed
	chainTracker.maxHeight = roundNum - 1
	c <- chainTracker
}

func run(trials int, lbp int, roundNum int, totalMiners int) []*chainTracker {

	if trials <= 0 {
		panic("None of your assumptions have been proven wrong")
	}
	suite = trials > 1
	var cts []*chainTracker
	c := make(chan *chainTracker, trials)

	for n := 0; n < trials; n++ {
		printNonTest(fmt.Sprintf("Trial %d\n-*-*-*-*-*-*-*-*-*-*-\n", n))
		go runSingleSim(totalMiners, roundNum, lbp, c)
	}
	for n := 0; n < trials; n++ {
		cts = append(cts, <-c)
	}

	if trials == 1 {
		printNonTest(fmt.Sprintf("Sim produced %d blocks\n", len(cts[0].blocks)))
		drawChain(cts[0])
	} else {
		printNonTest(fmt.Sprintf("%d trials run\n", len(cts)))
		avg := analyzeSim(cts)
		printNonTest(fmt.Sprintf("%.2f average forks per round across %d chains with lbp %d", avg, len(cts), lbp))
	}

	return cts
}

func runAndAnalyze(trials int, lbp int, roundNum int, totalMiners int, results map[int]map[int]float64, wg sync.WaitGroup) {
	defer wg.Done()
	cts := run(trials, lbp, roundNum, totalMiners)
	results[totalMiners][lbp] = analyzeSim(cts)
}

func runTests() {
	// x is lbps, y is number of forks at any height for different numbers of miners
	// map num of miners to map of lbp to average num of forks for that lbp
	results := make(map[int]map[int]float64)
	numMiners := []int{10, 100} //, 1000}
	maxLBP := 150
	numRounds := 300
	numTrials := 20
	lbps := []int{1}
	for lbp := 10; lbp <= maxLBP; lbp += 30 {
		lbps = append(lbps, lbp)
	}

	fmt.Printf("Starting %d different sims run %d times each... (can take a long time)\n", len(lbps)*len(numMiners), numTrials)
	var wgSims sync.WaitGroup
	for _, numM := range numMiners {
		fmt.Printf("\n%d Miners:", numM)
		results[numM] = make(map[int]float64)
		for _, lbp := range lbps {
			fmt.Printf("\tLBP: %d", lbp)
			wgSims.Add(1)
			go runAndAnalyze(numTrials, lbp, numRounds, numM, results, wgSims)
		}
	}
	wgSims.Wait()

	fmt.Printf("we have %d miners, %d lbps per miner, and one is %.2f", len(results), len(results[10]), results[10][1])
}

func main() {

	fLbp := flag.Int("lbp", 1, "sim lookback")
	fRoundNum := flag.Int("rounds", 100, "number of rounds to sim")
	fTotalMiners := flag.Int("miners", 10, "number of miners to sim")
	fNumTrials := flag.Int("trials", 1, "number of trials to run")
	fTest := flag.Bool("test", false, "run automated tests")

	flag.Parse()
	lbp := *fLbp
	roundNum := *fRoundNum
	totalMiners := *fTotalMiners
	trials := *fNumTrials
	test = *fTest

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if test {
		runTests()
	} else {
		run(trials, lbp, roundNum, totalMiners)
	}
}

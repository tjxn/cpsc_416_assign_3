// Simple client to connect to the key-value service and exercise the
// key-value RPC API (put/get/test-set).
//
// Usage: go run kvclientmain.go [ip:port]
//
// - [ip:port] : the ip and TCP port on which the KV service is
//               listening for client connections.
//
// TODOs:
// - Needs refactoring and optional support for vector-timestamps.

package main

import (
	"fmt"
	"net/rpc"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Constants
var DELAY time.Duration = 2
var DELAY_SHORT time.Duration = 1

// Global Variables
var kvService *rpc.Client
var lastNum int
var leaderNum int
var registerNum int
var myIDNum int
var myID string
var activeNodes []string
var nodeList []string
var currentNodeKey map[string]int
var currentNodeNum map[string]int
var currentLeader string
var activeNodeKey int
var verbosePrintOnOff bool
var die bool
var killLeader bool

// args in get(args)
type GetArgs struct {
	Key string // key to look up
}

// args in put(args)
type PutArgs struct {
	Key string // key to associate value with
	Val string // value
}

// args in testset(args)
type TestSetArgs struct {
	Key     string // key to test
	TestVal string // value to test against actual value
	NewVal  string // value to use if testval equals to actual value
}

// Reply from service for all three API calls above.
type ValReply struct {
	Val string // value; depends on the call
}

type KeyValService int

func checkArgs() {

	usage := fmt.Sprintf("Usage: %s ip:port id\n", os.Args[0])

	if len(os.Args) != 3 {
		fmt.Printf(usage)
		os.Exit(1)
	}
}

func checkForLeader() {
	var stringLeaderNum string = strconv.Itoa(leaderNum)

	var result string = get("Leader-" + stringLeaderNum)
	switch {

	case result == "unavailable":
		leaderNum++
		checkForLeader()

	case result == "":
		attemptToBecomeLeader("")

	default:
		currentLeader = result

		currentNodeKey = make(map[string]int)
		currentNodeNum = make(map[string]int)
		activeNodes = []string{}

		die = false

		registerWithLeader()
	}
}

func attemptToBecomeLeader(oldLeaderID string) {
	var result string = testset(getLeaderString(), oldLeaderID, myID)

	switch {
	case result == "unavailable":
		checkForLeader()

	case result == myID:
		// success

		if checkIfNewOrOld(myID) {
			addIDtoActiveNodes(result)
		}

		killLeader = false
		go updateListofActiveNodes()

		checkForNewNode()

	default:
		// failure
		checkForLeader()
	}
}

func registerWithLeader() {
	var result string = testset(getRegisterString(), "", myID)
	switch {

	case result == "unavailable":
		incrementRegisterString()
		registerWithLeader()

	case result == myID:
		// success
		result = get(getRegisterString())

		go getActiveNodes()
		iAmActive()

	default:
		// failure
		registerWithLeader()
	}
}

func getActiveNodes() {
	for {

		if die {
			return
		}

		var result string = get(getActiveNodeKey())

		switch {
		case result == "unavailable":
			incrementActiveNodeKey()
		case result == "":
			delaySecond(DELAY)

		default:
			displayActiveNodes(result)
			delaySecond(DELAY)
		}
	}

}

func iAmActive() {
	var result string = get(getMyIDString())
	result_int, _ := strconv.Atoi(result)

	switch {
	case result == "unavailable":
		incrementMyIDString()
		iAmActive()

	case result == "":
		put(getMyIDString(), getLastNumString())
		iAmActive()

	case result_int > lastNum:

		lastNum = result_int
		incrementLastNumString()

		put(getMyIDString(), getLastNumString())
		iAmActive()

	case result_int == lastNum:

		var succ int
		for i := 0; i < 10; i++ {
			succ = hasChanged()

			if succ == 1 {
				break

			} else if succ == 0 {
				delaySecond(DELAY)

			} else if succ == 2 {
				checkForLeader()
			}
		}

		if succ == 0 {
			isLeaderDead()
		} else if succ == 1 {
			iAmActive()
		}

	default:
		fmt.Println("Error occured in iAmActive function switch statement")
		os.Exit(1)

	}
}

func hasChanged() int {

	var result string = get(getMyIDString())
	result_int, _ := strconv.Atoi(result)

	switch {
	case result == "unavailable":
		incrementMyIDString()
		return 0

	case result == "":
		put(getMyIDString(), getLastNumString())
		return 1

	case result_int > lastNum:

		lastNum = result_int
		incrementLastNumString()

		put(getMyIDString(), getLastNumString())
		return 1

	case result_int == lastNum:

		return 0
	case result_int < lastNum:
		return 2

	default:
		fmt.Println("Something went wrong in hasChanged")
		os.Exit(1)
	}
	return 0
}

func isLeaderDead() {
	delaySecond(DELAY)

	var result string = get(getMyIDString())

	result_int, _ := strconv.Atoi(result)

	switch {

	case result == "unavailable":
		incrementMyIDString()
		isLeaderDead()

	case result == "":
		put(getMyIDString(), getLastNumString())
		iAmActive()

	case result_int > lastNum:
		incrementLastNumString()
		put(getMyIDString(), getLastNumString())
		iAmActive()

	case result_int == lastNum:
		die = true
		delaySecond(DELAY)
		delaySecond(DELAY)
		attemptToBecomeLeader(currentLeader)
	}
}

func delaySecond(n time.Duration) {
	time.Sleep(n * time.Second)
}

func smallDelay() {
	time.Sleep(1 * time.Millisecond)
}

func displayActiveNodes(nodes string) {
	fmt.Println(nodes)
}

func addIDtoActiveNodes(id string) {
	activeNodes = append(activeNodes, id)
}

func removeIDfromActiveNodes(id string) {
	size := len(activeNodes)

	for index, element := range activeNodes {

		if element == id {

			// Remove id by replacing it with the last element
			// in the slice
			activeNodes[index] = activeNodes[size-1]
			activeNodes = append(activeNodes[:size-1], activeNodes[size:]...)
		}
	}
}

func incrementLastNumString() string {
	lastNum++
	result := strconv.Itoa(lastNum)
	return result
}

func getLastNumString() string {
	result := strconv.Itoa(lastNum)
	return result
}

func incrementMyIDString() string {
	myIDNum++
	result := strconv.Itoa(myIDNum)
	return myID + "-" + result
}

func getMyIDString() string {
	result := strconv.Itoa(myIDNum)
	return myID + "-" + result
}

func incrementRegisterString() string {
	registerNum++
	result := strconv.Itoa(registerNum)
	return "Register-" + result
}

func getRegisterString() string {
	result := strconv.Itoa(registerNum)
	return "Register-" + result
}

func getActiveNodeKey() string {
	result := strconv.Itoa(activeNodeKey)
	return "Active-Nodes-" + result

}

func incrementActiveNodeKey() {
	activeNodeKey++
}

func incrementLeaderString() string {
	leaderNum++
	result := strconv.Itoa(leaderNum)
	return "Leader-" + result
}

func getLeaderString() string {
	result := strconv.Itoa(leaderNum)
	return "Leader-" + result
}

func get(key string) string {

	var kvVal ValReply

	getArgs := GetArgs{key}

	err := kvService.Call("KeyValService.Get", getArgs, &kvVal)
	checkError(err)

	return kvVal.Val
}

func put(key string, value string) string {
	var kvVal ValReply

	putArgs := PutArgs{
		Key: key,
		Val: value,
	}

	err := kvService.Call("KeyValService.Put", putArgs, &kvVal)
	checkError(err)

	return kvVal.Val
}

func testset(key string, value string, replacement string) string {
	var kvVal ValReply

	tsArgs := TestSetArgs{
		Key:     key,
		TestVal: value,
		NewVal:  replacement,
	}

	err := kvService.Call("KeyValService.TestSet", tsArgs, &kvVal)
	checkError(err)

	return kvVal.Val
}

// Main server loop.
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	checkArgs()

	kvAddr := os.Args[1]
	myID = os.Args[2]

	verbosePrintOnOff = true

	// Connect to the KV-service via RPC.
	var err error
	kvService, err = rpc.Dial("tcp", kvAddr)
	checkError(err)

	lastNum = 1
	leaderNum = 1
	registerNum = 1
	myIDNum = 1
	activeNodeKey = 1

	currentNodeKey = make(map[string]int)
	currentNodeNum = make(map[string]int)
	activeNodes = []string{}

	checkForLeader()

}

func verbosePrint(message string) {
	if verbosePrintOnOff {
		fmt.Println(message)
	}
}

// If error is non-nil, print it out and halt.
func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error ", err.Error())
		os.Exit(1)
	}
}

// TODO
func updateListofActiveNodes() {
	for {

		if killLeader {
			return
		}

		// Check that we are still leader
		var result string = get(getLeaderString())

		if result == "unavailable" {
			return
		}
		active_nodes := strings.Join(activeNodes, " ")
		fmt.Println(active_nodes)

		result = put(getActiveNodeKey(), active_nodes)

		switch {
		case result == "unavailable":
			incrementActiveNodeKey()

		case result == "":
			// success
			delaySecond(DELAY)

		default:
			fmt.Println("Error in updateListofActiveNodes")
			os.Exit(1)
		}
	}
}

func checkForNewNode() {
	for {
		delaySecond(DELAY)
		// Check that we are still Leader
		var result string = get(getLeaderString())

		if result == "unavailable" || result != myID {
			killLeader = true
			delaySecond(DELAY)
			delaySecond(DELAY)
			checkForLeader()
		}

		// Check for new Nodes
		result = get(getRegisterString())
		switch {

		case result == "unavailable":
			incrementRegisterString()

		case result == "":

		default:

			if checkIfNewOrOld(result) {
				addIDtoActiveNodes(result)
				incrementNodeIDString(result)

				put(getRegisterString(), "")

				go checkNodeisActive(result)
			}
		}
	}
}

func checkIfNewOrOld(id string) bool {

	for _, element := range activeNodes {

		if element == id {
			return false
		}
	}
	return true

}

func checkNodeisActive(id string) {
	for {
		var succ int
		for i := 0; i < 10; i++ {
			succ = hasNodeChanged(id)

			if succ == 1 {
				break
			} else if succ == 0 {
				delaySecond(DELAY_SHORT)

			} else if succ == 2 {
				return
			}
		}

		if succ == 0 {
			checkIfNodeDead(id)
			return
		}

	}
}

func hasNodeChanged(id string) int {
	var result string = get(getNodeIDString(id))
	result_num, _ := strconv.Atoi(result)

	switch {
	case result == "unavailable":
		incrementNodeIDString(id)
		return 1

	case result == "":
		return 0

	case result_num > currentNodeNum[id]:
		result_num++
		currentNodeNum[id] = result_num

		put(getNodeIDString(id), strconv.Itoa(currentNodeNum[id]))

		return 1
	case result_num == currentNodeNum[id]:

		return 0
	case result_num < currentNodeNum[id]:
		return 2

	default:
		fmt.Println("hasNodeChanged result: " + result)
		fmt.Println("Something went wrong in hasNodeChanged")
		os.Exit(1)
	}
	return 0

}

func checkIfNodeDead(id string) {
	delaySecond(DELAY_SHORT)

	var result string = get(getNodeIDString(id))
	result_num, _ := strconv.Atoi(result)

	switch {
	case result == "unavailable":
		incrementNodeIDString(id)

	case result == "":
		removeIDfromActiveNodes(id)

	case result_num > currentNodeNum[id]:
		result_num++
		currentNodeNum[id] = result_num

		put(getNodeIDString(id), strconv.Itoa(currentNodeNum[id]))

	case result_num == currentNodeNum[id]:
		removeIDfromActiveNodes(id)
		put(getNodeIDString(id), "")
		currentNodeKey[id] = 0
		currentNodeNum[id] = 0
	}
}

func getNodeIDString(id string) string {
	return id + "-" + strconv.Itoa(currentNodeKey[id])
}

func incrementNodeIDString(id string) {
	var num int = currentNodeKey[id]
	num++
	currentNodeKey[id] = num
}

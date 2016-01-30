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
	fmt.Println("inside checkForLeader")
	var stringLeaderNum string = strconv.Itoa(leaderNum)

	var result string = get("Leader-" + stringLeaderNum)
	fmt.Println("Leader id is: " + result)
	switch {

	case result == "unavailable":
		fmt.Println("Leader unavailable")
		leaderNum++
		checkForLeader()

	case result == "":
		fmt.Println("call Attempting to become leader from checkForLeader")
		attemptToBecomeLeader()

	default:
		fmt.Println("calling registerWithLeader from checkForLeader")
		registerWithLeader()
	}
}

func attemptToBecomeLeader() {
	fmt.Println("in attemptToBecomeLeader")
	var result string = testset(getLeaderString(), "", myID)
	fmt.Println("The Leader is: " + result)

	switch {
	case result == "unavailable":
		checkForLeader()

	case result == myID:
		// success
		addIDtoActiveNodes(myID)
		fmt.Println("I AM THE LEADER")

		//go updateListofActiveNodes()
		//fmt.Println("updateListofActiveNodes")
		go checkForNewNode()
		fmt.Println("started goroutine checkForNewNode")
		fmt.Println("call checkActiveNodesAreActive")
		checkActiveNodesAreActive()

	default:
		// failure
		checkForLeader()
	}
}

func registerWithLeader() {
	fmt.Println("inside registerWithLeader")
	var result string = testset(getRegisterString(), "", myID)
	fmt.Println("Registering with Leader")
	fmt.Println("testset Result")
	fmt.Printf("%s\n", result)
	switch {

	case result == "unavailable":
		incrementRegisterString()
		testset(getRegisterString(), "", myID)

	case result == myID:
		// success
		//go getActiveNodes()
		result = get(getRegisterString())
		fmt.Println("Register String Result")
		fmt.Println(result)
		iAmActive()

	default:
		// failure
		registerWithLeader()
	}
}

func getActiveNodes() {
	var result string = get(getLeaderString())

	switch {
	case result == "unavailable":
		checkForLeader()

	case result == "":
		getActiveNodes()

	default:
		displayActiveNodes(result)
	}

}

func iAmActive() {
	var result string = get(getMyIDString())
	result_int, _ := strconv.Atoi(result)
	fmt.Println()
	fmt.Println("in iAmActive")
	fmt.Printf("My Last Number: %d\n", lastNum)
	fmt.Printf("Recieved String: %s\n", result)
	fmt.Printf("Received Number: %d\n", result_int)
	fmt.Println()
	switch {
	case result == "unavailable":
		incrementMyIDString()
		iAmActive()

	case result == "":
		put(getMyIDString(), getLastNumString())
		delaySecond(DELAY_SHORT)
		iAmActive()

	case result_int > lastNum:

		lastNum = result_int
		incrementLastNumString()

		put(getMyIDString(), getLastNumString())
		delaySecond(DELAY_SHORT)
		iAmActive()

	case result_int == lastNum:
		isLeaderDead()

	default:
		fmt.Println("Error occured in iAmActive function switch statement")
		os.Exit(1)

	}
}

func isLeaderDead() {
	fmt.Println("Is Leader Dead?")
	delaySecond(DELAY)

	var result string = get(getMyIDString())

	result_int, err := strconv.Atoi(result)
	checkError(err)

	fmt.Printf("My Last Number: %d\n", lastNum)
	fmt.Printf("Recieved String: %s\n", result)
	fmt.Printf("Received Number: %d\n", result_int)

	switch {

	case result == "unavailable":
		incrementMyIDString()
		isLeaderDead()

	case result == "":
		put(getMyIDString(), getLastNumString())
		delaySecond(DELAY)
		iAmActive()

	case result_int > lastNum:
		incrementLastNumString()
		put(getMyIDString(), getLastNumString())
		delaySecond(DELAY)
		iAmActive()

	case result_int == lastNum:
		put(getLeaderString(), myID)
		checkForLeader()

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
	fmt.Println("Adding new node")
	activeNodes = append(activeNodes, id)
	fmt.Println(activeNodes)
}

func removeIDfromActiveNodes(id string) {
	size := len(activeNodes)

	for index, element := range activeNodes {

		if element == id {

			// Remove id by replacing it with the last element
			// in the slice
			activeNodes[index] = activeNodes[size-1]
			activeNodes[size-1] = ""
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

	// Connect to the KV-service via RPC.
	var err error
	kvService, err = rpc.Dial("tcp", kvAddr)
	checkError(err)

	lastNum = 1
	leaderNum = 1
	registerNum = 1
	myIDNum = 1

	currentNodeKey = make(map[string]int)
	currentNodeNum = make(map[string]int)
	activeNodes = []string{}

	checkForLeader()

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
	fmt.Println("Advertising active nodes")

	active_nodes := strings.Join(activeNodes, " ")

	fmt.Println(active_nodes)

	put(getLeaderString(), active_nodes)
}

//Unsure of how to check for every possible node that joins, may have arbitrary
//IDs, for now we will limit the IDs to be 0 to 10
func checkForNewNode() {
	for {
		smallDelay()
		//	for i := 0; i < 10; i++ {
		//		new_node := get(string(i))
		//		if !contains(activeNodes, new_node) {
		//			activeNodes = append(activeNodes, new_node)
		//		}
		//	}
		//	updateListofActiveNodes()

		var result string = get(getRegisterString())
		switch {

		case result == "unavailable":
			incrementRegisterString()

		case result == "":

		default:
			fmt.Println("Got a new Node")
			addIDtoActiveNodes(result)
			incrementNodeIDString(result)

			put(getRegisterString(), "")
		}
	}
}

func checkActiveNodesAreActive() {
	fmt.Println("in activeNodesAreActive")
	for {
		delaySecond(DELAY_SHORT)
		for _, element := range activeNodes {

			if element != myID && element != "" {
				var result string = get(getNodeIDString(element))
				result_num, _ := strconv.Atoi(result)

				fmt.Println("Last Number:")
				fmt.Printf("%d\n", currentNodeNum[element])
				fmt.Println("Received Number:")
				fmt.Printf("%d\n", result_num)

				switch {
				case result == "unavailable":
					incrementNodeIDString(element)

				case result == "":
					checkIfNodeDead(element)

				case result_num > currentNodeNum[element]:
					result_num++
					currentNodeNum[element] = result_num

					put(getNodeIDString(element), strconv.Itoa(currentNodeNum[element]))

				case result_num == currentNodeNum[element]:
					fmt.Println("inside of checkActiveNodesAreActive")
					fmt.Println("calling checkIfNodeDead")
					checkIfNodeDead(element)
				}
			}
		}
	}

}

func checkIfNodeDead(id string) {
	fmt.Printf("Checking if Node is Dead: %s\n", id)
	delaySecond(DELAY_SHORT)

	var result string = get(getNodeIDString(id))
	result_num, _ := strconv.Atoi(result)
	fmt.Printf("Last Number: %d\n", currentNodeNum[id])
	fmt.Printf("Received Number: %s\n", result)

	switch {
	case result == "unavailable":
		incrementNodeIDString(id)

	case result == "":
		removeIDfromActiveNodes(id)

	case result_num > currentNodeNum[id]:

		fmt.Println("Last Number:")
		fmt.Printf("%d\n", currentNodeNum[id])
		fmt.Println("Received Number:")
		fmt.Printf("%d\n", result_num)
		result_num++
		currentNodeNum[id] = result_num

		put(getNodeIDString(id), strconv.Itoa(currentNodeNum[id]))

	case result_num == currentNodeNum[id]:
		removeIDfromActiveNodes(id)
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

// Contains function used to check slice found from this thread:
// http://stackoverflow.com/questions/10485743/contains-method-for-a-slice
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

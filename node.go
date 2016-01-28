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
	"strconv"
	"strings"
	"time"
)

// Constants
var DELAY time.Duration = 5

// Global Variables
var kvService *rpc.Client
var lastNum int
var leaderNum int
var registerNum int
var myIDNum int
var myID string
var activeNodes []string
var nodeList []string

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
	fmt.Println("Checking for leader")
	var stringLeaderNum string = strconv.Itoa(leaderNum)

	var result string = get("Leader-" + stringLeaderNum)
	fmt.Println(result)
	switch {

	case result == "unavailable":
		fmt.Println("Leader unavailable")
		leaderNum++
		checkForLeader()

	case result == "":
		fmt.Println("Attempting to become leader")
		attemptToBecomeLeader()

	default:
		fmt.Println("Registering with leader")
		registerWithLeader()
	}
}

func attemptToBecomeLeader() {
	var result string = testset(getLeaderString(), "", myID)
	fmt.Println(result)

	switch {
	case result == "unavailable":
		checkForLeader()

	case result == myID:
		// success
		addIDtoActiveNodes(myID)
		fmt.Println("I AM THE LEADER")
		new_leader, err := strconv.Atoi(myID)
		checkError(err)
		leaderNum = new_leader
		//go updateListofActiveNodes()
		checkForNewNode()
		//go checkForNewNode()
		//go checkActiveNodesAreActive()

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
		testset(getRegisterString(), "", myID)

	case result == myID:
		// success
		go getActiveNodes()
		go iAmActive()

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
	result_int, err := strconv.Atoi(result)
	checkError(err)

	switch {
	case result == "unavailable":
		incrementMyIDString()
		iAmActive()

	case result == "":
		put(getMyIDString(), string(lastNum))
		delaySecond(DELAY)
		iAmActive()

	case result_int > lastNum:
		incrementLastNumString()
		put(getMyIDString(), getLastNumString())
		iAmActive()

	case result_int == lastNum:
		isLeaderDead()

	default:
		fmt.Println("Error occured in iAmActive function switch statement")
		os.Exit(1)

	}
}

func isLeaderDead() {
	delaySecond(DELAY)

	var result string = get(getMyIDString())
	result_int, err := strconv.Atoi(result)
	checkError(err)

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
		checkForLeader()

	}
}

func delaySecond(n time.Duration) {
	time.Sleep(n * time.Second)
}

func displayActiveNodes(nodes string) {
	fmt.Println(nodes)
}

func addIDtoActiveNodes(id string) {
	fmt.Println("Adding new node")
	activeNodes = append(activeNodes, id)
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
	//fmt.Println(active_nodes)
	result := strconv.Itoa(leaderNum)
	put("Leader-"+result, active_nodes)
	var check_active string = testset(getLeaderString(), "", myID)
	fmt.Println(check_active)
}

//Unsure of how to check for every possible node that joins, may have arbitrary
//IDs, for now we will limit the IDs to be 0 to 10
func checkForNewNode() {
	fmt.Println("Finding nodes")
	for i := 0; i < 10; i++ {
		new_node := get(string(i))
		if !contains(activeNodes, new_node) {
			activeNodes = append(activeNodes, new_node)
		}
	}
	updateListofActiveNodes()
}

func checkActiveNodesAreActive() {
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

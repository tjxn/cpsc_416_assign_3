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
	"container/list"
	"fmt"
	"net/rpc"
	"os"
	"strconv"
	"time"
)

// Constants
var DELAY int = 5

// Global Variables
var kvService *rpc.Client
var lastNum int
var leaderNum int
var registerNum int
var myIDNum int
var myID string
var activeNodes *List

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

	switch result {

	case "unavailable":
		leaderNum++
		checkForLeader()

	case "":
		attemptToBecomeLeader()

	default:
		registerWithLeader()
	}
}

func attemptToBecomeLeader() {
	var result string = testset(getLeaderString(), "", myID)

	switch result {
	case "unavailable":
		checkForLeader()

	case myID:
		// success
		addIDtoActiveNodes(myID)

		go updateListofActiveNodes()
		go checkForNewNode()
		go checkActiveNodesAreActive()

	default:
		// failure
		checkForLeader()
	}
}

func registerWithLeader() {
	var result string = testset(getRegisterString(), "", myID)

	switch result {

	case "unavailable":
		incrementRegisterString()
		testset(getRegisterString(), "", myID)

	case myID:
		// success
		go getActiveNodes()
		go iAmAlive()

	default:
		// failure
		registerWithLeader()
	}
}

func getActiveNodes() {
	var result string = get(getLeaderString())

	switch result {
	case "unavailable":
		checkForLeader()

	case "":
		getActiveNodes()

	default:
		displayActiveNodes(result)
	}

}

func iAmActive() {
	var result string = get(getMyIDString())

	switch result {
	case "unavailable":
		incrementMyIDString()
		iAmActive()

	case "":
		put(getMyIDString(), lastNum)
		delaySecond(DELAY)
		iAmActive()

	case result > lastNum:
		incrementLastNum()
		put(getMyIDString(), getLastNum())
		iAmActive()

	case result == lastNum:
		isLeaderDead()

	default:
		fmt.Println("Error occured in iAmActive function switch statement")
		os.Exit(1)

	}
}

func isLeaderDead() {
	delaySecond(DELAY)

	var result string = get(getMyIDString())

	switch result {

	case "unavailable":
		incrementMyIDString()
		isLeaderDead()

	case "":
		put(getMyIDString(), getLastNumString())
		delaySecond(DELAY)
		iAmActive()

	case result > lastNum:
		incrementLastNumString()
		put(getMyIDString(), getLastNumString())
		delaySecond(DELAY)
		iAmActive()

	case result == lastNum:
		checkForLeader()

	}
}

func delaySecond(n time.Duration) {
	time.Sleep(n * time.Second)
}

func displayActiveNodes(string nodes) {
	fmt.Println(nodes)
}

func addIDtoActiveNodes(string id) {
	activeNodes.PushBack(id)
}

func removeIDfromActiveNodes(string id) {
	activeNodes.Remove(id)
}

func incrementLastNumString() {
	lastNum++
	result := strconv.Itoa(lastNum)
	return result
}

func getLastNumString() string {
	result := strconv.Itoa(lastNum)
	return result
}

func incrementMyIDString() {
	myIDNum++
	result := strconv.Itoa(myIDNum)
	return myID + "-" + result
}

func getMyIDString() string {
	result := strconv.Itoa(myIDNum)
	return myID + "-" + result
}

func incrementRegisterString() {
	registerNum++
	result := strconv.Itoa(registerNum)
	return "Register-" + result
}

func getRegisterString() string {
	result := strconv.Itoa(registerNum)
	return "Register-" + result
}

func incrementLeaderString() {
	leaderNum++
	result := strconv.Itoa(leaderNum)
	return "Leader-" + result
}

func getLeaderString() string {
	result := strconv.Itoa(leaderNum)
	return "Leader-" + result
}

func get(string key) string {

	var kvVal ValReply

	getArgs := GetArgs{key}

	err = kvService.Call("KeyValService.Get", getArgs, &kvVal)
	checkError(err)

	return kvVal.Val
}

func put(string key, string value) string {
	var kvVal ValReply

	putArgs := PutArgs{
		Key: key,
		Val: value,
	}

	err = kvService.Call("KeyValService.Put", putArgs, &kvVal)
	checkError(err)

	return kvVal.Val
}

func testset(string key, string value, string replacement) string {
	var kvVal ValReply

	tsArgs := TestSetArgs{
		Key:     key,
		TestVal: value,
		NewVal:  replacement,
	}

	err = kvService.Call("KeyValService.TestSet", tsArgs, &kvVal)
	checkError(err)

	return kvVal.Val
}

// Main server loop.
func main() {

	checkArgs()

	kvAddr := os.Args[1]
	myID = os.Args[2]

	// Connect to the KV-service via RPC.
	kvService, err := rpc.Dial("tcp", kvAddr)
	checkError(err)

	lastNum = 1
	leaderNum = 1
	registerNum = 1
	myIDNum = 1

	activeNodes = list.New()

	checkForLeader()

}

// If error is non-nil, print it out and halt.
func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error ", err.Error())
		os.Exit(1)
	}
}

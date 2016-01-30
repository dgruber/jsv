/*
Copyright (c) 2013, 2014, 2015 Daniel Gruber (dgruber@univa.com), Univa

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// Package jsv implements Univa Grid Engine's job submission verifier
// protocol.
package jsv

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// State represents the internal state within the JSV
// protocol.
type State int

const (
	initialized = iota
	started
	verifying
)

// State within the JSV processing
var state = initialized

var in *bufio.Reader
var out *bufio.Writer
var log *bufio.Writer

// LoggingEnabled turns logging on or off. Note that when the
// logfile can't be opened LoggingEnabled is set to false automatically.
// This can be used as a check in the JSV application. Don't change
// the setting during the runtime of a JSV application.
var LoggingEnabled = false
// Logfile is the path to the logfile which should be used
// for logging when LoggingEnabled is set to true.
var Logfile  = "/tmp/jsv_logfile.log"

// Available parameters:
// var jsv_cli_params = "a ar A b ckpt cwd C display dl e hard h hold_jid hold_jid_ad i inherit j jc js m M masterq notify now N noshell nostdin o ot P p pty R r shell sync S t tc terse u w wd"
// var jsv_mod_params = "ac l_hard l_soft masterl q_hard q_soft pe_min pe_max pe_name binding_strategy binding_type binding_amount binding_socket binding_core binding_step binding_exp_n"
// var jsv_add_params = "CLIENT CONTEXT GROUP VERSION JOB_ID SCRIPT CMDARGS USER"
// var jsv_all_params = jsv_cli_params + " " + jsv_mod_params + " " + jsv_add_params

// cached commands
var commandList map[string]string

// cached job environment
var environmentList map[string]string

// init initializes global variables
func init() {
	commandList = make(map[string]string)
	environmentList = make(map[string]string)
	in = bufio.NewReader(os.Stdin)
	out = bufio.NewWriter(os.Stdout)
}

// handleStartCommand is executed when Grid Engine sends the START
// command to the JSV script.
func handleStartCommand(checkEnvironment bool, jsvOnStartFunction func()) {
	if state == initialized {
		// execution of the function for getting the environment
		if jsvOnStartFunction != nil {
			jsvOnStartFunction()
		}
		sendCommand("STARTED")
		state = started
	} else {
		sendCommand("ERROR JSV script got START command bit is in state ...")
	}
}

// handleBeginCommand is executed when BEGIN was sent from Grid Engine
// to the JSV script.
func handleBeginCommand(verificationCommand func()) {
	if state == started {
		state = verifying
		// run administrators verification function
		if verificationCommand != nil {
			verificationCommand()
		}
		// clear all params and environment variables we got for the next run
		commandList = make(map[string]string)
		environmentList = make(map[string]string)
	} else {
		sendCommand("ERROR JSV script got BEGIN command but is in state ...")
	}
}

// scriptLog writes the given parameters to a logfile when defined.
func scriptLog(param string, param2 string) {
	if LoggingEnabled == true {
		log.WriteString(param)
		log.WriteString(param)
		log.Flush()
	}
}

// sendCommand sends the given parameter (command) to STDOUT.
func sendCommand(param string) {
	/* echo $@ */
	out.WriteString(param + "\n")
	out.Flush()
	scriptLog("<<< ", param)
}

// handleEnvCommand processes an enviornment variable sent from
// Grid Engine and stores it in a global map.
func handleEnvCommand(line string) {
	if state == started {
		tokens := strings.SplitN(line, " ", 4)
		if len(tokens) == 4 {
			if tokens[1] == "ADD" {
				// add a new variable
				environmentList[tokens[2]] = tokens[3]
			}
		}
	} else {
		sendCommand("ERROR JSV script got ENV command but is not in state STARTED")
	}
}

// filterJobClassSpec filters out substrings like "{~}" - which anyhow
// should not be send over the protocol.
func filterJobClassSpec(unfiltered string) string {
	if strings.Contains(unfiltered, "{") && strings.Contains(unfiltered, "}") {
		first := strings.Index(unfiltered, "{")
		last := strings.LastIndex(unfiltered, "}")
		var filteredCmd string
		for i, c := range unfiltered {
			if i < first || i > last {
				filteredCmd += string(c)
			}
		}
		return filteredCmd
	}
	return unfiltered
}

// handleParamCommand puts a job submission command from Grid Engine to
// a global map. (input is like PARAM <cmd> <value>)
func handleParamCommand(line string) {
	if state == started {
		tokens := strings.SplitN(line, " ", 3)
		if len(tokens) == 3 {
			// a hack for fixing a bug which comes from external
			if tokens[1] == "l_hard" || tokens[1] == "l_soft" {
				// filter possible {} (which should not be sent, but could be
				// in case of job classes)
				values := strings.Split(tokens[2], ",")
				newString := ""
				stringChanged := false
				for i, value := range values {
					if i > 0 {
						newString = newString + ","
					}
					// here we have h_rt=123 m_mem_free=1G
					request := strings.Split(value, "=")
					if len(request) == 2 {
						filtered := filterJobClassSpec(request[0])
						if filtered != request[0] {
							// bug found -> remove job class specifier
							stringChanged = true
							newString = newString + filtered + "=" + request[1]
							continue
						}
					}
					newString = newString + value
				}
				if stringChanged {
					commandList[tokens[1]] = newString
				} else {
					commandList[tokens[1]] = tokens[2]
				}
			} else {
				commandList[tokens[1]] = tokens[2]
			}
		} else if len(tokens) == 2 {
			commandList[tokens[1]] = ""
		} else {
			sendCommand("ERROR PARAM without any argument: " + line)
		}
	} else {
		sendCommand("ERROR JSV script got PARAM command but is not in STARTED state")
	}
}

// showParams logs the job submission parameters (for client side JSV on stdout).
func showParams() {
	for param := range commandList {
		name := "jsv_param_" + param
		sendCommand("LOG INFO got param " + name + "=" + commandList[param])
	}
}

// showEnvs logs the environment variables passed to the job (for client side JSV on stdout)
func showEnvs() {
	for env := range environmentList {
		name := "jsv_env_" + env
		sendCommand("LOG INFO got env " + name + "=" + environmentList[env])
	}
}

// Run is the main JSV function. Must be called by the JSV 'script'.
// requires the verification function to be passed. Optional
// a function which is run before the verification process can
// be passed or nil instead.
func Run(checkEnvironment bool, verificationFunction func(), onStartFunction func()) {
	/* here the traditional main loop runs (jsv_main) */

	/* while there is data from stdin and quit was not send */
	hasInput := true
	abort := false

	if verificationFunction == nil {
		panic("verification function is nil!")
	}

	// enable logging
	if LoggingEnabled {
		lf, err := os.Open(Logfile)
		if err != nil {
			// error logfile can't be opened - disable logging
			LoggingEnabled = false
		} else {
			log = bufio.NewWriter(lf)
		}
	}

	for hasInput && abort == false {
		/* get input from stdin */
		line, isPrefix, err := in.ReadLine()
		if err == nil && isPrefix == false {
			//out.Write(line)
			//out.Flush()

			/* ignore emtpy lines */
			if string(line) == "" {
				continue
			}
			// abort program as soon as quit is sent
			if string(line) == "QUIT" {
				abort = true
				break
			}

			// Grid Engine adds a new parameter
			if strings.HasPrefix(string(line), "PARAM") {
				handleParamCommand(string(line))
				continue
			}

			// Grid Engine adds a new environment variable
			if strings.HasPrefix(string(line), "ENV") {
				handleEnvCommand(string(line))
				continue
			}

			// Grid Engine sends a start -> state transition
			if strings.HasPrefix(string(line), "START") {
				handleStartCommand(checkEnvironment, onStartFunction)
				continue
			}

			// Grid Engine calls the JSV verification function
			if strings.HasPrefix(string(line), "BEGIN") {
				handleBeginCommand(verificationFunction)
				continue
			}

			if strings.HasPrefix(string(line), "SHOW") {
				showEnvs()
				showParams()
				continue
			}

			/* ERROR JSV script got unknown command ... */
			sendCommand("ERROR JSV script got unknown command xy")
			abort = true
		} else {
			/* buffer should always be big enough, we treat it like an input error */
			hasInput = false
		}
	}
}

// IsParam checks if the given parameter is requested by the job.
func IsParam(param string) bool {
	_, exists := GetParam(param)
	return exists
}

// GetParam returns the value of a simple job submission parameter
// which was requested by the job.
// Example: JSV_get_param("SCRIPT")
func GetParam(suffix string) (string, bool) {
	command, exists := commandList[suffix]
	return command, exists
}

// SetParam adds a simple job submission parameter.
func SetParam(suffix string, value string) {
	commandList[suffix] = value
	sendCommand("PARAM " + suffix + " " + value)
}

// DelParam deletes a simple job submission parameter.
func DelParam(suffix string) {
	// delete parameter only if it exists (got from master or sent)
	if _, exists := commandList[suffix]; exists {
		sendCommand("PARAM " + suffix)
	}
}

// SubIsParam returns true in case a specific sub
// parameter is set.
// Example: qsub -l h_vmem=1G ...
// jsv_sub_is_param("l", "h_vmem") == true
func SubIsParam(param, subParam string) bool {
	_, exists := SubGetParam(param, subParam)
	return exists
}

// SubGetParam returns the value of a sub-parameter.
// Example: qsub -l h_vmem=1G ...
// JSV_sub_get_param("l", "h_vmem") == "1G"
func SubGetParam(param, subParam string) (string, bool) {
	if value, exists := GetParam(param); exists {
		for _, pair := range strings.Split(value, ",") {
			sub := strings.Split(pair, "=")
			if sub[0] == subParam {
				return sub[1], true
			}
		}
	}
	return "", false
}

// SubDelParam deletes a sublist element from a list (like
// removing h_vmem from l_hard request list).
func SubDelParam(param, subParam string) {
	// only remove when the sub parameter is defined
	if subValue, exists := SubGetParam(param, subParam); exists {
		if subParamList, isParam := GetParam(param); isParam {
			// replace value with "" . remove ",," and "," at the beginning
			// or end of the string
			newSubList := strings.Replace(subParamList, subParam+"="+subValue, "", 1)
			cleanedUpSubList := strings.Replace(newSubList, ",,", ",", -1)
			// beginning and end
			if strings.HasPrefix(cleanedUpSubList, ",") {
				cleanedUpSubList = strings.Trim(cleanedUpSubList, ",")
			}
			SetParam(param, cleanedUpSubList)
		}
	}
}

// SubAddParam adds a new sublist parameter to a list.
// The expected parameter is a sub parameter of a parameter group,
// like a resource request (qsub -l h_vmem=1G ...). In this case the
// function would be called like:
// JSV_sub_add_param("l", "h_vmem", "1G")
func SubAddParam(param, subParam, value string) {
	// Add the parameter, overwrite, or append to the
	// sublist.
	if subValue, exists := SubGetParam(param, subParam); exists {
		// Overwrite this sub parameter if it is changed only.
		if subValue == value {
			// the old value is the same than the new one
			return
		}
		v, _ := GetParam(param)
		// replace "..,h_vmem=1G,..." by "..,h_vmem=2G,.."
		newSubList := strings.Replace(v, subParam+"="+subValue, subParam+"="+value, 1)
		SetParam(param, newSubList)

		return
	} else if subParamList, isParam := GetParam(param); isParam {
		// Append the sub parameter to existing list.
		SetParam(param, subParamList+","+subParam+"="+value)
		return
	}
	// Add parameter. It is the first sub-parameter.
	SetParam(param, subParam+"="+value)
}

// IsEnv returns true in the case the given environment variable
// was set for the job.
func IsEnv(envVar string) bool {
	_, exists := GetEnv(envVar)
	return exists
}

// GetEnv returns the value of an environment variable.
func GetEnv(envVar string) (string, bool) {
	env, exists := environmentList[envVar]
	return env, exists
}

// AddEnv adds an environment variable to a job.
func AddEnv(envVar, value string) {
	environmentList[envVar] = value
	sendCommand("ENV ADD " + envVar + " " + value)
}

// ModEnv modifies an environment variable of a job.
func ModEnv(envVar, value string) {
	environmentList[envVar] = value
	sendCommand("ENV MOD " + envVar + " " + value)
}

// DelEnv removes an environment variable from a job.
func DelEnv(envVar string) {
	if _, exists := environmentList[envVar]; exists {
		delete(environmentList, envVar)
		sendCommand("ENV DEL " + envVar)
	}
}

// SetTimeout overrides the timeout for server side
// JSVs specified in the SGE_JSV_TIMEOUT environment variable.
// The timeout is specified in seconds and must be greater
// than one. The command might only with Univa Grid Engine.
func SetTimeout(timeout int) {
	sendCommand(fmt.Sprintf("SEND TIMEOUT %d", timeout))
}

// Additional helpers: Not specified in JSV protocol
// -------------------------------------------------

// ListEnvs prints all environment variables on stdout.
func ListEnvs() {
	for key, value := range environmentList {
		fmt.Println("EV name:", key, "Value:", value)
	}
}

// TODO parameter for sublists

// Correct must be called in the JSV function when the job was modified
// and corrected. Currently it the same like jsv_accept().
func Correct(args string) {
	if state == verifying {
		sendCommand("RESULT STATE CORRECT " + args)
		state = initialized
	} else {
		sendCommand("ERROR jsv_correct() called in wrong state")
	}
}

// Accept must be called in the JSV function when the job is accepted.
// Alternativly Correct() can be called at the end of a JSV when
// the job was modified.
// Currently both have the same semantic only Java JSV differs in that.
func Accept(args string) {
	if state == verifying {
		sendCommand("RESULT STATE ACCEPT " + args)
		state = initialized
	} else {
		sendCommand("ERROR jsv_correct() called in wrong state")
	}
}

// Reject rejects a job. That means the job is not added
// to the qmasters job list. The argument specifies the
// reject message.
func Reject(args string) {
	if state == verifying {
		sendCommand("RESULT STATE REJECT " + args)
		state = initialized
	} else {
		sendCommand("ERROR jsv_correct() called in wrong state")
	}
}

// RejectWait rejects a job due to a temporary reason.
// Must be called when is job is going to be rejected because
// of a temporary reason. The only difference to jsv_reject() is
// that a different message is logged by Grid Engine.
func RejectWait(args string) {
	if state == verifying {
		sendCommand("RESULT STATE REJECT_WAIT " + args)
		state = initialized
	} else {
		sendCommand("ERROR jsv_correct() called in wrong state")
	}
}

// SendEnv can be called in the jsv_on_start function in order
// to let Grid Engine send all environment variables to the JSV script.
func SendEnv() {
	sendCommand("SEND ENV")
}

// LogInfo logs the string provided as argmument as info message.
// In case of an server side JSV the output appears in the messages
// file of qmaster if the log level allows it.
func LogInfo(message string) {
	sendCommand("LOG INFO " + message)
}

// LogWarning logs the string provided as argument as warning.
// In case of an server side JSV the output appears in the messages
// file of qmaster if the log level allows it.
func LogWarning(message string) {
	sendCommand("LOG WARNING " + message)
}

// LogError logs the string provided as argument as error.
// In case of an server side JSV the output appears in the messages
// file of qmaster if the log level allows it.
func LogError(message string) {
	sendCommand("LOG ERROR " + message)
}

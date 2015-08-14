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

package jsv

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

/* should environment be sent? */
var send_env = false

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

var logging_enabled = false
var logfile = "/tmp/jsv_logfile.log"

var jsv_cli_params = "a ar A b ckpt cwd C display dl e hard h hold_jid hold_jid_ad i inherit j jc js m M masterq notify now N noshell nostdin o ot P p pty R r shell sync S t tc terse u w wd"
var jsv_mod_params = "ac l_hard l_soft masterl q_hard q_soft pe_min pe_max pe_name binding_strategy binding_type binding_amount binding_socket binding_core binding_step binding_exp_n"
var jsv_add_params = "CLIENT CONTEXT GROUP VERSION JOB_ID SCRIPT CMDARGS USER"
var jsv_all_params = jsv_cli_params + " " + jsv_mod_params + " " + jsv_add_params

// cached commands
var command_list map[string]string

// cached job environment
var environment_list map[string]string

// init initializes global variables
func init() {
	command_list = make(map[string]string)
	environment_list = make(map[string]string)
	in = bufio.NewReader(os.Stdin)
	out = bufio.NewWriter(os.Stdout)
}

// jsv_hande_start_command is executed when Grid Engine sends the START
// command to the JSV script.
func jsv_handle_start_command(checkEnvironment bool, jsvOnStartFunction func()) {
	if state == initialized {
		// execution of the function for getting the environment
		if jsvOnStartFunction != nil {
			jsvOnStartFunction()
		}
		jsv_send_command("STARTED")
		state = started
	} else {
		jsv_send_command("ERROR JSV script got START command bit is in state ...")
	}
}

// jsv_handle_begin_command is executed when BEGIN was sent from Grid Engine
// to the JSV script.
func jsv_handle_begin_command(verificationCommand func()) {
	if state == started {
		state = verifying
		// run administrators verification function
		if verificationCommand != nil {
			verificationCommand()
		}
		// clear all params and environment variables we got for the next run
		command_list = make(map[string]string)
		environment_list = make(map[string]string)
	} else {
		jsv_send_command("ERROR JSV script got BEGIN command but is in state ...")
	}
}

// jsv_script_log writes the given parameters to a logfile when defined.
func jsv_script_log(param string, param2 string) {
	if logging_enabled == true {
		log.WriteString(param)
		log.WriteString(param)
		log.Flush()
	}
}

// jsv_send_commannd sends the given parameter (command) to STDOUT.
func jsv_send_command(param string) {
	/* echo $@ */
	out.WriteString(param + "\n")
	out.Flush()
	jsv_script_log("<<< ", param)
}

// jsv_handle_env_command processes an enviornment variable sent from
// Grid Engine and stores it in a global map.
func jsv_handle_env_command(line string) {
	if state == started {
		tokens := strings.SplitN(line, " ", 4)
		if len(tokens) == 4 {
			if tokens[1] == "ADD" {
				// add a new variable
				environment_list[tokens[2]] = tokens[3]
			}
		}
	} else {
		jsv_send_command("ERROR JSV script got ENV command but is not in state STARTED")
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

// jsv_handle_param_command puts a job submission command from Grid Engine to
// a global map. (input is like PARAM <cmd> <value>)
func jsv_handle_param_command(line string) {
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
					command_list[tokens[1]] = newString
				} else {
					command_list[tokens[1]] = tokens[2]
				}
			} else {
				command_list[tokens[1]] = tokens[2]
			}
		} else if len(tokens) == 2 {
			command_list[tokens[1]] = ""
		} else {
			jsv_send_command("ERROR PARAM without any argument: " + line)
		}
	} else {
		jsv_send_command("ERROR JSV script got PARAM command but is not in STARTED state")
	}
}

// JSV_show_params logs the job submission parameters (for client side JSV on stdout).
func JSV_show_params() {
	for param := range command_list {
		name := "jsv_param_" + param
		jsv_send_command("LOG INFO got param " + name + "=" + command_list[param])
	}
}

// JSV_show_envs logs the environment variables passed to the job (for client side JSV on stdout)
func JSV_show_envs() {
	for env := range environment_list {
		name := "jsv_env_" + env
		jsv_send_command("LOG INFO got env " + name + "=" + environment_list[env])
	}
}

// Run is the main JSV function. Must be called by the JSV 'script'.
// requires the verification function to be passed. Optional
// a function which is run before the verification process can
// be passed or nil instead.
func Run(checkEnvironment bool, verificationFunction func(), jsv_on_start_function func()) {
	/* here the traditional main loop runs (jsv_main) */

	/* while there is data from stdin and quit was not send */
	has_input := true
	abort := false

	if verificationFunction == nil {
		panic("verification function is nil!")
	}

	for has_input && abort == false {
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
				jsv_handle_param_command(string(line))
				continue
			}

			// Grid Engine adds a new environment variable
			if strings.HasPrefix(string(line), "ENV") {
				jsv_handle_env_command(string(line))
				continue
			}

			// Grid Engine sends a start -> state transition
			if strings.HasPrefix(string(line), "START") {
				jsv_handle_start_command(checkEnvironment, jsv_on_start_function)
				continue
			}

			// Grid Engine calls the JSV verification function
			if strings.HasPrefix(string(line), "BEGIN") {
				jsv_handle_begin_command(verificationFunction)
				continue
			}

			if strings.HasPrefix(string(line), "SHOW") {
				JSV_show_envs()
				JSV_show_params()
				continue
			}

			/* ERROR JSV script got unknown command ... */
			jsv_send_command("ERROR JSV script got unknown command xy")
			abort = true
		} else {
			/* buffer should always be big enough, we treat it like an input error */
			has_input = false
		}
	}
}

// JSV_is_param checks if the given parameter is requested by the job.
func JSV_is_param(param string) bool {
	_, exists := JSV_get_param(param)
	return exists
}

// JSV_get_param returns the value of the parameter which was requested by the job.
// Example: JSV_get_param("SCRIPT")
func JSV_get_param(suffix string) (string, bool) {
	command, exists := command_list[suffix]
	return command, exists
}

// sets a job submission parameter
func JSV_set_param(suffix string, value string) {
	command_list[suffix] = value
	jsv_send_command("PARAM " + suffix + " " + value)
}

// delete a job submission parameter
func JSV_del_param(suffix string) {
	// delete parameter only if it exists (got from master or sent)
	if _, exists := command_list[suffix]; exists {
		jsv_send_command("PARAM " + suffix)
	}
}

// Returns true in case a specific sub parameter is set.
// Example: qsub -l h_vmem=1G ...
// jsv_sub_is_param("l", "h_vmem") == true
func JSV_sub_is_param(param, subParam string) bool {
	_, exists := JSV_sub_get_param(param, subParam)
	return exists
}

// Returns the value of a sub-parameter.
// Example: qsub -l h_vmem=1G ...
// JSV_sub_get_param("l", "h_vmem") == "1G"
func JSV_sub_get_param(param, subParam string) (string, bool) {
	if value, exists := JSV_get_param(param); exists {
		for _, pair := range strings.Split(value, ",") {
			sub := strings.Split(pair, "=")
			if sub[0] == subParam {
				return sub[1], true
			}
		}
	}
	return "", false
}

func JSV_sub_del_param(param, subParam string) {
	// only remove when the sub parameter is defined
	if subValue, exists := JSV_sub_get_param(param, subParam); exists {
		if subParamList, isParam := JSV_get_param(param); isParam {
			// replace value with "" . remove ",," and "," at the beginning
			// or end of the string
			newSubList := strings.Replace(subParamList, subParam+"="+subValue, "", 1)
			cleanedUpSubList := strings.Replace(newSubList, ",,", ",", -1)
			// beginning and end
			if strings.HasPrefix(cleanedUpSubList, ",") {
				cleanedUpSubList = strings.Trim(cleanedUpSubList, ",")
			}
			JSV_set_param(param, cleanedUpSubList)
		}
	}
}

// Adds a new job submission paramter to the job. The expected
// parameter is a sub parameter of a parameter group, like a
// resource request (qsub -l h_vmem=1G ...). In this case the
// function would be called like:
// JSV_sub_add_param("l", "h_vmem", "1G")
func JSV_sub_add_param(param, subParam, value string) {
	// Add the parameter, overwrite, or append to the
	// sublist.
	if subValue, exists := JSV_sub_get_param(param, subParam); exists {
		// Overwrite this sub parameter if it is changed only.
		if subValue == value {
			// the old value is the same than the new one
			return
		} else {
			v, _ := JSV_get_param(param)
			// replace "..,h_vmem=1G,..." by "..,h_vmem=2G,.."
			newSubList := strings.Replace(v, subParam+"="+subValue, subParam+"="+value, 1)
			JSV_set_param(param, newSubList)
		}
		return
	} else if subParamList, isParam := JSV_get_param(param); isParam {
		// Append the sub parameter to existing list.
		JSV_set_param(param, subParamList+","+subParam+"="+value)
		return
	}
	// Add parameter. It is the first sub-parameter.
	JSV_set_param(param, subParam+"="+value)
}

// JSV_is_env returns true in the case the given environment variable
// was set for the job.
func JSV_is_env(envVar string) bool {
	_, exists := JSV_get_env(envVar)
	return exists
}

// JSV_get_env returns the value of an environment variable.
func JSV_get_env(envVar string) (string, bool) {
	env, exists := environment_list[envVar]
	return env, exists
}

// JSV_add_env adds an environment variable to a job.
func JSV_add_env(envVar, value string) {
	environment_list[envVar] = value
	jsv_send_command("ENV ADD " + envVar + " " + value)
}

// JSV_mod_env modifies an environment variable of a job.
func JSV_mod_env(envVar, value string) {
	environment_list[envVar] = value
	jsv_send_command("ENV MOD " + envVar + " " + value)
}

// JSV_del_env removes an environment variable from a job.
func JSV_del_env(envVar string) {
	if _, exists := environment_list[envVar]; exists {
		delete(environment_list, envVar)
		jsv_send_command("ENV DEL " + envVar)
	}
}

// JSV_set_timeout overrides the timeout for server side
// JSVs specified in the SGE_JSV_TIMEOUT environment variable.
// The timeout is specified in seconds and must be greater
// than one. The command might only with Univa Grid Engine.
func JSV_set_timeout(timeout int) {
	jsv_send_command(fmt.Sprintf("SEND TIMEOUT %d", timeout))
}

// Additional helpers: Not specified in JSV protocol
// -------------------------------------------------

// JSV_list_env prints all environment variables on stdout.
func JSV_list_env() {
	for key, value := range environment_list {
		fmt.Println("EV name:", key, "Value:", value)
	}
}

// TODO parameter for sublists

// JSV_correct must be called in the JSV function when the job was modified
// and corrected. Currently it the same like jsv_accept().
func JSV_correct(args string) {
	if state == verifying {
		jsv_send_command("RESULT STATE CORRECT " + args)
		state = initialized
	} else {
		jsv_send_command("ERROR jsv_correct() called in wrong state")
	}
}

// JSV_accept must be called in the JSV function when the job is accepted.
// Alternativly jsv_correct() can be called when a job was modified.
// Currently both have the same sematic.
func JSV_accept(args string) {
	if state == verifying {
		jsv_send_command("RESULT STATE ACCEPT " + args)
		state = initialized
	} else {
		jsv_send_command("ERROR jsv_correct() called in wrong state")
	}
}

// JSV_reject rejects a job.
func JSV_reject(args string) {
	if state == verifying {
		jsv_send_command("RESULT STATE REJECT " + args)
		state = initialized
	} else {
		jsv_send_command("ERROR jsv_correct() called in wrong state")
	}
}

// JSV_reject_wait rejects a job due to a temporary reason.
// Must be called when is job is going to be rejected because
// of a temporary reason. The only difference to jsv_reject() is
// that a different message is logged by Grid Engine.
func JSV_reject_wait(args string) {
	if state == verifying {
		jsv_send_command("RESULT STATE REJECT_WAIT " + args)
		state = initialized
	} else {
		jsv_send_command("ERROR jsv_correct() called in wrong state")
	}
}

// JSV_send_env can be called in the jsv_on_start function in order
// to let Grid Engine send all environment variables to the JSV script.
func JSV_send_env() {
	jsv_send_command("SEND ENV")
}

// JSV_log_info logs the string povided as argmument as info message.
func JSV_log_info(message string) {
	jsv_send_command("LOG INFO " + message)
}

// JSV_log_warning logs the string provided as argument as warning.
func JSV_log_warning(message string) {
	jsv_send_command("LOG WARNING " + message)
}

// JSV_log_error logs the string provided as argument as error.
func JSV_log_error(message string) {
	jsv_send_command("LOG ERROR " + message)
}

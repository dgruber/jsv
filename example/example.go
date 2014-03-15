/*
Copyright (c) 2013, 2014, Daniel Gruber (dgruber@univa.com), Univa

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

package main

import (
	"github.com/dgruber/jsv"
)

func jsv_on_start_function() {
	//jsv_send_env()
}

func job_verification_function() {
	// setting -binding linear:1 to each job (so that each
	// job can only use one core on the compute node)
	jsv.JSV_set_param("binding_strategy", "linear_automatic")
	jsv.JSV_set_param("binding_type", "set")
	jsv.JSV_set_param("binding_amount", "1")
	jsv.JSV_set_param("binding_exp_n", "0")

	// Can be used for displaying submission parameters and
	// submission environment variables.
	//jsv_show_params()
	//jsv_show_envs()

	// Can be used with server side JSV script to log
	// in qmaster messages file. For client side JSV
	// scripts to print out some messages when doing
	// qsub.
	//jsv.JSV_log_info("info message")
	//jsv.JSV_log_warning("warning message")
	//jsv.JSV_log_error("error message")

	// accepting the job but indicating that we did
	// some changes
	jsv.JSV_correct("Job was modified")
	return
}

/* example JSV 'script' */
func main() {
	jsv.Run(true, job_verification_function, jsv_on_start_function)
}

package main

import (
	"strconv"

	"github.com/dgruber/jsv"
)

func jsvOnStartFunction() {
	// As we do not need to process any environment variables,
	// we can skip this step.
	//jsv_send_env()
}

func jsvVerificationFunction() {

	// check if job has requested the queue "long.q"
	if !jsv.SubIsParam("q_hard", "long.q") {
		jsv.Accept("No long.q job")
		return
	}

	// check if the runtime limit is at least 10 minutes
	runtimeLimit, exists := jsv.SubGetParam("l_hard", "h_rt")
	if !exists {
		jsv.Reject("No hard runtime limit requested (h_rt)")
		return
	}
	runtimeLimitInt, err := strconv.Atoi(runtimeLimit)
	if err != nil {
		jsv.Reject("Unexpected runtime limit: " + runtimeLimit)
		return
	}

	if runtimeLimitInt < 600 {
		jsv.SubAddParam("l_hard", "h_rt", "600")
		jsv.Correct("Runtime limit was increased to 10 minutes")
		return
	}

	jsv.Accept("Job accepted")
}

func main() {
	jsv.Run(true, jsvVerificationFunction, jsvOnStartFunction)
}

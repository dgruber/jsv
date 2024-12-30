# jsv

[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/dgruber/jsv)
[![Go Report Card](http://goreportcard.com/badge/github.com/dgruber/jsv)](http://goreportcard.com/report/github.com/dgruber/jsv)

Grid Engine JSV (Job Submission Verifier) implementation for Go (#golang).

Check out the [Open Cluster Scheduler](https://github.com/hpc-gridware/clusterscheduler/) as
the successor of open source Grid Engine and the [Gridware Cluster Scheduler](http://www.hpc-gridware.com/)
as long term support version. Both aim to be fully compatible with Grid Engine and provide a lot
of new, modern features.

## What is it?

JSV or Job Submission Verifiers are a part of the Grid Engine cluster scheduler eco system.
JSV scripts or binaries are executed after a job was submitted and before the job is
accepted by the cluster scheduler / manager (the Grid Engine master process).
They are a powerful tool for an administrator to inspect, correct, and set job
submission parameters for jobs based on his own logic.

An example would be only allowing jobs with a certain sizes (number of cores/
slots requested) at a certain time. Another one would be adding a predefined
dynamically created accounting string for each job. JSV scripts can also be used
for gathering job submission statistics or draining the cluster by rejecting all
jobs before a cluster upgrade happens.

For more information please consult your Univa Grid Engine documentation
and the man pages ([JSV man page](http://gridengine.eu/mangridengine/htmlman1/jsv.html)
and [qsub man page](http://gridengine.eu/mangridengine/htmlman1/qsub.html)).

## Why using this library?

JSV helper functions are available for TCL, bash, perl etc. Go is a compiled
language which does not rely on a external runtime system (JVM).
It enforces strict typing and is simple and lean. This makes it an ideal candidate
for implementing JSV "scripts" in little Go programs. Performance measurements
also showed that Go is the fastest available option for JSV. This is critical
since a JSV script is usually executed for each submitted job.

## How to use it?

Once you have this library, please consult the example in the examples
directory. Please consult the JSV documentation of Grid Engine for a
more detailed description.

## Example

Go to examples directory. Compile the example:

    git clone https://github.com/dgruber/jsv.git
    cd jsv/examples/simple

    go build example.go

The example adds the *-binding linear:1* core binding request to the *qsub* job submission.

In order to use it as client side JSV add the binary on *qsub* command line.

    qsub -jsv ./example -b y /bin/sleep 123

Check the *qsub* parameters in *qstat*. It added the new parameter.

An administrator certainly wants to enforce rules encoded in the JSV application for
*each* job, even when it not requests the *-jsv* parameter. Grid Engine /
**Gridware Cluster Scheduler** allows to configure the location of a global JSV
script executed for all jobs by the master process.

This is called *server sided JSV*.

In order to use your application as server side JSV you need to be Grid Engine
admin user. Set the path to the binary in *jsv_url* in the global configuration
*qconf -mconf global*. Then it is executed for each submitted job automatically.

    qsub -b y /bin/sleep 123

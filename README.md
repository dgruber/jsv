jsv
===

Grid Engine JSV (Job Submission Verifier) implementation for Go (#golang).


## What is it?

JSV or Job Submission Verifiers are a part of the Grid Engine cluster scheduler eco system. JSV scripts or binaries are be executed after a job was submitted to the cluster and before the job is accepted by the cluster scheduler / manager (the Grid Engine master process). They are a powerful tool for an administrator to _inspect_, _correct_, and _set_ job submission parameters for jobs based on his own logic. 

An example of using JSVs would be restricting jobs of certain sizes (e.g. based on number of requested cores or slots) for being submitted at a certain peak times. Another example is be adding a predefined dynamically created accounting string for each job.

Note that the code is not really Go style. Is was kept as close as possible to JSV implementations available in other programming languages. But I'm thinking to make it more Go-ish in the future. So please vendor the library in your project to avoid complications.

## Why using this library?

JSV helper functions are already available for Java, TCL, bash, perl etc. Go is a compiled language which does not rely on a external runtime system (JVM). It enforces strict typing and is simple and lean. This makes it an ideal candidate for implementing JSV "scripts" in little Go programs. Performance measurements also showed that Go is the fastest available option for JSV. This is critical since a JSV script is usually executed for each submitted job.

## How to use it?

Once you have this library, please consult the example in the examples directory. Please consult the JSV documentation of Grid Engine for a more detailed description.

## Example

Go to examples directory. Compile the example:

    go build example.go
    
The example adds the *-binding linear:1* core binding request to the *qsub* job submission.

In order to use it as client side JSV add the binary on *qsub* command line.

    qsub -jsv ./example -b y /bin/sleep 123
    
Check the *qsub* parameters in *qstat*. It added the new parameter.

An administrator certainly wants to enforce rules encoded in the JSV application for 
*each* job, even when it not requests the *-jsv* parameter. Grid Engine allows to configure
the location of a global JSV script executed for all jobs by the master process.
This is calles *server sided JSV*.

In order to use your application as server side JSV you need to be Grid Engine admin user. Set the path
to the binary in _jsv_url_ in the global configuration *qconf -mconf global*. Then it is 
executed for each submitted job automatically.

    qsub -b y /bin/sleep 123

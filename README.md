jsv
===

Grid Engine JSV (Job Submission Verifier) implementation for Go (#golang).


## What is it?

JSV or Job Submission Verifiers are a part of the Grid Engine cluster scheduler eco system. JSV scripts or binariers can be executed after a job was submitted to the cluster and before the job is accepted by the cluster scheduler / manager (the Grid Engine master process). They are a powerfull tool for an administrator to inspect, correct, and set job submission parameters for jobs based on his own logic. One example would be only allowing jobs with a certain sizes (number of cores / slots requested) at a certain time. Another one would be adding an certain accounting string for each job.

## Why using this library?

JSV helper functions are already available for Java, TCL, bash, perl etc. Go is a compiled language which does not rely on a external runtime system (JVM). It enforces strict typing and is simple and lean. This makes it an ideal candidate for implementing JSV "scripts" in little Go programs. Performance measurements also showed that Go is the fastest available option for JSV. This is critical since a JSV script is usually executed for each submitted job.

## How to use it?

Once you have this library, please consult the exmaple in the examples directory. Please consult the JSV documentation of Grid Engine for a more detailed description.

## Example

Go to examples directory. Compile the example:

    go build example.go
    
The example adds the *-binding linear:1* core binding request to the *qsub* job submission.

In order to use it as client side JSV add the binary on *qsub* command line.

    qsub -jsv ./example -b y /bin/sleep 123
    
Check the *qsub* parameters in *qstat*. It added the new parameter.

In order to use it as server side JSV you need to be Grid Engine admin user. Set the path
to the binary in _jsv_url_ in the global configuration *qconf -mconf global*. Then it is 
executed for each job submitted.

    qsub -b y /bin/sleep 123

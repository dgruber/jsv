jsv
===

Grid Engine JSV (Job Submission Verifier) implementation for Go (#golang).


## What is it?

JSV or Job Submission Verifiers are a part of the Grid Engine cluster scheduler eco system. JSV scripts or binariers can be executed after a job was submitted to the cluster and before the job is accepted by the cluster scheduler / manager (the Grid Engine master process). They are a powerfull tool for an administrator to inspect, correct, and set job submission parameters for jobs based on his own logic. One example would be only allowing jobs with a certain sizes (number of cores / slots requested) at a certain time. Another one would be adding an certain accounting string for each job.

## Why using this library?

JSV helper functions are already available for Java, TCL, bash, perl etc. Go is a compiled language which does not rely on a external runtime system (JVM). It enforces strict typing and is simple and lean. This makes it an ideal candidate for implementing JSV "scripts" in little Go programs. Performance measurements also showed that Go is the fastest available option for JSV. This is critical since a JSV script is usually executed for each submitted job.

## How to use it?

Once you have this library, please consult the exmaple in the examples directory. Please consult the JSV documentation of Grid Engine for a more detailed description.


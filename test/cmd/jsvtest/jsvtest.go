package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/dgruber/jsv/test/jsvserver"
)

func main() {
	// Check for required arguments.
	// The first argument: path to the JSV script.
	// The second optional argument: directory with simulated job specifications.
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <path-to-jsv-script> [job-spec-dir]", os.Args[0])
	}

	jsvScript := os.Args[1]
	server, err := jsvserver.NewJSVTestServer(jsvScript)
	if err != nil {
		log.Fatal(err)
	}
	// Ensure the server is stopped when main exits.
	defer func() {
		if err := server.Stop(); err != nil {
			log.Println("Error stopping server:", err)
		}
	}()

	if err := server.Start(); err != nil {
		log.Fatal(err)
	}

	var job *jsvserver.JobSpec

	// If a job specification directory is provided, loading a job
	// specification from it.
	if len(os.Args) >= 3 {
		jobSpecDir := os.Args[2]
		files, err := os.ReadDir(jobSpecDir)
		if err != nil {
			log.Fatalf("Failed to read job spec directory %q: %v", jobSpecDir, err)
		}
		if len(files) == 0 {
			log.Fatalf("No job specifications found in directory: %s", jobSpecDir)
		}

		jobSpecs := make([]jsvserver.JobSpec, 0, len(files))

		// for all json files in the directory, load the job specification
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			if filepath.Ext(file.Name()) != ".json" {
				continue
			}
			simulatedJobSpecFile := filepath.Join(jobSpecDir, file.Name())
			log.Printf("Using simulated job specification from file: %s", simulatedJobSpecFile)
			// Read the job specification file content.
			data, err := os.ReadFile(simulatedJobSpecFile)
			if err != nil {
				log.Fatalf("Failed to read job specification file %q: %v", simulatedJobSpecFile, err)
			}
			// Parse the job specification from the file into a JobSpec struct.
			var jobSpec jsvserver.JobSpec
			if err := json.Unmarshal(data, &jobSpec); err != nil {
				log.Fatalf("Failed to parse job specification from file %q: %v", simulatedJobSpecFile, err)
			}
			jobSpecs = append(jobSpecs, jobSpec)
		}
		start := time.Now()
		for _, jobSpec := range jobSpecs {
			_, err := server.SendJob(&jobSpec)
			if err != nil {
				log.Printf("Job verification failed: %v", err)
			}
		}
		end := time.Now()
		log.Printf("Time taken: %v", end.Sub(start))
	} else {
		// No job specification directory provided; use the hardcoded job specification.
		log.Println("No job specification directory provided; using hardcoded job specification")
		job = &jsvserver.JobSpec{
			Context: "client",
			Client:  "qsub",
			User:    "testuser",
			Group:   "testgroup",
			CmdName: "/path/to/script.sh",
			CmdArgs: 1,
			Params: map[string]string{
				"l_hard":  "h_rt=99",
				"pe_name": "mpi",
				"pe_min":  "3",
				"pe_max":  "3",
				"q_hard":  "long.q",
			},
			Environment: map[string]string{
				"PATH": "/usr/bin:/bin",
				"USER": "testuser",
			},
		}

		_, err := server.SendJob(job)
		if err != nil {
			log.Printf("Job verification failed: %v", err)
		}
	}

}

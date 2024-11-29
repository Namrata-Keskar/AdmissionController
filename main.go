package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"

	"golang.org/x/exp/slog"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// JSON patch operation for Kubernetes API objects
type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// Helper function to log errors and send an HTTP 400 response
func httpError(w http.ResponseWriter, err error) {
	slog.Error("unable to complete request", "error", err.Error())
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(err.Error()))
}

// Parses incoming HTTP request into an AdmissionReview struct
func parseAdmissionReview(req *http.Request) (*admissionv1.AdmissionReview, error) {
	reqData, err := io.ReadAll(req.Body)
	if err != nil {
		log.Print("error reading request body", err)
		return nil, err
	}

	admissionReviewRequest := &admissionv1.AdmissionReview{}

	err = json.Unmarshal(reqData, admissionReviewRequest)
	if err != nil {
		log.Printf("Error deserializing request: %v", err)
		return nil, err
	}
	return admissionReviewRequest, nil
}

// Processes AdmissionReview requests, calculates resource requests, and logs them
func handleAdmissionReview(w http.ResponseWriter, r *http.Request) {
	log.Println("In handleAdmissionReview ...")

	admissionReviewRequest, err := parseAdmissionReview(r)

	// Make sure the incoming request is for a Pod
	if admissionReviewRequest.Request.Kind.Kind != "Pod" {
		httpError(w, fmt.Errorf("expected request for kind Pod but got %s", admissionReviewRequest.Request.Kind.Kind))
		return
	}

	// Deserialize the Pod object from the request
	pod := corev1.Pod{}
	err = json.Unmarshal(admissionReviewRequest.Request.Object.Raw, &pod)
	if err != nil {
		httpError(w, err)
		return
	}

	log.Println("Successfully decoded AdmissionReview")

	// Calculate total CPU and memory requested by all containers
	var totalMemory, totalCPU resource.Quantity
	for _, container := range pod.Spec.Containers {
		// Accumulate CPU requests
		if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
			log.Printf("Container %s requests %s of CPU", container.Name, cpu.String())
			totalCPU.Add(cpu)
		}
		// Accumulate memory requests
		if memory, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
			log.Printf("Container %s requests %s of memory", container.Name, memory.String())
			totalMemory.Add(memory)
		}
	}

	// Log the total CPU and memory requested
	log.Printf("Total Memory Requested: %s\n", totalMemory.String())
	log.Printf("Total CPU Requested: %s\n", totalCPU.String())

	// Create an AdmissionResponse to allow the pod creation
	admissionResponse := &admissionv1.AdmissionResponse{
		UID:     admissionReviewRequest.Request.UID,
		Allowed: true,
	}

	// Wrap the response in an AdmissionReview and send it back
	admissionReviewResponse := admissionv1.AdmissionReview{
		Response: admissionResponse,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(admissionReviewResponse); err != nil {
		log.Printf("could not encode response: %v", err)
		http.Error(w, "could not encode response", http.StatusInternalServerError)
	}
}

func openLogFile(path string) (*os.File, error) {
	logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return logFile, nil
}

func main() {
	fmt.Println("Running admission controller")

	// Create log file
	file, err := openLogFile("./mylog.log")
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(file)
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	log.Println("Log file created")

	// Print general stats
	log.Printf("Number of CPUs: %d\n", runtime.NumCPU())
	log.Printf("Number of goroutines: %d\n", runtime.NumGoroutine())

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	log.Printf("Total allocated memory: %d bytes\n", mem.TotalAlloc)
	log.Printf("Number of memory allocations: %d\n", mem.Mallocs)

	// Start HTTP server on /validate path
	http.HandleFunc("/validate", handleAdmissionReview)
	log.Printf("Starting admission controller on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

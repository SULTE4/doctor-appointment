package client

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"appointment-service/internal/usecase"
)

type doctorServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewDoctorServiceClient(baseURL string) usecase.DoctorServiceClient {
	return &doctorServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *doctorServiceClient) DoctorExists(doctorID string) (bool, error) {
	url := fmt.Sprintf("%s/doctors/%s", c.baseURL, doctorID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		log.Printf("[ERROR] Doctor Service unreachable at %s: %v", url, err)
		return false, fmt.Errorf("Doctor Service is currently unavailable: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		log.Printf("[ERROR] Doctor Service unexpected status %d for doctor %s", resp.StatusCode, doctorID)
		return false, fmt.Errorf("Doctor Service returned unexpected status %d", resp.StatusCode)
	}
}

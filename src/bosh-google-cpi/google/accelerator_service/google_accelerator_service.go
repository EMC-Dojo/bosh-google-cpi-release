package accelerator

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	computebeta "google.golang.org/api/compute/v0.beta"
)

const googleAcceleratorServiceLogTag = "GoogleAcceleratorService"

type GoogleAcceleratorService struct {
	project        string
	computeService *computebeta.Service
	logger         boshlog.Logger
}

func NewGoogleAcceleratorService(
	project string,
	computeService *computebeta.Service,
	logger boshlog.Logger,
) GoogleAcceleratorService {
	return GoogleAcceleratorService{
		project:        project,
		computeService: computeService,
		logger:         logger,
	}
}

package fakse

import (
	"bosh-google-cpi/google/accelerator_service"
)

type FakeAcceleratorService struct {
	FindCalled      bool
	FindFound       bool
	FindAccelerator accelerator.Accelerator
	FindErr         error
}

func (a *FakeAcceleratorService) Find(acceleratorType string, zone string) (accelerator.Accelerator, bool, error) {
	a.FindCalled = true
	return a.FindAccelerator, a.FindFound, a.FindErr
}

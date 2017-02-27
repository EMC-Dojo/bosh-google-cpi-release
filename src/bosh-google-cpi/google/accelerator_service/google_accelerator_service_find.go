package accelerator

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	"bosh-google-cpi/util"

	"google.golang.org/api/googleapi"
)

func (a GoogleAcceleratorService) Find(acceleratorType string, zone string) (Accelerator, bool, error) {
	a.logger.Debug(googleAcceleratorServiceLogTag, "Finding Google Accelerator '%s' in zone '%s'", acceleratorType, zone)
	acceleratorItem, err := a.computeService.AcceleratorTypes.Get(a.project, util.ResourceSplitter(zone), acceleratorType).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			return Accelerator{}, false, nil
		}

		return Accelerator{}, false, bosherr.WrapErrorf(err, "Failed to find Google Accelerator Type '%s' in zone '%s'", acceleratorType, zone)
	}

	a.logger.Info(googleAcceleratorServiceLogTag, "Found returned Google Accelerator '%+v'", acceleratorItem)

	accelerator := Accelerator{
		Type:     acceleratorItem.Name,
		Limit:    acceleratorItem.MaximumCardsPerInstance,
		Zone:     acceleratorItem.Zone,
		SelfLink: acceleratorItem.SelfLink,
	}
	return accelerator, true, nil
}

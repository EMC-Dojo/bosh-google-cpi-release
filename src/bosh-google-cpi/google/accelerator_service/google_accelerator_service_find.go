package accelerator

import (
	"google.golang.org/api/googleapi"
)

func (a GoogleAcceleratorService) Find(name string, zone string) (Accelerator, bool, error) {
	a.logger.Debug(googleAcceleratorServiceLogTag, "Finding Google Accelerator '%s' in zone '%s'", string, zone)
	acceleratorItem, err := a.computeService.AcceleratorTypes.Get(a.project, util.ResourceSpliter(zone), name)
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			return Accelerator{}, false, nil
		}

		return Accelerator{}, false, bosherr.WrapErrorf(err, "Failed to find Google Accelerator Type '%s' in zone '%s'", name, zone)
	}

	accelerator := Accelerator{
		Name:  acceleratorItem.Name,
		Limit: acceleratorItem.MaximumCardsPerInstance,
		Zone:  acceleratorItem.Zone,
	}
	return accelerator, true, nil
}

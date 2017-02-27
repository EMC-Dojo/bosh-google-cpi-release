package accelerator

type Service interface {
	Find(acceleratorType string, zone string) (Accelerator, bool, error)
}

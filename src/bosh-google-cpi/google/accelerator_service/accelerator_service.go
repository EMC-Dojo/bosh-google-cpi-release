package accelerator

type Service interface {
  Find(type string) (Accelerator, bool, error)
}
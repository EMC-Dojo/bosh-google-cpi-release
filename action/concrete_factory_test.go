package action_test

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/frodenas/bosh-google-cpi/action"

	clientfakes "github.com/frodenas/bosh-google-cpi/google/client/fakes"

	"github.com/frodenas/bosh-google-cpi/google/address_service"
	"github.com/frodenas/bosh-google-cpi/google/client"
	"github.com/frodenas/bosh-google-cpi/google/disk_service"
	"github.com/frodenas/bosh-google-cpi/google/disk_type_service"
	"github.com/frodenas/bosh-google-cpi/google/image_service"
	"github.com/frodenas/bosh-google-cpi/google/instance_service"
	"github.com/frodenas/bosh-google-cpi/google/machine_type_service"
	"github.com/frodenas/bosh-google-cpi/google/network_service"
	"github.com/frodenas/bosh-google-cpi/google/operation_service"
	"github.com/frodenas/bosh-google-cpi/google/snapshot_service"
	"github.com/frodenas/bosh-google-cpi/google/target_pool_service"

	"github.com/frodenas/bosh-registry/client"
)

var _ = Describe("ConcreteFactory", func() {
	var (
		uuidGen      *fakeuuid.FakeGenerator
		googleClient client.GoogleClient
		logger       boshlog.Logger

		options = ConcreteFactoryOptions{
			Registry: registry.ClientOptions{
				Protocol: "http",
				Host:     "fake-host",
				Port:     5555,
				Username: "fake-username",
				Password: "fake-password",
			},
		}

		factory Factory
	)

	var (
		operationService   operation.GoogleOperationService
		addressService     address.Service
		diskService        disk.Service
		diskTypeService    disktype.Service
		imageService       image.Service
		machineTypeService machinetype.Service
		networkService     network.Service
		snapshotService    snapshot.Service
		registryClient     registry.Client
		targetPoolService  targetpool.Service
		vmService          instance.Service
	)

	BeforeEach(func() {
		googleClient = clientfakes.NewFakeGoogleClient()
		uuidGen = &fakeuuid.FakeGenerator{}
		logger = boshlog.NewLogger(boshlog.LevelNone)

		factory = NewConcreteFactory(
			googleClient,
			uuidGen,
			options,
			logger,
		)
	})

	BeforeEach(func() {
		operationService = operation.NewGoogleOperationService(
			googleClient.Project(),
			googleClient.ComputeService(),
			logger,
		)

		addressService = address.NewGoogleAddressService(
			googleClient.Project(),
			googleClient.ComputeService(),
			logger,
		)

		diskService = disk.NewGoogleDiskService(
			googleClient.Project(),
			googleClient.ComputeService(),
			operationService,
			uuidGen,
			logger,
		)

		diskTypeService = disktype.NewGoogleDiskTypeService(
			googleClient.Project(),
			googleClient.ComputeService(),
			logger,
		)

		imageService = image.NewGoogleImageService(
			googleClient.Project(),
			googleClient.ComputeService(),
			googleClient.StorageService(),
			operationService,
			uuidGen,
			logger,
		)

		machineTypeService = machinetype.NewGoogleMachineTypeService(
			googleClient.Project(),
			googleClient.ComputeService(),
			logger,
		)

		networkService = network.NewGoogleNetworkService(
			googleClient.Project(),
			googleClient.ComputeService(),
			logger,
		)

		registryClient = registry.NewHTTPClient(
			options.Registry,
			logger,
		)

		snapshotService = snapshot.NewGoogleSnapshotService(
			googleClient.Project(),
			googleClient.ComputeService(),
			operationService,
			uuidGen,
			logger,
		)

		targetPoolService = targetpool.NewGoogleTargetPoolService(
			googleClient.Project(),
			googleClient.ComputeService(),
			operationService,
			logger,
		)

		vmService = instance.NewGoogleInstanceService(
			googleClient.Project(),
			googleClient.ComputeService(),
			addressService,
			networkService,
			operationService,
			targetPoolService,
			uuidGen,
			logger,
		)
	})

	It("returns error if action cannot be created", func() {
		action, err := factory.Create("fake-unknown-action")
		Expect(err).To(HaveOccurred())
		Expect(action).To(BeNil())
	})

	It("create_disk", func() {
		action, err := factory.Create("create_disk")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewCreateDisk(
			diskService,
			diskTypeService,
			vmService,
			googleClient.DefaultZone(),
		)))
	})

	It("delete_disk", func() {
		action, err := factory.Create("delete_disk")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewDeleteDisk(diskService)))
	})

	It("attach_disk", func() {
		action, err := factory.Create("attach_disk")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewAttachDisk(diskService, vmService, registryClient)))
	})

	It("detach_disk", func() {
		action, err := factory.Create("detach_disk")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewDetachDisk(vmService, registryClient)))
	})

	It("snapshot_disk", func() {
		action, err := factory.Create("snapshot_disk")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewSnapshotDisk(snapshotService, diskService)))
	})

	It("delete_snapshot", func() {
		action, err := factory.Create("delete_snapshot")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewDeleteSnapshot(snapshotService)))
	})

	It("create_stemcell", func() {
		action, err := factory.Create("create_stemcell")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewCreateStemcell(imageService)))
	})

	It("delete_stemcell", func() {
		action, err := factory.Create("delete_stemcell")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewDeleteStemcell(imageService)))
	})

	It("create_vm", func() {
		action, err := factory.Create("create_vm")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewCreateVM(
			vmService,
			diskService,
			diskTypeService,
			imageService,
			machineTypeService,
			registryClient,
			options.Registry,
			options.Agent,
			googleClient.DefaultRootDiskSizeGb(),
			googleClient.DefaultRootDiskType(),
			googleClient.DefaultZone(),
		)))
	})

	It("configure_networks", func() {
		action, err := factory.Create("configure_networks")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewConfigureNetworks(vmService, registryClient)))
	})

	It("delete_vm", func() {
		action, err := factory.Create("delete_vm")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewDeleteVM(vmService, registryClient)))
	})

	It("reboot_vm", func() {
		action, err := factory.Create("reboot_vm")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewRebootVM(vmService)))
	})

	It("set_vm_metadata", func() {
		action, err := factory.Create("set_vm_metadata")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewSetVMMetadata(vmService)))
	})

	It("has_vm", func() {
		action, err := factory.Create("has_vm")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewHasVM(vmService)))
	})

	It("get_disks", func() {
		action, err := factory.Create("get_disks")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewGetDisks(vmService)))
	})

	It("ping", func() {
		action, err := factory.Create("ping")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewPing()))
	})

	It("when action is current_vm_id returns an error because this CPI does not implement the method", func() {
		action, err := factory.Create("current_vm_id")
		Expect(err).To(HaveOccurred())
		Expect(action).To(BeNil())
	})

	It("when action is wrong returns an error because it is not an official CPI method", func() {
		action, err := factory.Create("wrong")
		Expect(err).To(HaveOccurred())
		Expect(action).To(BeNil())
	})
})

default: test

# Builds bosh-google-cpi for linux-amd64
build:
	go build -o out/cpi bosh-google-cpi/main

# Build cross-platform binaries
build-all:
	gox -output="out/cpi_{{.OS}}_{{.Arch}}" bosh-google-cpi/main

# Prepration for tests
get-deps:
	# Go lint tool
	go get github.com/golang/lint/golint

	# Simplify cross-compiling
	go get github.com/mitchellh/gox

	# Ginkgo and omega test tools
	go get github.com/onsi/ginkgo/ginkgo
	go get github.com/onsi/gomega

# Cleans up directory and source code with gofmt
clean:
	go clean ./...

# Run gofmt on all code
fmt:
	gofmt -l -w .

# Run linter with non-stric checking
lint:
	@echo ls -d */ | grep -v vendor | xargs -L 1 golint
	ls -d */ | grep -v vendor | xargs -L 1 golint

# Vet code
vet:
	go tool vet $$(ls -d */ | grep -v vendor)

# Runs the unit tests with coverage
test: get-deps clean fmt lint vet build
	ginkgo -r -race -skipPackage=integration .

# Runs the integration tests from Concourse
testintci: get-deps
	ginkgo integration -slowSpecThreshold=500 -progress -nodes=3 -randomizeAllSpecs -randomizeSuites $(GINKGO_ARGS) -v

# Runs the integration tests with coverage
testint: check-proj check-destroy get-deps clean fmt
	$(eval INTEGRATION_ADDRESS = $(shell gcloud compute addresses describe cfintegration --region=us-east1 | head  -n1 | cut -f2 -d' '))
	#GOOGLE_PROJECT=$(GOOGLE_PROJECT) EXTERNAL_STATIC_IP=$(INTEGRATION_ADDRESS) go test -v -timeout=60m ./integration/... -ginkgo.slowSpecThreshold=500 -ginkgo.progress -ginkgo.v -ginkgo.p -ginkgo.randomizeAllSpecs --ginkgo.randomizeSuites $(GINKGO_ARGS)
	GOOGLE_PROJECT=$(GOOGLE_PROJECT) EXTERNAL_STATIC_IP=$(INTEGRATION_ADDRESS) ginkgo integration -slowSpecThreshold=500 -progress -nodes=3 -randomizeAllSpecs -randomizeSuites $(GINKGO_ARGS) -v

# Checks and creates, if necessary, resources in a project required to run integration tests.
configint: check-proj
	@gcloud compute networks describe cfintegration >/dev/null 2>&1; if [ $$? -ne 0 ]; then \
		echo "Creating network cfintegration"; \
		gcloud --project=$(GOOGLE_PROJECT) compute networks create cfintegration; \
	fi

	@gcloud compute networks describe cfintegration-custom >/dev/null 2>&1; if [ $$? -ne 0 ]; then \
		echo "Creating network cfintegration-custom"; \
		gcloud --project=$(GOOGLE_PROJECT) compute networks create cfintegration-custom --mode=custom; \
	fi

	@gcloud compute networks subnets describe cfintegration-custom-us-east1 --region=us-east1 >/dev/null 2>&1; if [ $$? -ne 0 ]; then \
		echo "Creating subnetwork cfintegration-custom-us-east1"; \
		gcloud --project=$(GOOGLE_PROJECT) compute networks subnets create cfintegration-custom-us-east1 --network=cfintegration-custom --range=192.168.0.0/16 --region=us-east1; \
	fi

	@gcloud compute addresses describe cfintegration --region us-east1 >/dev/null 2>&1; if [ $$? -ne 0 ]; then \
		echo "Creating static IP cfintegration"; \
		gcloud --project=$(GOOGLE_PROJECT) compute addresses create cfintegration --region=us-east1; \
	fi

	@gcloud compute target-pools describe cfintegration --region us-east1 >/dev/null 2>&1; if [ $$? -ne 0 ]; then \
		echo "Creating target pool cfintegration"; \
		gcloud --project=$(GOOGLE_PROJECT) compute target-pools create cfintegration --region=us-east1; \
	fi

	@gcloud compute target-pools describe cfintegration2 --region us-east1 >/dev/null 2>&1; if [ $$? -ne 0 ]; then \
		echo "Creating target pool cfintegration"; \
		gcloud --project=$(GOOGLE_PROJECT) compute target-pools create cfintegration2 --region=us-east1; \
	fi

	@gcloud compute instance-groups unmanaged describe cfintegration --zone us-east1-d >/dev/null 2>&1; if [ $$? -ne 0 ]; then \
		echo "Creating instance group cfintegration in us-east1-d"; \
		gcloud --project=$(GOOGLE_PROJECT) compute instance-groups unmanaged create cfintegration --zone us-east1-d; \
	fi

	@gcloud compute instance-groups unmanaged describe cfintegration --zone us-east1-b >/dev/null 2>&1; if [ $$? -ne 0 ]; then \
		echo "Creating instance group cfintegration in us-east1-b"; \
		gcloud --project=$(GOOGLE_PROJECT) compute instance-groups unmanaged create cfintegration --zone us-east1-b; \
	fi

	@gcloud compute http-health-checks describe cfintegration >/dev/null 2>&1; if [ $$? -ne 0 ]; then \
		echo "Creating HTTP health check cfintegration"; \
		gcloud --project=$(GOOGLE_PROJECT) compute http-health-checks create cfintegration; \
	fi

	@gcloud compute backend-services describe cfintegration --global >/dev/null 2>&1; if [ $$? -ne 0 ]; then \
		echo "Creating backend service cfintegration"; \
		gcloud --project=$(GOOGLE_PROJECT) compute backend-services create "cfintegration" --http-health-checks "cfintegration" --global --port-name "http" --timeout "30"; \
		gcloud --project=$(GOOGLE_PROJECT) compute backend-services add-backend "cfintegration" --instance-group "cfintegration" --instance-group-zone "us-east1-d" --global --balancing-mode "UTILIZATION" --capacity-scaler "1" --max-utilization "0.8"; \
		gcloud --project=$(GOOGLE_PROJECT) compute backend-services add-backend "cfintegration" --instance-group "cfintegration" --instance-group-zone "us-east1-b" --global --balancing-mode "UTILIZATION" --capacity-scaler "1" --max-utilization "0.8"; \
	fi

	@echo "Your project is ready for integration tests."

# Deletes the resources created by the configint target
cleanint: check-proj
	@gcloud compute networks subnets describe cfintegration-custom-us-east1 --region=us-east1 >/dev/null 2>&1; if [ $$? -eq 0 ]; then \
		echo "Deleting subnetwork cfintegration-custom-us-east1"; \
		gcloud --project=$(GOOGLE_PROJECT) compute networks subnets delete cfintegration-custom-us-east1 --region=us-east1 --quiet; \
	fi

	@gcloud compute addresses describe cfintegration --region us-east1 >/dev/null 2>&1; if [ $$? -eq 0 ]; then \
		echo "Deleting static IP cfintegration"; \
		gcloud --project=$(GOOGLE_PROJECT) compute addresses delete cfintegration --region=us-east1 --quiet; \
	fi

	@gcloud compute target-pools describe cfintegration --region us-east1 >/dev/null 2>&1; if [ $$? -eq 0 ]; then \
		echo "Deleting target pool cfintegration"; \
		gcloud --project=$(GOOGLE_PROJECT) compute target-pools delete cfintegration --region=us-east1 --quiet; \
	fi

	@gcloud compute target-pools describe cfintegration2 --region us-east1 >/dev/null 2>&1; if [ $$? -eq 0 ]; then \
		echo "Deleting target pool cfintegration"; \
		gcloud --project=$(GOOGLE_PROJECT) compute target-pools delete cfintegration2 --region=us-east1 --quiet; \
	fi

	@gcloud compute backend-services describe cfintegration --global >/dev/null 2>&1; if [ $$? -eq 0 ]; then \
		echo "Deleting backend service cfintegration"; \
		gcloud --project=$(GOOGLE_PROJECT) compute backend-services delete "cfintegration" --global --quiet; \
	fi

	@gcloud compute http-health-checks describe cfintegration >/dev/null 2>&1; if [ $$? -eq 0 ]; then \
		echo "Deleting HTTP health check cfintegration"; \
		gcloud --project=$(GOOGLE_PROJECT) compute http-health-checks delete cfintegration --quiet; \
	fi

	@gcloud compute networks describe cfintegration >/dev/null 2>&1; if [ $$? -eq 0 ]; then \
		echo "Deleting network cfintegration"; \
		gcloud --project=$(GOOGLE_PROJECT) compute networks delete cfintegration --quiet; \
	fi

	@gcloud compute networks describe cfintegration-custom >/dev/null 2>&1; if [ $$? -eq 0 ]; then \
		echo "Deleting network cfintegration-custom"; \
		gcloud --project=$(GOOGLE_PROJECT) compute networks delete cfintegration-custom --quiet; \
	fi

	@gcloud compute instance-groups unmanaged describe cfintegration-us-east1-d --zone us-east1-d >/dev/null 2>&1; if [ $$? -eq 0 ]; then \
		echo "Deleting instance group cfintegration in us-east1-d"; \
		gcloud --project=$(GOOGLE_PROJECT) compute instance-groups unmanaged delete cfintegration --zone us-east1-d --quiet; \
	fi

	@gcloud compute instance-groups unmanaged describe cfintegration --zone us-east1-b >/dev/null 2>&1; if [ $$? -eq 0 ]; then \
		echo "Deleting instance group cfintegration in us-east1-b"; \
		gcloud --project=$(GOOGLE_PROJECT) compute instance-groups unmanaged delete cfintegration --zone us-east1-b --quiet; \
	fi

	@echo "All integration tests resources have been removed. You will need to run the configint target before doing integration tests again."

check-proj:
	@if test "$(GOOGLE_PROJECT)" = "" ; then \
	  echo "GOOGLE_PROJECT must be set"; \
	  exit 1; \
	fi

check-destroy:
	@if test "$(DESTROY_OK)" = "" ; then \
	  echo "This is a potentially destructive operation. Existing instances with the "integration-delete" tag will be deleted. You just set DESTROY_OK to acknowledge."; \
	  exit 1; \
	fi

docker-build:
	docker build -t mitchya1/ecs-config-retriever:$(VERSION) .

docker-push:
	docker push mitchya1/ecs-config-retriever:$(VERSION)

unit-tests:
	go test -v ./cmd/retriever/
	rm -rf /tmp/ci-*
	go test -v ./pkg/retriever/

integration-tests:
	bash tests/run.sh

cleanup:
	rm /tmp/param-*
	rm /tmp/binary-param-*

docker-cleanup:
	docker rmi --force mitchya1/ecs-config-retriever:$(VERSION)

docker-tests:
	@echo "ACCESS_KEY=${ACCESS_KEY}" > .env
	@echo "SECRET_KEY=${SECRET_KEY}" >> .env
	docker-compose -f tests/docker-compose.yaml up | tee /tmp/ci-compose-out
	grep "with code 1" /tmp/ci-compose-out && exit 1 || exit 0
	# Not sure if this test actually tests permissions correctly
	# The goal is to ensure that retriever can write to a volume mounted as a subdir of /init-out/
	docker-compose -f tests/docker-compose-multi-volume.yaml up | tee /tmp/ci-compose-out
	grep "with code 1" /tmp/ci-compose-out && exit 1 || exit 0
	

docker-vault-tests:
	docker-compose -f tests/docker-compose-vault.yaml up | tee /tmp/ci-vault-compose-out &
	# The test only takes a few seconds to run
	sleep 10
	docker stop vault
	grep "with code 1" /tmp/ci-vault-compose-out && exit 1 || exit 0
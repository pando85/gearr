transcoderd-cli:
	go run  build.go build worker -m console

transcoderd-docker:
	docker build -t transcoderd -f server/Dockerfile .

transcoderd-gui:

transcoderd: transcoderd-cli transcoderd-docker transcoderd-gui

transcoderw-cli:
	go run  build.go build worker -m console

transcoderw-docker:
	docker build -t transcoderw -f worker/Dockerfile .

transcoderw-gui:

transcoderw: transcoderw-cli transcoderw-docker transcoderw-gui

build: transcoderd transcoderw


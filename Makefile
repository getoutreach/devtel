APP := devtel
OSS := true
_ := $(shell ./scripts/devbase.sh) 

include .bootstrap/root/Makefile

###Block(targets)
pre-release::
	./scripts/update-plugin-version.sh $(APP_VERSION)
###EndBlock(targets)

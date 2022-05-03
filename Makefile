APP := devtel
OSS := false
_ := $(shell ./scripts/devbase.sh) 

include .bootstrap/root/Makefile

###Block(targets)
pre-release::
	./scripts/update-plugin-version.sh $(APP_VERSION)
###EndBlock(targets)

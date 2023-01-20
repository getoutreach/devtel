APP := devtel
OSS := true
_ := $(shell ./scripts/devbase.sh) 

include .bootstrap/root/Makefile

## <<Stencil::Block(targets)>>
pre-release::
	./scripts/update-plugin-version.sh $(APP_VERSION)
## <</Stencil::Block>>

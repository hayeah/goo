.PHONY: wire run dev

wire:
	go run github.com/google/wire/cmd/wire .

# make run ARGS="somefile.go --flag1 --flag2"
run:
	CONFIG_FILE=cfg.toml go run ./cmd $(ARGS)

dev:
	CONFIG_FILE=cfg.toml go run github.com/cortesi/modd/cmd/modd@latest

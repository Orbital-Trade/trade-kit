TOOLS := tiger moomoo etoro alpaca sidecar scheduler daytrader earnings bounce controller index notifier alert journal options backtest

.PHONY: all clean test $(TOOLS)

all: $(TOOLS)

tiger:
	cd tiger     && go build -o tiger-cli    ./cmd/

moomoo:
	cd moomoo    && go build -o moomoo-cli   ./cmd/

etoro:
	cd etoro     && go build -o etoro-cli    ./cmd/

alpaca:
	cd alpaca    && GOWORK=off go build -o alpaca-cli   ./cmd/

sidecar:
	cd sidecar   && go build -ldflags "-X main.Version=$$(cat ../VERSION)" -o trade-kit ./cmd/

scheduler:
	cd scheduler && GOWORK=off go build -o scheduler    ./cmd/

daytrader:
	cd daytrader && GOWORK=off go build -o daytrader-bot ./cmd/

earnings:
	cd earnings  && GOWORK=off go build -o earnings-bot ./cmd/

bounce:
	cd bounce    && GOWORK=off go build -o bounce-bot   ./cmd/

controller:
	cd controller && GOWORK=off go build -o controller  ./cmd/

index:
	cd index     && GOWORK=off go build -o index-trader ./cmd/

notifier:
	cd notifier  && GOWORK=off go build -o notifier     ./cmd/

alert:
	cd alert     && GOWORK=off go build -o alert        ./cmd/

journal:
	cd journal   && GOWORK=off go build -o journal      ./cmd/

options:
	cd options   && GOWORK=off go build -o options      ./cmd/

backtest:
	cd backtest  && GOWORK=off go build -o backtest     ./cmd/

test:
	cd tiger && go test ./...
	cd etoro && go test ./...

clean:
	rm -f tiger/tiger-cli
	rm -f moomoo/moomoo-cli
	rm -f scheduler/scheduler
	rm -f daytrader/daytrader-bot
	rm -f earnings/earnings-bot
	rm -f bounce/bounce-bot
	rm -f index/index-trader
	rm -f notifier/notifier
	rm -f alert/alert
	rm -f journal/journal
	rm -f options/options
	rm -f etoro/etoro-cli
	rm -f sidecar/trade-kit
	rm -f alpaca/alpaca-cli
	rm -f controller/controller
	rm -f backtest/backtest

TOOLS := tiger moomoo scheduler daytrader earnings bounce index notifier alert journal options

.PHONY: all clean test $(TOOLS)

all: $(TOOLS)

tiger:
	cd tiger     && go build -o tiger-cli    ./cmd/

moomoo:
	cd moomoo    && go build -o moomoo-cli   ./cmd/

scheduler:
	cd scheduler && go build -o scheduler    ./cmd/

daytrader:
	cd daytrader && go build -o daytrader-bot ./cmd/

earnings:
	cd earnings  && go build -o earnings-bot ./cmd/

bounce:
	cd bounce    && go build -o bounce-bot   ./cmd/

index:
	cd index     && go build -o index-trader ./cmd/

notifier:
	cd notifier  && go build -o notifier     ./cmd/

alert:
	cd alert     && go build -o alert        ./cmd/

journal:
	cd journal   && go build -o journal      ./cmd/

options:
	cd options   && go build -o options      ./cmd/

test:
	cd tiger && go test ./...

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

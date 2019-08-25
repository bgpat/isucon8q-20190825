.PHONY: deploy
deploy:
	systemctl stop torb.go
	systemctl stop mariadb
	go build -v ./src/torb
	rm /var/lib/mysql/slow.log
	systemctl start mariadb
	systemctl start torb.go

result: log/kataribe.log
	mysqldumpslow -s t /var/lib/mysql/slow.log > /home/isucon/torb/webapp/go/log/slow.log
	dd if=/home/isucon/torb/webapp/go/log/slow.log bs=1024 count=1 | slackcat --channel slowlog

log/kataribe.log: /var/log/h2o/access.log
	./kataribe -conf kataribe.toml < $< > $@
	slackcat --channel kataribe < $@
	rm -f $<
	systemctl restart h2o

.PHONY: clean
clean:
	rm -rf torb

deps:
	gb vendor restore

.PHONY: build
build:
	GOPATH=`pwd`:`pwd`/vendor go build -v torb

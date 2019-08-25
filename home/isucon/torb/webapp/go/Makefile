.PHONY: deploy
deploy:
	systemctl stop torb.go
	systemctl stop mariadb
	go build -v ./src/torb
	rm /var/lib/mysql/slow.log
	systemctl start mariadb
	systemctl start torb.go

result:
	mysqldumpslow -s t /var/lib/mysql/slow.log > /home/isucon/torb/webapp/go/log/slow.log
	dd if=/home/isucon/torb/webapp/go/log/slow.log bs=1024 count=1 | slackcat --channel slowlog

.PHONY: clean
clean:
	rm -rf torb

deps:
	gb vendor restore

.PHONY: build
build:
	GOPATH=`pwd`:`pwd`/vendor go build -v torb

PREFIX=/usr/local

build: src/* src/*/*
	cd src && go build
	mv src/IB1 .

install:
	cp IB1 ${PREFIX}/bin/
	chmod 755 ${PREFIX}/bin/IB1

uninstall:
	rm ${PREFIX}/bin/IB1

clean:
	rm -f IB1

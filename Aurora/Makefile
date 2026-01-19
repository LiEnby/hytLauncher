out=Build
binary=$(out)/Aurora.so

all: setup $(binary)

$(binary): linux.c
	gcc $< -o $@ -Os -fPIC -nostdlib -fshort-wchar -shared


.PHONY: setup
setup:
	mkdir -p $(out)

.PHONY: clean
clean:
	rm -f $(binary) *.o

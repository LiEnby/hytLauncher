CC=gcc
CFLAGS=-Os -fPIC -nostdlib -fshort-wchar -shared
OBJ = cs_string.o shared.o patch.o
OUTDIR=Build
BINARY=$(OUTDIR)/Aurora.so

all: setup $(BINARY)


%.o: %.c $(DEPS)
	$(CC) -c -o $@ $< $(CFLAGS)

$(BINARY): $(OBJ)
	$(CC) -o $@ $^ $(CFLAGS)



.PHONY: setup
setup:
	mkdir -p $(OUTDIR)

.PHONY: clean
clean:
	rm -f $(BINARY) *.o

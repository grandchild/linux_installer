
SRC = *.go \
	main/*.go
PKG = github.com/grandchild/linux_installer

RES = \
	resources/

BIN = linux_installer
DATA_SRC_DIR = data
DATA_DIST_DIR = data_compressed

ZIP = zip

WIN_DIST_DIR = win
XCC_GOFLAGS = \
	CGO_LDFLAGS_ALLOW="-Wl,-luuid"\
	PKG_CONFIG_PATH=/usr/x86_64-w64-mingw32/lib/pkgconfig \
	CGO_ENABLED=1 \
	CC=x86_64-w64-mingw32-cc \
	GOOS=windows \
	GOARCH=amd64
XCC_LD_FLAGS = -ldflags -H=windowsgui
WIN_DLL_SRC = /usr/x86_64-w64-mingw32/bin
WIN_DLLS = \
	libatk-1.0-0.dll \
	libbz2-1.dll \
	libcairo-2.dll \
	libcairo-gobject-2.dll \
	libepoxy-0.dll \
	libexpat-1.dll \
	libffi-6.dll \
	libfontconfig-1.dll \
	libfreetype-6.dll \
	libfribidi-0.dll \
	libgcc_s_seh-1.dll \
	libgdk-3-0.dll \
	libgdk_pixbuf-2.0-0.dll \
	libgio-2.0-0.dll \
	libglib-2.0-0.dll \
	libgmodule-2.0-0.dll \
	libgobject-2.0-0.dll \
	libgraphite2.dll \
	libgtk-3-0.dll \
	libharfbuzz-0.dll \
	libiconv-2.dll \
	libintl-8.dll \
	libjasper-4.dll \
	libjpeg-8.dll \
	liblzma-5.dll \
	libpango-1.0-0.dll \
	libpangocairo-1.0-0.dll \
	libpangoft2-1.0-0.dll \
	libpangowin32-1.0-0.dll \
	libpcre-1.dll \
	libpixman-1-0.dll \
	libpng16-16.dll \
	libstdc++-6.dll \
	libtiff-5.dll \
	libwinpthread-1.dll \
	zlib1.dll \


default: linux

all: linux windows dist

linux: linux_build $(DATA_DIST_DIR)/data.zip

windows: windows_build $(DATA_DIST_DIR)/data.zip


dist: linux_dist

linux_build: $(SRC)
	go build -o $(BIN) $(PKG)/main

$(DATA_DIST_DIR)/data.zip: $(DATA_SRC_DIR)
	mkdir -p $(DATA_DIST_DIR)
	cd $(DATA_SRC_DIR) ; $(ZIP) -r ../$(DATA_DIST_DIR)/data.zip .

linux_dist: linux_build $(DATA_DIST_DIR)/data.zip
	$(GOPATH)/bin/rice append --exec $(BIN)

run: linux_dist
	./$(BIN)


windows_build: $(SRC)
	$(XCC_GOFLAGS) go build $(XCC_LD_FLAGS) -o $(WIN_DIST_DIR)/$(BIN).exe $(PKG)/main

windows_dist: windows_build $(DATA_DIST_DIR)/data.zip
	# cp -r $(RES) $(WIN_DIST_DIR)
	mkdir -p $(WIN_DIST_DIR)
	cp $(foreach dll,$(WIN_DLLS),$(WIN_DLL_SRC)/$(dll)) $(WIN_DIST_DIR)
	$(GOPATH)/bin/rice append --exec $(WIN_DIST_DIR)/$(BIN).exe

run_win: windows_dist
	wine $(WIN_DIST_DIR)/$(BIN).exe


clean: windows_clean linux_clean

windows_clean: clean_data
	rm -rf $(WIN_DIST_DIR)

linux_clean: clean_data
	rm -f $(BIN)

clean_data:
	rm -rf $(DATA_DIST_DIR)

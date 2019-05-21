
SRC = *.go \
	gui/*.go \
	main/*.go
PKG = github.com/grandchild/linux_installer

BIN = linux_installer
RES_DIR = resources
DATA_SRC_DIR = data
DATA_DIST_DIR = data_compressed
BUILDER_DIR = linux-builder
BUILDER_ARCHIVE = $(BUILDER_DIR).zip

ZIP_EXE = zip
RICE_EXE = rice

GOPATH ?= ~/go

GO_MOD_FLAGS = -mod=vendor

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

linux_builder: linux_clean $(BUILDER_ARCHIVE)

windows: windows_build $(DATA_DIST_DIR)/data.zip


dist: linux_dist

linux_build: $(SRC) $(RES_DIR)/gui/gui.so
	go build -v $(GO_MOD_FLAGS) -o $(BIN) $(PKG)/main

$(RES_DIR)/gui/gui.so: $(SRC)
	go build -v $(GO_MOD_FLAGS) -buildmode=plugin -o $(RES_DIR)/gui/gui.so $(PKG)/gui

$(DATA_DIST_DIR)/data.zip: $(DATA_SRC_DIR)
	mkdir -p $(DATA_DIST_DIR)
	rm -f $(DATA_DIST_DIR)/data.zip
	cd $(DATA_SRC_DIR) ; $(ZIP_EXE) -r ../$(DATA_DIST_DIR)/data.zip .

linux_dist: linux_build $(DATA_DIST_DIR)/data.zip
	rice_bin/rice append --exec $(BIN)

run: linux_dist
	./$(BIN)

$(BUILDER_DIR): linux_build $(DATA_SRC_DIR)
	cp -r $(DATA_SRC_DIR) $(RES_DIR) $(BIN) rice_bin/$(RICE_EXE)* $(BUILDER_DIR)/
	chmod +x $(BUILDER_DIR)/$(RICE_EXE)

$(BUILDER_ARCHIVE): $(BUILDER_DIR)
	chmod -R g+w $(BUILDER_DIR)
	zip -r $(BUILDER_ARCHIVE) $(BUILDER_DIR)


windows_build: $(SRC)
	$(XCC_GOFLAGS) go build -v $(GO_MOD_FLAGS) $(XCC_LD_FLAGS) -o $(WIN_DIST_DIR)/$(BIN).exe $(PKG)/main

windows_dist: windows_build $(DATA_DIST_DIR)/data.zip
	# cp -r $(RES_DIR) $(WIN_DIST_DIR)
	mkdir -p $(WIN_DIST_DIR)
	cp $(foreach dll,$(WIN_DLLS),$(WIN_DLL_SRC)/$(dll)) $(WIN_DIST_DIR)
	rice_bin/rice.exe append --exec $(WIN_DIST_DIR)/$(BIN).exe

run_win: windows_dist
	wine $(WIN_DIST_DIR)/$(BIN).exe


$(DATA_SRC_DIR):
	mkdir $@

clean: windows_clean linux_clean

windows_clean: clean_data clean_builder
	rm -rf $(WIN_DIST_DIR)

linux_clean: clean_data clean_builder
	rm -f $(RES_DIR)/gui/gui.so
	rm -f $(BIN)

clean_data:
	rm -rf $(DATA_DIST_DIR)

clean_builder:
	rm -rf $(BUILDER_DIR)/{$(RES_DIR),$(DATA_DIST_DIR),$(DATA_SRC_DIR),$(BIN),$(RICE_EXE)}
	rm -f $(BUILDER_ARCHIVE)


# To update rice to the newest version run this target ("make rice_bin").
#
# The GOBIN-trick doesn't work for cross-compilation, so that one is created in
# $GOPATH/bin as usual and then copied.
rice_bin: .FORCE
	go get github.com/GeertJohan/go.rice
	GOBIN=`readlink -f rice_bin` go install github.com/GeertJohan/go.rice/rice
	GOOS=windows go install github.com/GeertJohan/go.rice/rice
	cp $(GOPATH)/bin/windows_amd64/rice.exe rice_bin/

.FORCE: # targets with this requirement are always out of date

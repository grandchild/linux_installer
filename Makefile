
SRC = $(wildcard *.go gui/*.go main/*.go)
PKG = github.com/grandchild/linux_installer

BIN = linux_installer
BIN_DEV = linux_installer_dev
RES_DIR = resources
DATA_SRC_DIR = data
DATA_DIST_DIR = data_compressed
BUILDER_DIR = linux-builder
BUILDER_ARCHIVE = $(BUILDER_DIR).zip

ZIP_EXE = zip
RICE_EXE = rice

GOPATH ?= ~/go

GO_MOD_FLAGS = -mod=vendor


default: build $(DATA_DIST_DIR)/data.zip

builder: clean $(BUILDER_ARCHIVE)


build: $(SRC) $(RES_DIR)/gui/gui.so
	go build -v $(GO_MOD_FLAGS) -o "$(BIN)" "$(PKG)/main"

$(RES_DIR)/gui/gui.so: $(SRC)
	go build -v $(GO_MOD_FLAGS) -buildmode=plugin -o "$(RES_DIR)/gui/gui.so" "$(PKG)/gui"

$(DATA_DIST_DIR)/data.zip: $(DATA_SRC_DIR)
	mkdir -p "$(DATA_DIST_DIR)"
	rm -f "$(DATA_DIST_DIR)/data.zip"
	cd "$(DATA_SRC_DIR)" ; $(ZIP_EXE) -r "../$(DATA_DIST_DIR)/data.zip" .

dist: build $(DATA_DIST_DIR)/data.zip rice_bin
	cp "$(BIN)" "$(BIN_DEV)"
	rice_bin/rice append --exec "$(BIN_DEV)"

run: dist
	./"$(BIN_DEV)"

$(BUILDER_DIR): build $(DATA_SRC_DIR) rice_bin
	cp -r "$(DATA_SRC_DIR)" "$(RES_DIR)" "$(BIN)" rice_bin/$(RICE_EXE)* "$(BUILDER_DIR)/"
	chmod +x "$(BUILDER_DIR)/$(RICE_EXE)"

$(BUILDER_ARCHIVE): $(BUILDER_DIR)
	chmod -R g+w "$(BUILDER_DIR)"
	zip -r "$(BUILDER_ARCHIVE)" "$(BUILDER_DIR)"


$(DATA_SRC_DIR):
	mkdir "$@"

clean: clean_data clean_builder
	rm -f "$(RES_DIR)/gui/gui.so"
	rm -f "$(BIN)"

clean_data:
	rm -rf "$(DATA_DIST_DIR)"

clean_builder:
	rm -rf "$(BUILDER_DIR)/"{"$(RES_DIR)","$(DATA_DIST_DIR)","$(DATA_SRC_DIR)","$(BIN)","$(RICE_EXE)"}
	rm -f "$(BUILDER_ARCHIVE)"


rice_bin:
	mkdir -p rice_bin
	go get github.com/GeertJohan/go.rice
	GOBIN=`readlink -f rice_bin` go install github.com/GeertJohan/go.rice/rice
	# The $GOBIN-trick doesn't work for cross-compilation, so that one is created
	# in $GOPATH/bin as usual and then copied.
	GOOS=windows go install github.com/GeertJohan/go.rice/rice
	cp "$(GOPATH)/bin/windows_amd64/rice.exe" rice_bin/

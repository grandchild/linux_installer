
SRC = $(wildcard *.go gui/*.go main/*.go)
PKG = github.com/grandchild/linux_installer

BIN = linux-installer
BIN_DEV = linux-installer-dev
RES_DIR = resources
DATA_SRC_DIR = data
DATA_DIST_DIR = data-compressed
BUILDER_DIR = linux-builder
BUILDER_ARCHIVE = $(BUILDER_DIR).zip
RICE_BIN_DIR = rice-bin

ZIP_EXE = zip
RICE_EXE = rice

GOPATH ?= ~/go

GO_MOD_FLAGS = -mod=vendor


default: build $(DATA_DIST_DIR)/data.zip
builder: $(BUILDER_ARCHIVE)


build: $(SRC)
	go build -v $(GO_MOD_FLAGS) -o "$(BIN)" "$(PKG)/main"

$(RES_DIR)/gui/gui.so: $(SRC)
	go build -v $(GO_MOD_FLAGS) -buildmode=plugin -o "$(RES_DIR)/gui/gui.so" "$(PKG)/gui"

$(DATA_DIST_DIR)/data.zip: $(DATA_SRC_DIR)
	mkdir -p "$(DATA_DIST_DIR)"
	rm -f "$(DATA_DIST_DIR)/data.zip"
	cd "$(DATA_SRC_DIR)" ; "$(ZIP_EXE)" -r "../$(DATA_DIST_DIR)/data.zip" .

dev: build $(RES_DIR)/gui/gui.so $(DATA_DIST_DIR)/data.zip $(RICE_BIN_DIR)
	cp "$(BIN)" "$(BIN_DEV)"
	$(RICE_BIN_DIR)/rice append --exec "$(BIN_DEV)"

run: dev
	./"$(BIN_DEV)"

runcli: dev
	./"$(BIN_DEV)" -target ./DevInstallation -accept

$(BUILDER_DIR): build $(RES_DIR)/gui/gui.so $(DATA_SRC_DIR) $(RICE_BIN_DIR)
	cp -r "$(DATA_SRC_DIR)" "$(RES_DIR)" "$(BIN)" "$(RICE_BIN_DIR)/$(RICE_EXE)"* "$(BUILDER_DIR)/"
	chmod +x "$(BUILDER_DIR)/$(RICE_EXE)"

$(BUILDER_ARCHIVE): $(BUILDER_DIR)
	chmod -R g+w "$(BUILDER_DIR)"
	"$(ZIP_EXE)" -r "$(BUILDER_ARCHIVE)" "$(BUILDER_DIR)"


$(DATA_SRC_DIR):
	mkdir "$@"

clean: clean-data clean-builder
	rm -f "$(RES_DIR)/gui/gui.so"
	rm -f "$(BIN)" "$(BIN_DEV)"

clean-data:
	rm -rf "$(DATA_DIST_DIR)"

clean-builder:
	rm -rf "$(BUILDER_DIR)/"{"$(RES_DIR)","$(DATA_DIST_DIR)","$(DATA_SRC_DIR)","$(BIN)","$(RICE_EXE)"}
	rm -f "$(BUILDER_ARCHIVE)"


$(RICE_BIN_DIR):
	mkdir -p "$(RICE_BIN_DIR)"
	go get github.com/GeertJohan/go.rice
	GOBIN=`readlink -f "$(RICE_BIN_DIR)"` go install github.com/GeertJohan/go.rice/rice
	# The GOBIN-trick doesn't work for cross-compilation, so that one is created
	# in GOPATH/bin as usual and then copied.
	GOOS=windows go install github.com/GeertJohan/go.rice/rice
	cp "$(GOPATH)/bin/windows_amd64/rice.exe" $(RICE_BIN_DIR)/
	# This last cp fails on some systems (filesystems?) when this target is called for
	# the first time. Happened on live-iso-systems.


SRC = $(wildcard *.go main/*.go)
SRC_GUI = $(wildcard gui/*.go)
PKG = github.com/grandchild/linux_installer

BIN = linux-installer
BIN_DEV = linux-installer-dev
RES_DIR = resources
DATA_SRC_DIR = data
DATA_DIST_DIR = data-compressed
RELEASE_DIST_DIR = .release
RELEASE_BIN = setup-installer-builder
BUILDER_DIR = linux-builder
BUILDER_ARCHIVE = $(BUILDER_DIR).zip
RICE_BIN_DIR = rice-bin

ZIP_EXE = zip
RICE_EXE = rice

GOPATH ?= $(HOME)/go

GO_MOD_FLAGS = -mod=vendor

GTK_VERSION ?= 3.12
GOTK3_BUILD_TAGS = $(if $(GTK_VERSION),-tags gtk_$(subst .,_,$(GTK_VERSION)))


default: build $(DATA_DIST_DIR)/data.zip
build: installer $(RES_DIR)/gui/gui.so
builder: $(BUILDER_ARCHIVE)


installer: $(SRC)
	go build -v $(GO_MOD_FLAGS) -o "$(BIN)" "$(PKG)/main"

$(RES_DIR)/gui/gui.so: $(SRC_GUI)
	go build -v $(GO_MOD_FLAGS) $(GOTK3_BUILD_TAGS) -buildmode=plugin \
		-o "$(RES_DIR)/gui/gui.so" "$(PKG)/gui"

$(DATA_DIST_DIR)/data.zip: $(DATA_SRC_DIR)
	mkdir -p "$(DATA_DIST_DIR)"
	rm -f "$(DATA_DIST_DIR)/data.zip"
	cd "$(DATA_SRC_DIR)" ; "$(ZIP_EXE)" -r "../$(DATA_DIST_DIR)/data.zip" .

dev: build $(DATA_DIST_DIR)/data.zip $(RICE_BIN_DIR)
	cp "$(BIN)" "$(BIN_DEV)"
	$(RICE_BIN_DIR)/rice append --exec "$(BIN_DEV)"

run: dev
	./"$(BIN_DEV)"

runcli: dev
	./"$(BIN_DEV)" -target ./DevInstallation -accept

$(BUILDER_DIR): build $(DATA_SRC_DIR) $(RICE_BIN_DIR)
	cp -r "$(DATA_SRC_DIR)" "$(RES_DIR)" "$(BIN)" "$(RICE_BIN_DIR)/$(RICE_EXE)"* \
		"$(BUILDER_DIR)/"
	chmod +x "$(BUILDER_DIR)/$(RICE_EXE)"

$(BUILDER_ARCHIVE): $(BUILDER_DIR)
	chmod -R g+w "$(BUILDER_DIR)"
	"$(ZIP_EXE)" -r "$(BUILDER_ARCHIVE)" "$(BUILDER_DIR)"

self-installer: clean-builder $(BUILDER_DIR)
	cp "$(BIN)" "$(RELEASE_DIST_DIR)/$(RELEASE_BIN)"
	mkdir -p "$(RELEASE_DIST_DIR)/$(DATA_DIST_DIR)/"
	cd "$(BUILDER_DIR)" ; \
		"$(ZIP_EXE)" -r "../$(RELEASE_DIST_DIR)/$(DATA_DIST_DIR)/$(BUILDER_ARCHIVE)" *
	cp "$(RES_DIR)/gui/gui.so" "$(RES_DIR)/gui/gui.glade" \
		"$(RELEASE_DIST_DIR)/$(RES_DIR)/gui/"
	cp -r "$(RES_DIR)/languages" "$(RES_DIR)/uninstaller" \
		"$(RELEASE_DIST_DIR)/$(RES_DIR)/"
	cp "$(BUILDER_DIR)/rice.go" "$(RELEASE_DIST_DIR)/rice.go"
	cd "$(RELEASE_DIST_DIR)" ; \
		"../$(RICE_BIN_DIR)/rice" append --exec "$(RELEASE_BIN)"

$(DATA_SRC_DIR):
	mkdir "$@"

clean: clean-data clean-builder clean-self-installer clean-rice
	rm -f "$(RES_DIR)/gui/gui.so"
	rm -f "$(BIN)" "$(BIN_DEV)"

clean-data:
	rm -rf "$(DATA_DIST_DIR)"

clean-builder:
	rm -rf \
		"$(BUILDER_DIR)/$(RES_DIR)" \
		"$(BUILDER_DIR)/$(DATA_DIST_DIR)" \
		"$(BUILDER_DIR)/$(DATA_SRC_DIR)" \
		"$(BUILDER_DIR)/$(BIN)" \
		"$(BUILDER_DIR)/$(RICE_EXE)" \
		"$(BUILDER_ARCHIVE)"

clean-self-installer:
	rm -rf \
		"$(RELEASE_DIST_DIR)/$(DATA_DIST_DIR)" \
		"$(RELEASE_DIST_DIR)/$(RES_DIR)/gui/gui.so" \
		"$(RELEASE_DIST_DIR)/$(RES_DIR)/gui/gui.glade" \
		"$(RELEASE_DIST_DIR)/$(RES_DIR)/languages" \
		"$(RELEASE_DIST_DIR)/$(RES_DIR)/uninstaller" \

clean-rice:
	rm -rf "$(RICE_BIN_DIR)"


$(RICE_BIN_DIR):
	mkdir -p "$(RICE_BIN_DIR)"
	GOBIN=`readlink -f "$(RICE_BIN_DIR)"` go install github.com/GeertJohan/go.rice/rice
	# The GOBIN-trick doesn't work for cross-compilation, so that one is created
	# in GOPATH/bin as usual and then copied.
	GOOS=windows go install github.com/GeertJohan/go.rice/rice
	cp "$(GOPATH)/bin/windows_amd64/rice.exe" $(RICE_BIN_DIR)/

# This is the little dance that's required to vendor a binary tool, which usually aren't
# importable go modules. Enable the `vendor_rice_cmd.go` file which _does_ import
# go.rice/rice. This is required for `go mod` to include it in the dependencies to
# vendor when recreating the vendor directory. The file is restored to its ".disabled"
# name afterwards because "go build" will fail with that import present.
vendor-refresh:
	mv vendor_rice_cmd.go{.disabled,}
	rm -rf \
		vendor \
		go.mod \
		go.sum
	go mod init $(PKG)
	go mod tidy
	go mod vendor
	mv vendor_rice_cmd.go{,.disabled}

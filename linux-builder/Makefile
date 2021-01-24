# HOWTO:
# To package an installer, fill the "data" folder with the complete set of files, add
# other static zip files (if any) into "data-compressed", and finally run
#
#   make VERSION=1.1 OUTPUT=Setup_ExampleApp_v1.1
#
# Set VERSION to the current version number (this will appear in the installer GUI) and
# OUTPUT to the full name of the desired installer.


# The final installer filename. You can override this when calling make like this:
#   make OUTPUT=Setup_ExampleApp_v1.1
# or something similar.
OUTPUT=Setup
# The raw installer's filename (created by the parent linux-installer project)
INPUT=linux-installer

# Always set version explicitly! This is supposed to be a noticably wrong default.
VERSION=0.0

# The name of the folder containing all installer data.
DATA_SRC_DIR = data
# The name of the folder where resources are located. If you change this, you have to
# change it in rice.go as well.
RESOURCE_SRC_DIR = resources
# The name of the folder containing the compressed data. Files from data will be
# compressed into a data.zip file inside this folder, but you can add other pre-zipped
# data files as well, so save on compression time for static files.
# If you change this you have to change it in rice.go as well.
DATA_DIST_DIR = data-compressed

# Name of the "zip" command (might be different on some systems)
ZIP_EXE = zip

# Running just `make` depends on (i.e. creates) the OUTPUT file.
default: $(OUTPUT)

version: $(RESOURCE_SRC_DIR)/config.yml
	sed -i 's/^  version: .\+$$/  version: $(VERSION)/' $(RESOURCE_SRC_DIR)/config.yml

# Pre-compress the source data
$(DATA_DIST_DIR)/data.zip: $(DATA_SRC_DIR)
	mkdir -p $(DATA_DIST_DIR)
	cd $(DATA_SRC_DIR) ; $(ZIP_EXE) -r ../$(DATA_DIST_DIR)/data.zip .

# Make a copy of the raw installer & append the zipped data and resources to it.
$(OUTPUT): $(INPUT) version clean $(DATA_DIST_DIR)/data.zip
	cp $(INPUT) $(OUTPUT)
	./rice append --exec=$(OUTPUT)

# Remove previously created installer.
clean:
	rm -f $(DATA_DIST_DIR)/data.zip
	rm -f $(OUTPUT)
